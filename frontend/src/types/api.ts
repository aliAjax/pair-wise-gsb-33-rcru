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

export type ReminderStatus = 'pending' | 'done' | 'overdue' | 'unbound'
export type ReminderFrequency = 'once' | 'daily' | 'weekly' | 'monthly' | 'yearly'

export interface CareReminder {
  id: number
  user_id: number
  plant_species_id: number
  garden_id: number
  task_title: string
  remind_date: string
  frequency: ReminderFrequency | string
  status: ReminderStatus
  series_key: string
  schedule_version: number
  created_at: string
  // 关联展示字段：来源植物与所在盆栽
  plant_name?: string
  origin?: string
  plant_type?: string
  garden_name?: string
}

export interface ReminderStatusResult {
  reminder: CareReminder
  processed: boolean
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
  created_at: string
  // 关联展示字段：植物来源与养护状态
  plant_name?: string
  alias?: string
  plant_type?: string
  origin?: string
  water_frequency?: string
  image_urls?: string
  pending_reminder_count?: number
  unbound_reminder_count?: number
}

export interface GardenRemoveResult {
  removed: boolean
  message: string
  transfer_targets: UserGarden[]
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
