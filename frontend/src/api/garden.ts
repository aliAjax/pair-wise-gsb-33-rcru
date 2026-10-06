import request from '@/utils/request'
import type { UserGarden } from '@/types/api'

export function listGardens() {
  return request.get<never, UserGarden[]>('/gardens')
}

export function addGarden(payload: { plant_species_id: number; nickname?: string; owned_since?: string; location?: string }) {
  return request.post<never, UserGarden>('/gardens', payload)
}

export function bindReminder(id: number, careReminderId: number) {
  return request.put<never, UserGarden>(`/gardens/${id}/reminder`, { care_reminder_id: careReminderId })
}

// Removing a pot suspends its open reminders to awaiting_confirm inside one
// transaction; the response reports how many reminders are awaiting a decision.
export function removeGarden(id: number) {
  return request.delete<never, { removed: boolean; awaiting_reminders: number }>(`/gardens/${id}`)
}
