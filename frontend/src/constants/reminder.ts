import type { ReminderFrequency, ReminderStatus } from '@/types/api'

// Mirrors backend/internal/model/care_reminder.go frequency constants.
export const REMINDER_FREQUENCIES: { value: ReminderFrequency; label: string }[] = [
  { value: '', label: '单次' },
  { value: 'daily', label: '每日' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
  { value: 'yearly', label: '每年' },
]

export const FrequencyTextMap: Record<string, string> = {
  '': '单次',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  yearly: '每年',
}

// Mirrors the backend care reminder state machine:
// pending -> done / overdue; moving the plant out suspends to awaiting_confirm.
export const ReminderStatusTextMap: Record<ReminderStatus, string> = {
  pending: '待处理',
  done: '已完成',
  overdue: '已逾期',
  awaiting_confirm: '待确认',
}

export function statusTagType(s: ReminderStatus): 'warning' | 'success' | 'danger' | 'info' {
  switch (s) {
    case 'pending':
      return 'warning'
    case 'done':
      return 'success'
    case 'overdue':
      return 'danger'
    case 'awaiting_confirm':
      return 'info'
  }
}

export function frequencyText(f: string): string {
  return FrequencyTextMap[f ?? ''] ?? f ?? '-'
}
