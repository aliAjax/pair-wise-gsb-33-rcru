package util

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
	d := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	if got := FormatDate(d); got != "2026-08-16" {
		t.Errorf("FormatDate = %s", got)
	}
}

func TestPlantTypeText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"flower", "观花"},
		{"foliage", "观叶"},
		{"succulent", "多肉"},
		{"aquatic", "水生"},
		{"unknown", "未知"},
	}
	for _, c := range cases {
		if got := PlantTypeText(c.in); got != c.want {
			t.Errorf("PlantTypeText(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestReminderStatusText(t *testing.T) {
	cases := map[string]string{
		"pending": "待处理",
		"done":    "已完成",
		"overdue": "已逾期",
		"unbound": "待确认",
	}
	for in, want := range cases {
		if got := ReminderStatusText(in); got != want {
			t.Errorf("ReminderStatusText(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestReminderFrequencyText(t *testing.T) {
	cases := map[string]string{
		"once": "单次", "daily": "每日", "weekly": "每周",
		"monthly": "每月", "yearly": "每年", "": "单次",
	}
	for in, want := range cases {
		if got := ReminderFrequencyText(in); got != want {
			t.Errorf("ReminderFrequencyText(%s) = %s, want %s", in, got, want)
		}
	}
}
