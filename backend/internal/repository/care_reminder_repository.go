package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// CareReminderRepository handles persistence of care reminders.
type CareReminderRepository struct {
	db *gorm.DB
}

// NewCareReminderRepository creates a CareReminderRepository.
func NewCareReminderRepository(db *gorm.DB) *CareReminderRepository {
	return &CareReminderRepository{db: db}
}

// WithTx returns a repository bound to the given transaction handle.
func (r *CareReminderRepository) WithTx(tx *gorm.DB) *CareReminderRepository {
	return &CareReminderRepository{db: tx}
}

// Create inserts a reminder.
func (r *CareReminderRepository) Create(m *model.CareReminder) error {
	return r.db.Create(m).Error
}

// FindByID locates a reminder by id.
func (r *CareReminderRepository) FindByID(id uint) (*model.CareReminder, error) {
	var m model.CareReminder
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// Update persists a reminder.
func (r *CareReminderRepository) Update(m *model.CareReminder) error {
	return r.db.Save(m).Error
}

// Delete removes a reminder.
func (r *CareReminderRepository) Delete(id uint) error {
	res := r.db.Delete(&model.CareReminder{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByUser returns reminders for a user with optional status filter.
func (r *CareReminderRepository) ListByUser(userID uint, status string) ([]model.CareReminder, error) {
	var items []model.CareReminder
	q := r.db.Where("user_id = ?", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("remind_date ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByMonth returns reminders for a user within a month of a given year.
func (r *CareReminderRepository) ListByMonth(userID uint, year, month int) ([]model.CareReminder, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	var items []model.CareReminder
	if err := r.db.Where("user_id = ? AND remind_date >= ? AND remind_date < ?", userID, start, end).
		Order("remind_date ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// MarkOverdue flips pending reminders whose date has passed to overdue.
func (r *CareReminderRepository) MarkOverdue(userID uint) (int64, error) {
	res := r.db.Model(&model.CareReminder{}).
		Where("user_id = ? AND status = ? AND remind_date < ?", userID, model.ReminderPending, time.Now()).
		Update("status", model.ReminderOverdue)
	return res.RowsAffected, res.Error
}

// ListByGarden returns reminders attached to a garden pot, optionally filtered
// by status.
func (r *CareReminderRepository) ListByGarden(gardenID uint, status string) ([]model.CareReminder, error) {
	var items []model.CareReminder
	q := r.db.Where("garden_id = ?", gardenID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("remind_date ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindOpenBySeries returns the single not-yet-finished occurrence of a series,
// i.e. the one occupying the "next period" slot. Returns ErrNotFound when the
// whole series is finished (one-off done, or no successor generated yet).
func (r *CareReminderRepository) FindOpenBySeries(tx *gorm.DB, seriesID uint) (*model.CareReminder, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	var m model.CareReminder
	err := handle.
		Where("series_id = ? AND status IN ?", seriesID, []string{
			model.ReminderPending, model.ReminderOverdue, model.ReminderAwaitingConfirm,
		}).
		Order("seq DESC").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// CompleteOpen performs the idempotent completion of an open occurrence.
// The conditional UPDATE is the concurrency guard: when two windows tap
// "done" at the same time only one statement matches a row, so the second
// caller gets RowsAffected == 0 and reports "already processed" instead of
// generating a second next-period occurrence.
func (r *CareReminderRepository) CompleteOpen(tx *gorm.DB, id, userID uint) (int64, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	res := handle.Model(&model.CareReminder{}).
		Where("id = ? AND user_id = ? AND status IN ?", id, userID,
			[]string{model.ReminderPending, model.ReminderOverdue}).
		Update("status", model.ReminderDone)
	return res.RowsAffected, res.Error
}

// ReattachOpenByGarden rewrites the open reminders of a removed pot onto
// another pot of the same plant species during a transfer.
func (r *CareReminderRepository) ReattachOpenByGarden(tx *gorm.DB, fromGardenID, toGardenID, userID, plantSpeciesID uint) (int64, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	res := handle.Model(&model.CareReminder{}).
		Where("garden_id = ? AND user_id = ? AND plant_species_id = ? AND status = ?",
			fromGardenID, userID, plantSpeciesID, model.ReminderAwaitingConfirm).
		Updates(map[string]interface{}{
			"garden_id": toGardenID,
			"status":    model.ReminderPending,
		})
	return res.RowsAffected, res.Error
}

// CancelOpenByGarden cancels (deletes) all awaiting reminders of a removed pot.
func (r *CareReminderRepository) CancelOpenByGarden(tx *gorm.DB, gardenID, userID uint) (int64, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	res := handle.
		Where("garden_id = ? AND user_id = ? AND status = ?",
			gardenID, userID, model.ReminderAwaitingConfirm).
		Delete(&model.CareReminder{})
	return res.RowsAffected, res.Error
}

// SuspendOpenByGarden moves open reminders of a removed pot into the
// awaiting_confirm state and detaches the pot link (plant_species_id and the
// series are retained so the reminder can be transferred later).
func (r *CareReminderRepository) SuspendOpenByGarden(tx *gorm.DB, gardenID, userID uint) (int64, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	res := handle.Model(&model.CareReminder{}).
		Where("garden_id = ? AND user_id = ? AND status IN ?", gardenID, userID,
			[]string{model.ReminderPending, model.ReminderOverdue}).
		Update("status", model.ReminderAwaitingConfirm)
	return res.RowsAffected, res.Error
}

// DeleteOpenSuccessor removes the pre-generated next occurrence of a series
// (any version) so it can be recalculated after a schedule change.
func (r *CareReminderRepository) DeleteOpenSuccessor(tx *gorm.DB, seriesID, exceptID uint) (int64, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	q := handle.Where("series_id = ? AND id <> ? AND status IN ?", seriesID, exceptID,
		[]string{model.ReminderPending, model.ReminderOverdue, model.ReminderAwaitingConfirm})
	res := q.Delete(&model.CareReminder{})
	return res.RowsAffected, res.Error
}

// MaxSeqOfSeries returns the highest seq within a series.
func (r *CareReminderRepository) MaxSeqOfSeries(tx *gorm.DB, seriesID uint) (int, error) {
	handle := r.db
	if tx != nil {
		handle = tx
	}
	var maxSeq *int
	if err := handle.Model(&model.CareReminder{}).
		Where("series_id = ?", seriesID).
		Select("MAX(seq)").Scan(&maxSeq).Error; err != nil {
		return 0, err
	}
	if maxSeq == nil {
		return 0, nil
	}
	return *maxSeq, nil
}
