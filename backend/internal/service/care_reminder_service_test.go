package service

import (
	"testing"
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
)

func TestNormalizeDate(t *testing.T) {
	in := time.Date(2026, 10, 6, 15, 30, 45, 0, time.Local)
	got := normalizeDate(in)
	if got.Hour() != 0 || got.Minute() != 0 || got.Day() != 6 || got.Month() != time.October {
		t.Errorf("normalizeDate = %v, want 2026-10-06 00:00", got)
	}
}

func TestNextOccurrence(t *testing.T) {
	base := time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local)
	cases := []struct {
		name      string
		frequency string
		want      time.Time
	}{
		{"daily rolls one day", constants.FrequencyDaily, base.AddDate(0, 0, 1)},
		{"weekly rolls seven days", constants.FrequencyWeekly, base.AddDate(0, 0, 7)},
		{"monthly rolls one month", constants.FrequencyMonthly, time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)},
		{"yearly rolls one year", constants.FrequencyYearly, time.Date(2027, 1, 31, 0, 0, 0, 0, time.Local)},
		{"once has no next occurrence", constants.FrequencyOnce, time.Time{}},
		{"unknown frequency behaves once", "whatever", time.Time{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := nextOccurrence(base, c.frequency)
			if !got.Equal(c.want) {
				t.Errorf("nextOccurrence(%s) = %v, want %v", c.frequency, got, c.want)
			}
		})
	}
}

func TestSeriesKey(t *testing.T) {
	got := seriesKey(7, 3, "浇水", constants.FrequencyWeekly)
	want := "u7:g3:weekly:浇水"
	if got != want {
		t.Errorf("seriesKey = %q, want %q", got, want)
	}
	// 空频率按单次处理，保证一条计划一个 series_key
	if got := seriesKey(7, 3, "浇水", ""); got != "u7:g3:once:浇水" {
		t.Errorf("seriesKey empty frequency = %q", got)
	}
	// 同品种的两盆（garden 不同）是不同计划，各自只占一个下一期位置
	if seriesKey(7, 1, "施肥", constants.FrequencyMonthly) == seriesKey(7, 2, "施肥", constants.FrequencyMonthly) {
		t.Error("different pots must have different series keys")
	}
	// 同一盆上不同频率是不同计划
	if seriesKey(7, 1, "施肥", constants.FrequencyWeekly) == seriesKey(7, 1, "施肥", constants.FrequencyMonthly) {
		t.Error("different frequencies must have different series keys")
	}
}
