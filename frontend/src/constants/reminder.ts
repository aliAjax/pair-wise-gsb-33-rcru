import type { ReminderFrequency, ReminderStatus } from '@/types/api'

// CareReminder 频率/状态枚举，与后端 internal/constants/reminder.go、
// model/care_reminder.go、util/formatters.go、日志模板、错误码保持镜像。
export const ReminderFrequencyMap: Record<ReminderFrequency, string> = {
  once: '单次',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  yearly: '每年',
}

export const REMINDER_FREQUENCIES = Object.keys(ReminderFrequencyMap) as ReminderFrequency[]

export const ReminderStatusMap: Record<ReminderStatus, string> = {
  pending: '养护中',
  done: '已完成',
  overdue: '已逾期',
  unbound: '待确认',
}

export const REMINDER_STATUSES = Object.keys(ReminderStatusMap) as ReminderStatus[]

export function frequencyText(f: string): string {
  return ReminderFrequencyMap[f as ReminderFrequency] || f || '-'
}

export function statusText(s: string): string {
  return ReminderStatusMap[s as ReminderStatus] || s
}

export function statusTagType(s: string): 'warning' | 'success' | 'danger' | 'info' {
  switch (s) {
    case 'pending':
      return 'warning'
    case 'done':
      return 'success'
    case 'overdue':
      return 'danger'
    case 'unbound':
      return 'info'
    default:
      return 'info'
  }
}
