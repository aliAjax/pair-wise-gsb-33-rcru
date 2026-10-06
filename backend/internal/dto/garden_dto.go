package dto

import "time"

// GardenAddRequest adds a plant to the user's garden.
type GardenAddRequest struct {
	PlantSpeciesID uint      `json:"plant_species_id" binding:"required"`
	Nickname       string    `json:"nickname" binding:"omitempty,max=64"`
	OwnedSince     time.Time `json:"owned_since"`
	Location       string    `json:"location" binding:"omitempty,max=128"`
}

// GardenBindRequest binds a reminder to a garden item.
type GardenBindRequest struct {
	ReminderID uint `json:"care_reminder_id" binding:"required"`
}

// GardenView is a garden pot enriched with plant source info and its pending
// reminder summary, for the "my garden" page.
type GardenView struct {
	ID             uint      `json:"id"`
	UserID         uint      `json:"user_id"`
	PlantSpeciesID uint      `json:"plant_species_id"`
	Nickname       string    `json:"nickname"`
	OwnedSince     time.Time `json:"owned_since"`
	Location       string    `json:"location"`
	CareReminderID uint      `json:"care_reminder_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`

	// Source information rendered by the UI.
	PlantName    string `json:"plant_name"`
	PlantType    string `json:"plant_type"`
	PlantAlias   string `json:"plant_alias"`
	Origin       string `json:"origin"`
	Family       string `json:"family"`
	Genus        string `json:"genus"`
	DisplayLabel string `json:"display_label"`
}

// RemovedGardenView describes a removed pot with reminders awaiting
// confirmation, plus the transferable pots of the same plant species.
type RemovedGardenView struct {
	Garden   GardenView     `json:"garden"`
	Reminders []ReminderView `json:"reminders"`
	Targets  []GardenView   `json:"targets"`
}
