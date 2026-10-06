package service

import (
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

// nextRemindDate advances a date by one period of the given frequency.
// For one-off reminders (empty frequency) the zero time is returned, meaning
// no next occurrence is generated. Dates are normalized to midnight local
// time so the "next period" slot always lands on a calendar day.
func nextRemindDate(from time.Time, frequency string) time.Time {
	base := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
	switch frequency {
	case model.FrequencyDaily:
		return base.AddDate(0, 0, 1)
	case model.FrequencyWeekly:
		return base.AddDate(0, 0, 7)
	case model.FrequencyMonthly:
		return base.AddDate(0, 1, 0)
	case model.FrequencyYearly:
		return base.AddDate(1, 0, 0)
	default:
		return time.Time{}
	}
}

// isValidFrequency reports whether frequency is one of the supported values.
func isValidFrequency(frequency string) bool {
	switch frequency {
	case model.FrequencyNone, model.FrequencyDaily, model.FrequencyWeekly,
		model.FrequencyMonthly, model.FrequencyYearly:
		return true
	default:
		return false
	}
}
