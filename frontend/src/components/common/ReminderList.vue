<template>
  <el-table :data="reminders" stripe empty-text="暂无养护提醒" row-key="id">
    <el-table-column label="任务 / 来源" min-width="220">
      <template #default="{ row }">
        <div class="r-title">{{ row.task_title }}</div>
        <div class="r-source">
          <el-tag size="small" effect="plain" type="success">{{ row.source_label || plantFallback(row) }}</el-tag>
          <el-tag
            v-if="row.garden_status === 'removed'"
            size="small"
            type="info"
            class="r-pot"
          >盆已移出</el-tag>
          <span v-if="row.location" class="r-loc">📍 {{ row.location }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="提醒日期" width="150">
      <template #default="{ row }">
        <el-date-picker
          v-if="editingId === row.id"
          v-model="editForm.remind_date"
          type="date"
          size="small"
          value-format="YYYY-MM-DD"
        />
        <span v-else>{{ formatDate(row.remind_date) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="频率" width="120">
      <template #default="{ row }">
        <el-select
          v-if="editingId === row.id"
          v-model="editForm.frequency"
          size="small"
          style="width: 88px"
        >
          <el-option v-for="f in REMINDER_FREQUENCIES" :key="f.value" :label="f.label" :value="f.value" />
        </el-select>
        <span v-else>{{ frequencyText(row.frequency) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="期号" width="80">
      <template #default="{ row }">第{{ row.seq }}期</template>
    </el-table-column>
    <el-table-column label="状态" width="100">
      <template #default="{ row }">
        <el-tag :type="statusTagType(row.status)">{{ ReminderStatusTextMap[row.status] || row.status }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="200">
      <template #default="{ row }">
        <template v-if="editingId === row.id">
          <el-button size="small" type="primary" :loading="saving" @click="saveEdit(row)">保存</el-button>
          <el-button size="small" @click="cancelEdit">取消</el-button>
        </template>
        <template v-else>
          <el-button
            v-if="row.status === 'pending' || row.status === 'overdue'"
            size="small"
            type="success"
            :loading="props.completingId === row.id"
            @click="$emit('done', row.id)"
          >完成</el-button>
          <el-button
            v-if="row.status !== 'awaiting_confirm'"
            size="small"
            @click="startEdit(row)"
          >改期</el-button>
          <el-button size="small" type="danger" @click="$emit('remove', row.id)">删除</el-button>
        </template>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { CareReminder } from '@/types/api'
import { formatDate } from '@/utils/dateFormat'
import {
  REMINDER_FREQUENCIES,
  ReminderStatusTextMap,
  frequencyText,
  statusTagType,
} from '@/constants/reminder'

const props = defineProps<{
  reminders: CareReminder[]
  completingId?: number
}>()
const emit = defineEmits<{
  (e: 'done', id: number): void
  (e: 'remove', id: number): void
  (e: 'edit', payload: { id: number; task_title?: string; remind_date?: string; frequency?: string }): void
}>()

const editingId = ref<number | null>(null)
const saving = ref(false)
const editForm = reactive({ task_title: '', remind_date: '', frequency: '' })

function plantFallback(row: CareReminder): string {
  return row.plant_name ? row.plant_name : '通用任务'
}

function startEdit(row: CareReminder) {
  editingId.value = row.id
  editForm.task_title = row.task_title
  editForm.remind_date = formatDate(row.remind_date)
  editForm.frequency = row.frequency ?? ''
}

function cancelEdit() {
  editingId.value = null
}

async function saveEdit(row: CareReminder) {
  if (!editForm.remind_date) return
  saving.value = true
  try {
    emit('edit', {
      id: row.id,
      task_title: editForm.task_title,
      remind_date: editForm.remind_date,
      frequency: editForm.frequency,
    })
    editingId.value = null
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.r-title { font-weight: 600; }
.r-source { margin-top: 4px; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.r-pot { margin-left: 0; }
.r-loc { color: #909399; font-size: 12px; }
</style>
