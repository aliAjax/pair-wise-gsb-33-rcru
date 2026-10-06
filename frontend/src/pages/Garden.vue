<template>
  <div class="page">
    <h1>我的花园</h1>
    <el-row :gutter="16">
      <el-col :xs="24" :md="16">
        <el-card v-loading="loading">
          <template #header>花园清单（{{ gardenItems.length }} 盆）</template>
          <el-table :data="gardenItems" empty-text="花园还是空的，去品种库添加吧">
            <el-table-column label="植物 / 来源" min-width="200">
              <template #default="{ row }">
                <div class="garden-name">{{ row.display_label || row.nickname || row.plant_name || `品种#${row.plant_species_id}` }}</div>
                <div class="garden-meta">
                  <el-tag size="small" effect="plain">{{ PlantTypeMap[row.plant_type as PlantType] || '植物' }}</el-tag>
                  <span v-if="row.family || row.genus">{{ row.family }} · {{ row.genus }}</span>
                  <span v-if="row.origin">原产地：{{ row.origin }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="位置" prop="location" width="110" />
            <el-table-column label="拥有时间" width="120">
              <template #default="{ row }">{{ formatDate(row.owned_since) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default>
                <el-tag size="small" type="success">养护中</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="{ row }">
                <el-button size="small" type="danger" :loading="removingId === row.id" @click="remove(row.id)">
                  移出
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <AwaitingReminders
          class="block"
          :groups="awaitingGroups"
          @cancel="onCancelAwaiting"
          @transfer="onTransferAwaiting"
        />

        <el-card class="block">
          <template #header>我的养护提醒</template>
          <ReminderList
            :reminders="reminders"
            :completing-id="completingId"
            @done="markDone"
            @remove="removeReminder"
            @edit="editReminder"
          />
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card>
          <template #header>我的收藏</template>
          <el-table :data="favorites" empty-text="暂无收藏">
            <el-table-column prop="target_type" label="类型" width="80">
              <template #default="{ row }">{{ FavoriteTargetTypeMap[row.target_type as FavoriteTargetType] }}</template>
            </el-table-column>
            <el-table-column prop="target_id" label="目标 ID" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ReminderList from '@/components/common/ReminderList.vue'
import AwaitingReminders from '@/components/common/AwaitingReminders.vue'
import { listGardens, removeGarden } from '@/api/garden'
import { listFavorites } from '@/api/favorite'
import {
  listReminders,
  deleteReminder,
  completeReminder,
  updateReminder,
  listAwaitingReminders,
  cancelAwaitingReminders,
  transferAwaitingReminders,
} from '@/api/reminder'
import { FavoriteTargetTypeMap, type Favorite, type FavoriteTargetType } from '@/constants/favorite'
import { PlantTypeMap, type PlantType } from '@/constants/plant'
import { formatDate } from '@/utils/dateFormat'
import type { CareReminder, RemovedGardenGroup, UserGarden } from '@/types/api'

const gardenItems = ref<UserGarden[]>([])
const favorites = ref<Favorite[]>([])
const reminders = ref<CareReminder[]>([])
const awaitingGroups = ref<RemovedGardenGroup[]>([])
const loading = ref(false)
const completingId = ref<number | null>(null)
const removingId = ref<number | null>(null)

async function refresh() {
  loading.value = true
  try {
    const [g, f, r, a] = await Promise.all([
      listGardens(),
      listFavorites(),
      listReminders(),
      listAwaitingReminders(),
    ])
    gardenItems.value = g
    favorites.value = f
    reminders.value = r
    awaitingGroups.value = a
  } finally {
    loading.value = false
  }
}

onMounted(refresh)

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('移出后未完成的提醒将停在「待确认」，可取消或转给同品种的另一盆。', '移出该植物？', {
      type: 'warning',
    })
  } catch {
    return
  }
  removingId.value = id
  try {
    const res = await removeGarden(id)
    await refresh()
    const n = res?.awaiting_reminders ?? 0
    ElMessage.success(n > 0 ? `已移出，${n} 条提醒待确认` : '已移出')
  } finally {
    removingId.value = null
  }
}

async function markDone(id: number) {
  completingId.value = id
  try {
    // Idempotent: server guards double windows / double clicks; repeated tap
    // comes back processed=false without generating a second next occurrence.
    const result = await completeReminder(id)
    reminders.value = await listReminders()
    awaitingGroups.value = await listAwaitingReminders()
    ElMessage.success(result.processed ? '已完成，下一期已生成' : result.message || '该提醒已处理')
  } finally {
    completingId.value = null
  }
}

async function editReminder(payload: { id: number; remind_date: string; frequency: string; task_title?: string }) {
  await updateReminder(payload.id, {
    task_title: payload.task_title,
    remind_date: payload.remind_date,
    frequency: payload.frequency,
  })
  reminders.value = await listReminders()
  ElMessage.success('排期已更新，下一期已重新计算')
}

async function removeReminder(id: number) {
  await deleteReminder(id)
  reminders.value = await listReminders()
  ElMessage.success('已删除提醒')
}

async function onCancelAwaiting(gardenId: number) {
  await cancelAwaitingReminders(gardenId)
  await refresh()
  ElMessage.success('待确认提醒已取消')
}

async function onTransferAwaiting(gardenId: number, targetGardenId: number) {
  await transferAwaitingReminders(gardenId, targetGardenId)
  await refresh()
  ElMessage.success('提醒已转移给同品种另一盆')
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.block { margin-top: 16px; }
.garden-name { font-weight: 600; }
.garden-meta { margin-top: 4px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap; color: #909399; font-size: 12px; }
</style>
