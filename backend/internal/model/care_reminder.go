package model

import "time"

// ReminderStatus values.
//
// pending  养护中：当前有效的下一期提醒（每个 series_key 只允许一个）
// done     历史：本期已完成；周期性提醒完成后会再生成一个 pending 下一期
// overdue  逾期：仍需完成的待办（pending 的日期过去后由系统标记）
// unbound  待确认：植物移出花园后，尚未完成的提醒停在此状态，可取消或转养
const (
	ReminderPending = "pending"
	ReminderDone    = "done"
	ReminderOverdue = "overdue"
	ReminderUnbound = "unbound"
)

// CareReminder is a scheduled gardening task owned by a user.
//
// SeriesKey 标识一条“周期计划”：同一用户 + 同一盆植物（或无盆）+ 同一任务
// + 同一频率共享一个 series_key，每个 series_key 同时只占一个下一期位置
// （pending/overdue）。完成该位置后按频率滚到下一期；改动日期或频率后
// schedule_version +1，旧计划视为失效并立即按新参数重算。
type CareReminder struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	PlantSpeciesID  uint      `json:"plant_species_id"`
	GardenID        uint      `gorm:"not null;default:0" json:"garden_id"`
	TaskTitle       string    `gorm:"size:255;not null" json:"task_title"`
	RemindDate      time.Time `gorm:"type:date" json:"remind_date"`
	Frequency       string    `gorm:"size:32;not null;default:once" json:"frequency"`
	Status          string    `gorm:"size:16;default:pending" json:"status"`
	// 索引（idx_reminder_series / uk_reminder_series_date 等）由迁移脚本统一
	// 建立，AutoMigrate 不重复创建，避免索引名冲突。
	SeriesKey       string    `gorm:"size:255;not null;default:''" json:"series_key"`
	ScheduleVersion uint      `gorm:"not null;default:1" json:"schedule_version"`
	CreatedAt       time.Time `json:"created_at"`
}

// CareReminderView is a reminder joined with its plant source and garden pot,
// used by list endpoints to display 来源植物与当前状态. It is never migrated.
type CareReminderView struct {
	CareReminder `gorm:"embedded"`
	PlantName  string `gorm:"column:plant_name" json:"plant_name,omitempty"`
	Origin     string `gorm:"column:origin" json:"origin,omitempty"`
	PlantType  string `gorm:"column:plant_type" json:"plant_type,omitempty"`
	GardenName string `gorm:"column:garden_name" json:"garden_name,omitempty"`
}
