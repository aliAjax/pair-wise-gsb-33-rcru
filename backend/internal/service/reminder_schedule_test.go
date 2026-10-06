package service

import (
	"testing"
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/model"
)

func TestNextRemindDate(t *testing.T) {
	anchor := time.Date(2026, 1, 31, 9, 30, 0, 0, time.Local)
	cases := []struct {
		frequency string
		want      time.Time
		zero      bool
	}{
		{model.FrequencyDaily, time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local), false},
		{model.FrequencyWeekly, time.Date(2026, 2, 7, 0, 0, 0, 0, time.Local), false},
		{model.FrequencyMonthly, time.Date(2026, 2, 28, 0, 0, 0, 0, time.Local), false},
		{model.FrequencyYearly, time.Date(2027, 1, 31, 0, 0, 0, 0, time.Local), false},
		{model.FrequencyNone, time.Time{}, true},
		{"weird", time.Time{}, true},
	}
	for _, c := range cases {
		got := nextRemindDate(anchor, c.frequency)
		if c.zero {
			if !got.IsZero() {
				t.Errorf("nextRemindDate(%s) = %v, want zero", c.frequency, got)
			}
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("nextRemindDate(%s) = %v, want %v", c.frequency, got, c.want)
		}
	}
}

func TestIsValidFrequency(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{model.FrequencyDaily, true},
		{model.FrequencyWeekly, true},
		{model.FrequencyMonthly, true},
		{model.FrequencyYearly, true},
		{"hourly", false},
	}
	for _, c := range cases {
		if got := isValidFrequency(c.in); got != c.want {
			t.Errorf("isValidFrequency(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
