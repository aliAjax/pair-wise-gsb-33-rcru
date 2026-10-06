package model

import "time"

// GardenStatus values for a user-owned plant.
const (
	GardenStatusActive  = "active"
	GardenStatusRemoved = "removed"
)

// UserGarden represents a plant pot owned by a user inside their garden list.
// A user may own several pots of the same PlantSpecies, so (user_id,
// plant_species_id) is a plain index rather than unique. Removing a plant
// flips Status to removed instead of hard-deleting the row, which lets open
// reminders roll back to the pot when their cancellation/transfer fails.
type UserGarden struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index:idx_garden_user_plant;not null" json:"user_id"`
	PlantSpeciesID uint      `gorm:"index:idx_garden_user_plant;not null" json:"plant_species_id"`
	Nickname       string    `gorm:"size:64" json:"nickname"`
	OwnedSince     time.Time `gorm:"type:date" json:"owned_since"`
	Location       string    `gorm:"size:128" json:"location"`
	CareReminderID uint      `json:"care_reminder_id"`
	Status         string    `gorm:"size:16;default:active;index" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}
