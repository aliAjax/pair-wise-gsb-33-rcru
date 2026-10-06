package constants

// ReminderFrequency enumerates care reminder recurrence. The same set is
// mirrored in frontend constants/reminder.ts, ReminderList.vue, log templates,
// error codes and formatters, so adding a value touches the whole stack.
const (
	FrequencyOnce   = "once"   // 单次
	FrequencyDaily  = "daily"  // 每日
	FrequencyWeekly = "weekly" // 每周
	FrequencyMonthly = "monthly" // 每月
	FrequencyYearly = "yearly" // 每年
)

// ValidReminderFrequencies returns all accepted frequency values.
func ValidReminderFrequencies() []string {
	return []string{FrequencyOnce, FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyYearly}
}

// IsValidReminderFrequency reports whether f is a known frequency.
// Empty values are treated as one-shot reminders.
func IsValidReminderFrequency(f string) bool {
	if f == "" {
		return true
	}
	for _, v := range ValidReminderFrequencies() {
		if v == f {
			return true
		}
	}
	return false
}

// IsRecurringFrequency reports whether a reminder rolls to a next occurrence.
func IsRecurringFrequency(f string) bool {
	return f == FrequencyDaily || f == FrequencyWeekly || f == FrequencyMonthly || f == FrequencyYearly
}
