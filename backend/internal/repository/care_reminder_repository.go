package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// DB exposes the underlying handle so services can run transactions that span
// reminders and the garden.
func (r *CareReminderRepository) DB() *gorm.DB {
	return r.db
}

// Create inserts a reminder.
func (r *CareReminderRepository) Create(m *model.CareReminder) error {
	return r.CreateTx(r.db, m)
}

// CreateTx inserts a reminder inside an existing transaction.
func (r *CareReminderRepository) CreateTx(tx *gorm.DB, m *model.CareReminder) error {
	if err := tx.Create(m).Error; err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// FindByID locates a reminder by id.
func (r *CareReminderRepository) FindByID(id uint) (*model.CareReminder, error) {
	return r.FindByIDTx(r.db, id)
}

// FindByIDTx locates a reminder by id inside a transaction and locks the row
// (SELECT ... FOR UPDATE), serializing concurrent "complete" requests on the
// same reminder.
func (r *CareReminderRepository) FindByIDTx(tx *gorm.DB, id uint) (*model.CareReminder, error) {
	var m model.CareReminder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// Update persists a reminder.
func (r *CareReminderRepository) Update(m *model.CareReminder) error {
	return r.UpdateTx(r.db, m)
}

// UpdateTx persists a reminder inside an existing transaction.
func (r *CareReminderRepository) UpdateTx(tx *gorm.DB, m *model.CareReminder) error {
	return tx.Save(m).Error
}

// Delete removes a reminder by id.
func (r *CareReminderRepository) Delete(id uint) error {
	return r.DeleteTx(r.db, id)
}

// DeleteTx removes a reminder by id inside an existing transaction.
func (r *CareReminderRepository) DeleteTx(tx *gorm.DB, id uint) error {
	res := tx.Delete(&model.CareReminder{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ActiveBySeries returns the single open next-occurrence slot of a plan, i.e.
// the pending/overdue/unbound reminder carrying the series key.
func (r *CareReminderRepository) ActiveBySeries(tx *gorm.DB, userID uint, seriesKey string) (*model.CareReminder, error) {
	var m model.CareReminder
	err := tx.Where("user_id = ? AND series_key = ? AND status IN ?",
		userID, seriesKey, []string{model.ReminderPending, model.ReminderOverdue, model.ReminderUnbound}).
		Order("remind_date DESC, id DESC").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ExistsOpenAtDate reports whether the plan already occupies a slot on date.
// It is the application-level guard for "one next-occurrence per frequency";
// the uk_reminder_series_date unique index is the database-level backstop.
func (r *CareReminderRepository) ExistsOpenAtDate(tx *gorm.DB, userID uint, seriesKey string, date time.Time, excludeID uint) (bool, error) {
	var count int64
	err := tx.Model(&model.CareReminder{}).
		Where("user_id = ? AND series_key = ? AND remind_date = ? AND id <> ? AND status IN ?",
			userID, seriesKey, date, excludeID,
			[]string{model.ReminderPending, model.ReminderOverdue, model.ReminderUnbound}).
		Count(&count).Error
	return count > 0, err
}

// ListByUser returns reminders for a user with optional status filter, joined
// with plant species and garden display names.
func (r *CareReminderRepository) ListByUser(userID uint, status string) ([]model.CareReminderView, error) {
	var items []model.CareReminderView
	q := r.baseListQuery(r.db).Where("cr.user_id = ?", userID)
	if status != "" {
		q = q.Where("cr.status = ?", status)
	}
	if err := q.Order("cr.remind_date ASC, cr.id ASC").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByMonth returns reminders for a user within a month of a given year.
func (r *CareReminderRepository) ListByMonth(userID uint, year, month int) ([]model.CareReminderView, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	var items []model.CareReminderView
	if err := r.baseListQuery(r.db).
		Where("cr.user_id = ? AND cr.remind_date >= ? AND cr.remind_date < ?", userID, start, end).
		Order("cr.remind_date ASC, cr.id ASC").Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CareReminderRepository) baseListQuery(tx *gorm.DB) *gorm.DB {
	return tx.Table("care_reminders AS cr").
		Select("cr.*, p.name AS plant_name, p.origin AS origin, p.type AS plant_type, g.nickname AS garden_name").
		Joins("LEFT JOIN plant_species p ON p.id = cr.plant_species_id").
		Joins("LEFT JOIN user_gardens g ON g.id = cr.garden_id AND cr.garden_id <> 0")
}

// MarkOverdue flips pending reminders whose date has passed to overdue.
func (r *CareReminderRepository) MarkOverdue(userID uint) (int64, error) {
	res := r.db.Model(&model.CareReminder{}).
		Where("user_id = ? AND status = ? AND remind_date < ?", userID, model.ReminderPending, time.Now()).
		Update("status", model.ReminderOverdue)
	return res.RowsAffected, res.Error
}

// ListOpenByGarden returns unfinished reminders (pending/overdue) of one pot.
func (r *CareReminderRepository) ListOpenByGarden(tx *gorm.DB, gardenID uint) ([]model.CareReminder, error) {
	var items []model.CareReminder
	if err := tx.Where("garden_id = ? AND status IN ?", gardenID,
		[]string{model.ReminderPending, model.ReminderOverdue}).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ReopenForGarden moves unbound reminders of a plant species back onto a pot,
// returning how many reminders were claimed. Past-dated reminders stay overdue.
func (r *CareReminderRepository) ReopenForGarden(tx *gorm.DB, userID, gardenID, plantSpeciesID uint) (int64, error) {
	res := tx.Model(&model.CareReminder{}).
		Where("user_id = ? AND garden_id = 0 AND plant_species_id = ? AND status = ?",
			userID, plantSpeciesID, model.ReminderUnbound).
		Updates(map[string]interface{}{
			"garden_id": gardenID,
			"status":    gorm.Expr("CASE WHEN remind_date < CURDATE() THEN ? ELSE ? END", model.ReminderOverdue, model.ReminderPending),
		})
	return res.RowsAffected, res.Error
}
