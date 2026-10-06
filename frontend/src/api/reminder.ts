import request from '@/utils/request'
import type { CareReminder, ReminderStatusResult, UserGarden } from '@/types/api'

export function listReminders(status?: string) {
  return request.get<never, CareReminder[]>('/reminders', { params: { status } })
}

export function listRemindersByMonth(year: number, month: number) {
  return request.get<never, CareReminder[]>('/reminders/calendar', { params: { year, month } })
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

// 改动日期或频率：下一期失效并按新计划重算。
export function updateReminder(id: number, payload: { task_title?: string; remind_date?: string; frequency?: string }) {
  return request.put<never, CareReminder>(`/reminders/${id}`, payload)
}

// 完成/状态流转。完成是幂等的：重复完成返回 processed=true，不会再生成一份下一期。
export function updateReminderStatus(id: number, status: string) {
  return request.put<never, ReminderStatusResult>(`/reminders/${id}/status`, { status })
}

// 待确认提醒可转养到的同品种盆栽列表。
export function listTransferTargets(id: number) {
  return request.get<never, UserGarden[]>(`/reminders/${id}/transfer-targets`)
}

// 把待确认提醒转养给同品种的另一盆。
export function transferReminder(id: number, gardenId: number) {
  return request.post<never, { reminder: CareReminder; message: string }>(`/reminders/${id}/transfer`, { garden_id: gardenId })
}

// 取消（删除）待确认提醒。
export function cancelReminder(id: number) {
  return request.delete<never, { canceled: boolean; message: string }>(`/reminders/${id}/cancel`)
}

export function deleteReminder(id: number) {
  return request.delete<never, { deleted: boolean }>(`/reminders/${id}`)
}
