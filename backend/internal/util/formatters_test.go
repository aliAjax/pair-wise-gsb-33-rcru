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
	cases := []struct{ in, want string }{
		{"pending", "待处理"},
		{"done", "已完成"},
		{"overdue", "已逾期"},
		{"awaiting_confirm", "待确认"},
	}
	for _, c := range cases {
		if got := ReminderStatusText(c.in); got != c.want {
			t.Errorf("ReminderStatusText(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestReminderFrequencyText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"daily", "每日"},
		{"weekly", "每周"},
		{"monthly", "每月"},
		{"yearly", "每年"},
		{"", "单次"},
	}
	for _, c := range cases {
		if got := ReminderFrequencyText(c.in); got != c.want {
			t.Errorf("ReminderFrequencyText(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestGardenStatusText(t *testing.T) {
	if got := GardenStatusText("removed"); got != "已移出" {
		t.Errorf("GardenStatusText = %s", got)
	}
}
