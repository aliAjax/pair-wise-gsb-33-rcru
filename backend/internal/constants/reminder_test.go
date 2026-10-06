package constants

import "testing"

func TestReminderFrequencyValidation(t *testing.T) {
	valid := []string{"", FrequencyOnce, FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyYearly}
	for _, f := range valid {
		if !IsValidReminderFrequency(f) {
			t.Errorf("IsValidReminderFrequency(%q) = false, want true", f)
		}
	}
	if IsValidReminderFrequency("biweekly") {
		t.Error("biweekly should be an invalid frequency")
	}
	recurring := map[string]bool{
		FrequencyOnce: false, FrequencyDaily: true, FrequencyWeekly: true,
		FrequencyMonthly: true, FrequencyYearly: true,
	}
	for f, want := range recurring {
		if got := IsRecurringFrequency(f); got != want {
			t.Errorf("IsRecurringFrequency(%s) = %v, want %v", f, got, want)
		}
	}
}
