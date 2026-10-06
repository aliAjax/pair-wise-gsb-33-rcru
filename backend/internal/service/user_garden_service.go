package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// UserGardenService implements "my garden" list logic.
type UserGardenService struct {
	repo         *repository.UserGardenRepository
	reminderRepo *repository.CareReminderRepository
	plantRepo    *repository.PlantSpeciesRepository
	db           *gorm.DB
	logger       *slog.Logger
}

// NewUserGardenService creates a UserGardenService.
func NewUserGardenService(
	repo *repository.UserGardenRepository,
	reminderRepo *repository.CareReminderRepository,
	plantRepo *repository.PlantSpeciesRepository,
	db *gorm.DB,
	logger *slog.Logger,
) *UserGardenService {
	return &UserGardenService{
		repo:         repo,
		reminderRepo: reminderRepo,
		plantRepo:    plantRepo,
		db:           db,
		logger:       logger,
	}
}

// Add adds a plant to a user's garden. Several pots of the same species are
// allowed, so duplicate species in active pots is no longer an error.
func (s *UserGardenService) Add(userID uint, g *model.UserGarden) (*dto.GardenView, error) {
	g.UserID = userID
	g.Status = model.GardenStatusActive
	if g.OwnedSince.IsZero() {
		g.OwnedSince = time.Now()
	}
	if err := s.repo.Create(g); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogGardenAddFailed, g.PlantSpeciesID, userID), "error", err)
		return nil, fmt.Errorf("user garden add: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogGardenAddSuccess, g.PlantSpeciesID, userID), "id", g.ID)
	return s.viewOf(g)
}

// List returns a user's active garden items enriched with plant source info.
func (s *UserGardenService) List(userID uint) ([]dto.GardenView, error) {
	items, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("user garden list: %w", err)
	}
	return s.viewsOf(items)
}

func (s *UserGardenService) viewsOf(items []model.UserGarden) ([]dto.GardenView, error) {
	if len(items) == 0 {
		return []dto.GardenView{}, nil
	}
	ids := make([]uint, 0, len(items))
	seen := map[uint]struct{}{}
	for _, g := range items {
		if _, ok := seen[g.PlantSpeciesID]; ok {
			continue
		}
		seen[g.PlantSpeciesID] = struct{}{}
		ids = append(ids, g.PlantSpeciesID)
	}
	plants, err := s.plantRepo.FindByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("user garden plants: %w", err)
	}
	plantMap := map[uint]model.PlantSpecies{}
	for _, p := range plants {
		plantMap[p.ID] = p
	}
	out := make([]dto.GardenView, 0, len(items))
	for _, g := range items {
		v := gardenToView(g, plantMap[g.PlantSpeciesID])
		out = append(out, v)
	}
	return out, nil
}

func (s *UserGardenService) viewOf(g *model.UserGarden) (*dto.GardenView, error) {
	plant, err := s.plantRepo.FindByID(g.PlantSpeciesID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			v := gardenToView(*g, model.PlantSpecies{})
			return &v, nil
		}
		return nil, fmt.Errorf("user garden plant: %w", err)
	}
	v := gardenToView(*g, *plant)
	return &v, nil
}

func gardenToView(g model.UserGarden, p model.PlantSpecies) dto.GardenView {
	v := dto.GardenView{
		ID: g.ID, UserID: g.UserID, PlantSpeciesID: g.PlantSpeciesID,
		Nickname: g.Nickname, OwnedSince: g.OwnedSince, Location: g.Location,
		CareReminderID: g.CareReminderID, Status: g.Status, CreatedAt: g.CreatedAt,
		PlantName: p.Name, PlantType: p.Type, PlantAlias: p.Alias,
		Origin: p.Origin, Family: p.Family, Genus: p.Genus,
	}
	v.DisplayLabel = buildGardenDisplay(v)
	return v
}

// Remove marks a pot removed and suspends its open reminders in one
// transaction. Open reminders become awaiting_confirm and keep their plant
// species link so the user can cancel or transfer them. Any failure rolls the
// pot back to active — the original relation and reminders are restored.
func (s *UserGardenService) Remove(userID, id uint) (suspended int64, err error) {
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txGarden := s.repo.WithTx(tx)
		txReminder := s.reminderRepo.WithTx(tx)

		garden, ferr := txGarden.FindOwned(nil, id, userID)
		if ferr != nil {
			if errors.Is(ferr, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] not found", id))
			}
			return fmt.Errorf("user garden remove find: %w", ferr)
		}

		if ferr = txGarden.MarkRemoved(tx, garden.ID); ferr != nil {
			return fmt.Errorf("user garden mark removed: %w", ferr)
		}
		n, ferr := txReminder.SuspendOpenByGarden(tx, garden.ID, userID)
		if ferr != nil {
			// Compensate immediately; the outer transaction rollback also
			// restores the row, but keeping the relation recovery explicit.
			_ = txGarden.Restore(tx, garden.ID)
			return fmt.Errorf("user garden suspend reminders: %w", ferr)
		}
		suspended = n
		s.logger.Info(fmt.Sprintf(constants.LogGardenRemoved, garden.ID, n),
			"user_id", userID, "plant_species_id", garden.PlantSpeciesID)
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return suspended, nil
}

// BindReminder associates a care reminder with a garden item. The reminder
// must belong to the same user and (when set) the same plant species.
func (s *UserGardenService) BindReminder(userID, gardenID, reminderID uint) (*dto.GardenView, error) {
	var out *dto.GardenView
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txGarden := s.repo.WithTx(tx)
		txReminder := s.reminderRepo.WithTx(tx)

		item, err := txGarden.FindOwned(nil, gardenID, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] not found", gardenID))
			}
			return fmt.Errorf("user garden bind find: %w", err)
		}
		reminder, err := txReminder.FindByID(reminderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("CareReminder[id=%d] not found", reminderID))
			}
			return fmt.Errorf("user garden bind reminder find: %w", err)
		}
		if reminder.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("CareReminder[id=%d] bind failed: user_id=%d not owner", reminderID, userID))
		}
		if reminder.PlantSpeciesID != 0 && reminder.PlantSpeciesID != item.PlantSpeciesID {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("bind failed: CareReminder[id=%d] plant_species_id=%d differs from garden=%d",
					reminderID, reminder.PlantSpeciesID, item.PlantSpeciesID))
		}
		item.CareReminderID = reminderID
		if err := txGarden.Update(item); err != nil {
			return fmt.Errorf("user garden bind update: %w", err)
		}
		// Back-link the occurrence to this pot as well, unless it already
		// belongs to a different active pot.
		if reminder.GardenID == 0 || reminder.GardenID == item.ID {
			reminder.GardenID = item.ID
			if reminder.PlantSpeciesID == 0 {
				reminder.PlantSpeciesID = item.PlantSpeciesID
			}
			if err := txReminder.Update(reminder); err != nil {
				return fmt.Errorf("user garden bind reminder link: %w", err)
			}
		}
		v, verr := s.viewOf(item)
		if verr != nil {
			return verr
		}
		out = v
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
