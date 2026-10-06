package dto

import "time"

// ReminderCreateRequest is the payload for creating a care reminder.
type ReminderCreateRequest struct {
	PlantSpeciesID uint      `json:"plant_species_id"`
	GardenID       uint      `json:"garden_id"`
	TaskTitle      string    `json:"task_title" binding:"required,max=255"`
	RemindDate     time.Time `json:"remind_date" binding:"required"`
	Frequency      string    `json:"frequency" binding:"omitempty,max=32"`
}

// ReminderUpdateRequest edits a reminder schedule. Changing remind_date or
// frequency invalidates the old next occurrence and recomputes it.
type ReminderUpdateRequest struct {
	TaskTitle  string    `json:"task_title" binding:"omitempty,max=255"`
	RemindDate time.Time `json:"remind_date" binding:"omitempty"`
	Frequency  string    `json:"frequency" binding:"omitempty,max=32"`
}

// ReminderStatusRequest carries the new status for a reminder.
type ReminderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ReminderTransferRequest moves an unbound reminder onto another pot of the
// same plant species after the original pot left the garden.
type ReminderTransferRequest struct {
	GardenID uint `json:"garden_id" binding:"required"`
}
