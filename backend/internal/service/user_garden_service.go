package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"

	"gorm.io/gorm"
)

// UserGardenService implements "my garden" list logic.
type UserGardenService struct {
	repo         *repository.UserGardenRepository
	reminderRepo *repository.CareReminderRepository
	logger       *slog.Logger
}

// NewUserGardenService creates a UserGardenService.
func NewUserGardenService(repo *repository.UserGardenRepository, reminderRepo *repository.CareReminderRepository, logger *slog.Logger) *UserGardenService {
	return &UserGardenService{repo: repo, reminderRepo: reminderRepo, logger: logger}
}

// Add adds a plant (one pot) to a user's garden. Multiple pots of the same
// species are allowed. If the user previously removed a pot of this species,
// reminders left in "unbound / 待确认" are re-attached to the new pot.
func (s *UserGardenService) Add(userID uint, g *model.UserGarden) (*model.UserGarden, error) {
	g.UserID = userID
	if g.OwnedSince.IsZero() {
		g.OwnedSince = time.Now()
	}
	var claimed int64
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		// 保存失败时整笔事务回滚：新建关系不会残留。
		if err := s.repo.CreateTx(tx, g); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("UserGarden[user_id=%d plant_id=%d] add failed: already in garden", userID, g.PlantSpeciesID))
			}
			s.logger.Error(fmt.Sprintf(constants.LogGardenAddFailed, g.PlantSpeciesID, userID), "error", err)
			return fmt.Errorf("user garden add: %w", err)
		}
		n, err := s.reminderRepo.ReopenForGarden(tx, userID, g.ID, g.PlantSpeciesID)
		if err != nil {
			return fmt.Errorf("user garden add restore reminders: %w", err)
		}
		claimed = n
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogGardenAddSuccess, g.PlantSpeciesID, userID), "id", g.ID, "reminders_restored", claimed)
	return g, nil
}

// List returns a user's garden items with plant source and care status.
func (s *UserGardenService) List(userID uint) ([]model.UserGardenView, error) {
	items, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("user garden list: %w", err)
	}
	return items, nil
}

// Remove deletes a garden item. Unfinished reminders of that pot are not
// deleted — they stop at the "unbound / 待确认" state so the user can cancel
// them or transfer to another pot of the same species. Everything runs in one
// transaction; any failure restores the garden relation and reminder linkage.
func (s *UserGardenService) Remove(userID, id uint) (transferTargets []model.UserGarden, err error) {
	var unboundCount int64
	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		g, fErr := s.repo.FindByIDTx(tx, id)
		if fErr != nil {
			if errors.Is(fErr, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("UserGarden[id=%d] not found", id))
			}
			return fmt.Errorf("user garden remove find: %w", fErr)
		}
		if g.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("UserGarden[id=%d] remove failed: user_id=%d not owner", id, userID))
		}

		open, fErr := s.reminderRepo.ListOpenByGarden(tx, id)
		if fErr != nil {
			return fmt.Errorf("user garden remove reminders find: %w", fErr)
		}
		for i := range open {
			open[i].GardenID = 0
			open[i].Status = model.ReminderUnbound
			if fErr = s.reminderRepo.UpdateTx(tx, &open[i]); fErr != nil {
				return fmt.Errorf("user garden remove reminder unbind: %w", fErr)
			}
			unboundCount++
		}

		if fErr = s.repo.DeleteTx(tx, g.ID); fErr != nil {
			return fmt.Errorf("user garden remove delete: %w", fErr)
		}

		// 同品种的另一盆可作为转养目标（事务内查询，供界面直接展示）。
		transferTargets, fErr = s.repo.ListSameSpecies(tx, userID, g.PlantSpeciesID, g.ID)
		if fErr != nil {
			return fmt.Errorf("user garden remove transfer lookup: %w", fErr)
		}
		return nil
	})
	if err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogGardenRemoveRollback, id), "error", err)
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogGardenPlantRemoved, id, unboundCount),
		"user_id", userID, "transfer_targets", len(transferTargets))
	return transferTargets, nil
}

// BindReminder associates a care reminder with a garden item.
func (s *UserGardenService) BindReminder(userID, gardenID, reminderID uint) (*model.UserGarden, error) {
	item, err := s.repo.FindByID(gardenID)
	if err != nil {
		return nil, fmt.Errorf("user garden bind find: %w", err)
	}
	if item.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden, fmt.Sprintf("UserGarden[id=%d] bind failed: not owner", gardenID))
	}
	item.CareReminderID = reminderID
	if err := s.repo.Update(item); err != nil {
		return nil, fmt.Errorf("user garden bind update: %w", err)
	}
	return item, nil
}
