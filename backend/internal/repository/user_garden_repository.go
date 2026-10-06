package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// UserGardenRepository handles persistence of user garden items.
type UserGardenRepository struct {
	db *gorm.DB
}

// NewUserGardenRepository creates a UserGardenRepository.
func NewUserGardenRepository(db *gorm.DB) *UserGardenRepository {
	return &UserGardenRepository{db: db}
}

// DB exposes the underlying handle for cross-table transactions.
func (r *UserGardenRepository) DB() *gorm.DB {
	return r.db
}

// Create inserts a garden item.
func (r *UserGardenRepository) Create(g *model.UserGarden) error {
	return r.CreateTx(r.db, g)
}

// CreateTx inserts a garden item inside an existing transaction.
func (r *UserGardenRepository) CreateTx(tx *gorm.DB, g *model.UserGarden) error {
	if err := tx.Create(g).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// Find locates a garden item by user and plant.
func (r *UserGardenRepository) Find(userID, plantID uint) (*model.UserGarden, error) {
	var g model.UserGarden
	if err := r.db.Where("user_id = ? AND plant_species_id = ?", userID, plantID).First(&g).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

// FindByID locates a garden item by primary key.
func (r *UserGardenRepository) FindByID(id uint) (*model.UserGarden, error) {
	return r.FindByIDTx(r.db, id)
}

// FindByIDTx locates a garden item inside a transaction with row lock.
func (r *UserGardenRepository) FindByIDTx(tx *gorm.DB, id uint) (*model.UserGarden, error) {
	var g model.UserGarden
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

// Update persists a garden item.
func (r *UserGardenRepository) Update(g *model.UserGarden) error {
	return r.db.Save(g).Error
}

// DeleteTx removes a garden item by id inside an existing transaction.
func (r *UserGardenRepository) DeleteTx(tx *gorm.DB, id uint) error {
	res := tx.Delete(&model.UserGarden{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListSameSpecies returns other pots of the same plant species owned by the
// user, used as transfer targets when a pot leaves the garden.
func (r *UserGardenRepository) ListSameSpecies(tx *gorm.DB, userID, plantSpeciesID, excludeGardenID uint) ([]model.UserGarden, error) {
	var items []model.UserGarden
	if err := tx.Where("user_id = ? AND plant_species_id = ? AND id <> ?",
		userID, plantSpeciesID, excludeGardenID).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByUser returns all garden items of a user enriched with the plant
// species source info and the open/unbound reminder counts.
func (r *UserGardenRepository) ListByUser(userID uint) ([]model.UserGardenView, error) {
	var items []model.UserGardenView
	if err := r.db.Table("user_gardens AS ug").
		Select("ug.*, p.name AS plant_name, p.alias AS alias, p.type AS plant_type, p.origin AS origin, " +
			"p.water_frequency AS water_frequency, p.image_urls AS image_urls, " +
			"(SELECT COUNT(1) FROM care_reminders cr WHERE cr.garden_id = ug.id AND cr.status IN ('pending','overdue')) AS pending_reminder_count, " +
			"(SELECT COUNT(1) FROM care_reminders cr WHERE cr.garden_id = 0 AND cr.user_id = ug.user_id AND cr.plant_species_id = ug.plant_species_id AND cr.status = 'unbound') AS unbound_reminder_count").
		Joins("LEFT JOIN plant_species p ON p.id = ug.plant_species_id").
		Where("ug.user_id = ?", userID).
		Order("ug.id DESC").
		Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
