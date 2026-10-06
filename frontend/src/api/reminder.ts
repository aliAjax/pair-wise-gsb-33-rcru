import request from '@/utils/request'
import type { CareReminder, ReminderCompleteResult, RemovedGardenGroup } from '@/types/api'

export function listReminders(status?: string) {
  return request.get<never, CareReminder[]>('/reminders', { params: { status } })
}

export function listRemindersByMonth(year: number, month: number) {
  return request.get<never, CareReminder[]>('/reminders/calendar', { params: { year, month } })
}

// Removed pots with unfinished reminders awaiting confirmation + same-species
// pots the reminders can be transferred to.
export function listAwaitingReminders() {
  return request.get<never, RemovedGardenGroup[]>('/reminders/awaiting')
}

export function createReminder(payload: {
  plant_species_id?: number
  garden_id?: number
  task_title: string
  remind_date: string
  frequency?: string
}) {
  return request.post<never, CareReminder>('/reminders', payload)
}

// Change task title / remind date / frequency. The server invalidates the
// pre-generated next occurrence and recalculates it within one transaction.
export function updateReminder(
  id: number,
  payload: { task_title?: string; remind_date?: string; frequency?: string },
) {
  return request.put<never, ReminderCompleteResult>(`/reminders/${id}`, payload)
}

// Idempotent completion: a repeated call (double window/double click) returns
// processed=false with "already processed" instead of creating a 2nd next slot.
export function completeReminder(id: number) {
  return request.post<never, ReminderCompleteResult>(`/reminders/${id}/complete`)
}

export function updateReminderStatus(id: number, status: string) {
  return request.put<never, ReminderCompleteResult>(`/reminders/${id}/status`, { status })
}

export function deleteReminder(id: number) {
  return request.delete<never, { deleted: boolean }>(`/reminders/${id}`)
}

// Cancel (delete) all awaiting reminders of a removed pot.
export function cancelAwaitingReminders(gardenId: number) {
  return request.delete<never, { canceled: number }>(`/gardens/${gardenId}/reminders/awaiting`)
}

// Transfer awaiting reminders to another active pot of the same species.
export function transferAwaitingReminders(gardenId: number, targetGardenId: number) {
  return request.post<never, { transferred: number }>(`/gardens/${gardenId}/reminders/transfer`, {
    target_garden_id: targetGardenId,
  })
}
