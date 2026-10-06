package repository

import (
	"errors"

	"gorm.io/gorm"

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

// WithTx returns a repository bound to the given transaction handle.
func (r *UserGardenRepository) WithTx(tx *gorm.DB) *UserGardenRepository {
	return &UserGardenRepository{db: tx}
}

// Create inserts a garden item.
func (r *UserGardenRepository) Create(g *model.UserGarden) error {
	if err := r.db.Create(g).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// Find locates an active garden item by user and plant.
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
	var g model.UserGarden
	if err := r.db.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

// FindOwned locates an active garden item owned by the user.
func (r *UserGardenRepository) FindOwned(tx *gorm.DB, id, userID uint) (*model.UserGarden, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	var g model.UserGarden
	err := handle.Where("id = ? AND user_id = ? AND status = ?", id, userID, model.GardenStatusActive).First(&g).Error
	if err != nil {
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

// MarkRemoved flips a garden item to removed inside a transaction.
func (r *UserGardenRepository) MarkRemoved(tx *gorm.DB, id uint) error {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	res := handle.Model(&model.UserGarden{}).
		Where("id = ? AND status = ?", id, model.GardenStatusActive).
		Update("status", model.GardenStatusRemoved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Restore flips a garden item back to active (compensation after a failed
// reminder suspend within the removal transaction).
func (r *UserGardenRepository) Restore(tx *gorm.DB, id uint) error {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	return handle.Model(&model.UserGarden{}).
		Where("id = ?", id).
		Update("status", model.GardenStatusActive).Error
}

// Delete removes a garden item by id.
func (r *UserGardenRepository) Delete(id uint) error {
	return r.db.Delete(&model.UserGarden{}, id).Error
}

// ListByUser returns all active garden items of a user.
func (r *UserGardenRepository) ListByUser(userID uint) ([]model.UserGarden, error) {
	var items []model.UserGarden
	if err := r.db.Where("user_id = ? AND status = ?", userID, model.GardenStatusActive).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListRemoved returns the user's removed pots that still have reminders
// awaiting confirmation.
func (r *UserGardenRepository) ListRemoved(userID uint) ([]model.UserGarden, error) {
	handle := r.db
	var items []model.UserGarden
	if err := handle.
		Where("user_id = ? AND status = ?", userID, model.GardenStatusRemoved).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByIDs locates several garden pots at once for view assembly.
func (r *UserGardenRepository) ListByIDs(ids []uint) ([]model.UserGarden, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []model.UserGarden
	if err := r.db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListTransferTargets returns active pots of the same species owned by the
// user, excluding the source pot.
func (r *UserGardenRepository) ListTransferTargets(tx *gorm.DB, userID, plantSpeciesID, excludeGardenID uint) ([]model.UserGarden, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	var items []model.UserGarden
	err := handle.
		Where("user_id = ? AND plant_species_id = ? AND status = ? AND id <> ?",
			userID, plantSpeciesID, model.GardenStatusActive, excludeGardenID).
		Order("id DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
