package model

import "time"

// UserGarden represents a pot/plant owned by a user inside their garden list.
//
// 同一用户可以拥有多个“同品种”的盆栽（例如两盆月季），因此 (user_id,
// plant_species_id) 只保留普通索引而非唯一约束。养护提醒通过
// care_reminders.garden_id 反向挂到具体一盆上。
type UserGarden struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	PlantSpeciesID uint      `gorm:"not null" json:"plant_species_id"`
	Nickname       string    `gorm:"size:64" json:"nickname"`
	OwnedSince     time.Time `gorm:"type:date" json:"owned_since"`
	Location       string    `gorm:"size:128" json:"location"`
	CareReminderID uint      `json:"care_reminder_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// UserGardenView is a garden pot joined with its source plant species and the
// current care reminder counts, used by GET /gardens to display 来源与状态.
type UserGardenView struct {
	UserGarden           `gorm:"embedded"`
	PlantName            string `gorm:"column:plant_name" json:"plant_name,omitempty"`
	Alias                string `gorm:"column:alias" json:"alias,omitempty"`
	PlantType            string `gorm:"column:plant_type" json:"plant_type,omitempty"`
	Origin               string `gorm:"column:origin" json:"origin,omitempty"`
	WaterFrequency       string `gorm:"column:water_frequency" json:"water_frequency,omitempty"`
	ImageURLs            string `gorm:"column:image_urls" json:"image_urls,omitempty"`
	PendingReminderCount int    `gorm:"column:pending_reminder_count" json:"pending_reminder_count"`
	UnboundReminderCount int    `gorm:"column:unbound_reminder_count" json:"unbound_reminder_count"`
}
