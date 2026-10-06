import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  listReminders,
  createReminder,
  updateReminder,
  completeReminder,
  deleteReminder,
  listAwaitingReminders,
  cancelAwaitingReminders,
  transferAwaitingReminders,
} from '@/api/reminder'
import type { CareReminder, RemovedGardenGroup } from '@/types/api'

export const useReminderStore = defineStore('reminder', () => {
  const reminders = ref<CareReminder[]>([])
  const awaiting = ref<RemovedGardenGroup[]>([])

  async function load(status?: string) {
    reminders.value = await listReminders(status)
  }

  async function loadAwaiting() {
    awaiting.value = await listAwaitingReminders()
  }

  async function create(payload: {
    plant_species_id?: number
    garden_id?: number
    task_title: string
    remind_date: string
    frequency?: string
  }) {
    await createReminder(payload)
    await load()
  }

  // Idempotent completion; returns the server result so the UI can distinguish
  // "marked done" from "already processed (next slot already generated)".
  async function complete(id: number) {
    const result = await completeReminder(id)
    await load()
    return result
  }

  // Changing date/frequency invalidates the next occurrence server-side.
  async function update(
    id: number,
    payload: { task_title?: string; remind_date?: string; frequency?: string },
  ) {
    const result = await updateReminder(id, payload)
    await load()
    return result
  }

  async function setStatus(id: number, status: string) {
    await completeReminder(id)
    await load()
  }

  async function remove(id: number) {
    await deleteReminder(id)
    await load()
  }

  async function cancelAwaiting(gardenId: number) {
    const res = await cancelAwaitingReminders(gardenId)
    await Promise.all([load(), loadAwaiting()])
    return res
  }

  async function transferAwaiting(gardenId: number, targetGardenId: number) {
    const res = await transferAwaitingReminders(gardenId, targetGardenId)
    await Promise.all([load(), loadAwaiting()])
    return res
  }

  return {
    reminders,
    awaiting,
    load,
    loadAwaiting,
    create,
    complete,
    update,
    setStatus,
    remove,
    cancelAwaiting,
    transferAwaiting,
  }
})
