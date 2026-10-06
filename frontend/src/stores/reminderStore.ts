import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listReminders, updateReminder, updateReminderStatus, createReminder } from '@/api/reminder'
import type { CareReminder, ReminderStatusResult } from '@/types/api'

export const useReminderStore = defineStore('reminder', () => {
  const reminders = ref<CareReminder[]>([])

  async function load(status?: string) {
    reminders.value = await listReminders(status)
  }

  async function create(payload: { plant_species_id?: number; garden_id?: number; task_title: string; remind_date: string; frequency?: string }) {
    await createReminder(payload)
    await load()
  }

  // 完成是幂等的：返回 processed=true 表示重复完成、已处理，不会再生成下一期。
  async function setStatus(id: number, status: string): Promise<ReminderStatusResult> {
    const res = await updateReminderStatus(id, status)
    await load()
    return res
  }

  // 改动日期或频率后，下一期失效重算。
  async function update(id: number, payload: { task_title?: string; remind_date?: string; frequency?: string }) {
    await updateReminder(id, payload)
    await load()
  }

  return { reminders, load, create, setStatus, update }
})
