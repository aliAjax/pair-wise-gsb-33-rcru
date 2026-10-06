export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface UserInfo {
  id: number
  username: string
  email: string
  nickname: string
  avatar: string
  bio: string
  role: 'user' | 'admin'
  created_at: string
}

export type ReminderStatus = 'pending' | 'done' | 'overdue' | 'awaiting_confirm'

export type ReminderFrequency = '' | 'daily' | 'weekly' | 'monthly' | 'yearly'

export interface CareReminder {
  id: number
  user_id: number
  plant_species_id: number
  garden_id: number
  series_id: number
  seq: number
  schedule_version: number
  task_title: string
  remind_date: string
  frequency: string
  status: ReminderStatus
  created_at: string

  // joined plant source + pot status
  plant_name?: string
  plant_type?: string
  plant_alias?: string
  garden_name?: string
  location?: string
  garden_status?: 'active' | 'removed' | ''
  source_label?: string
}

export interface ReminderCompleteResult {
  reminder: CareReminder
  processed: boolean
  next_reminder_id: number
  message: string
}

export interface UserGarden {
  id: number
  user_id: number
  plant_species_id: number
  nickname: string
  owned_since: string
  location: string
  care_reminder_id: number
  status: 'active' | 'removed'
  created_at: string

  // joined plant source
  plant_name?: string
  plant_type?: string
  plant_alias?: string
  origin?: string
  family?: string
  genus?: string
  display_label?: string
}

// A removed pot together with its awaiting reminders and same-species targets.
export interface RemovedGardenGroup {
  garden: UserGarden
  reminders: CareReminder[]
  targets: UserGarden[]
}

export interface DiseasePest {
  id: number
  plant_species_id: number
  name: string
  symptoms: string
  cause: string
  treatment: string
  recommended_medicine: string
  images: string
  keywords: string
  created_at: string
}

export interface Question {
  id: number
  user_id: number
  title: string
  content: string
  images: string
  status: string
  created_at: string
}

export interface Answer {
  id: number
  question_id: number
  user_id: number
  content: string
  is_best: boolean
  like_count: number
  created_at: string
}
