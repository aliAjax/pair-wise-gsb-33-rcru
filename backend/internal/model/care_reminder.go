package model

import "time"

// ReminderStatus values. The state machine is intentionally mirrored in
// service transitions, frontend button visibility, log templates, error
// codes and formatters.
const (
	ReminderPending         = "pending"
	ReminderDone            = "done"
	ReminderOverdue         = "overdue"
	ReminderAwaitingConfirm = "awaiting_confirm"
)

// Reminder frequency values. Mirrored in frontend/src/constants/reminder.ts.
const (
	FrequencyNone    = ""
	FrequencyDaily   = "daily"
	FrequencyWeekly  = "weekly"
	FrequencyMonthly = "monthly"
	FrequencyYearly  = "yearly"
)

// CareReminder is one occurrence of a scheduled gardening task.
//
// A recurring task is represented as a series: every occurrence shares the
// same SeriesID and increments Seq. At most one not-yet-finished occurrence
// (pending/overdue/awaiting_confirm) may exist per series — it is the single
// "next period" slot. ScheduleVersion bumps whenever the date or frequency
// changes so a pre-generated next occurrence can be invalidated and rebuilt.
type CareReminder struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	PlantSpeciesID  uint      `gorm:"index" json:"plant_species_id"`
	GardenID        uint      `gorm:"index;not null;default:0" json:"garden_id"`
	SeriesID        uint      `gorm:"index;not null;default:0" json:"series_id"`
	Seq             int       `gorm:"not null;default:1" json:"seq"`
	ScheduleVersion int       `gorm:"not null;default:1" json:"schedule_version"`
	TaskTitle       string    `gorm:"size:255;not null" json:"task_title"`
	RemindDate      time.Time `gorm:"type:date;index" json:"remind_date"`
	Frequency       string    `gorm:"size:32" json:"frequency"`
	Status          string    `gorm:"size:32;default:pending;index" json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// IsOpen reports whether the occurrence still occupies the series' "next period" slot.
func (r *CareReminder) IsOpen() bool {
	return r.Status == ReminderPending || r.Status == ReminderOverdue || r.Status == ReminderAwaitingConfirm
}
