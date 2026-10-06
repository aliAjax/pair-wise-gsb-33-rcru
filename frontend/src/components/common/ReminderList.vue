<template>
  <el-table :data="reminders" stripe empty-text="暂无养护提醒" row-key="id">
    <el-table-column label="任务" min-width="200">
      <template #default="{ row }">
        <div class="task-cell">
          <span class="task-title">{{ row.task_title }}</span>
          <div v-if="showSource" class="source">
            <el-tag size="small" effect="plain">{{ row.plant_name || `品种#${row.plant_species_id}` }}</el-tag>
            <el-tag v-if="row.garden_name" size="small" type="success" effect="plain">🪴 {{ row.garden_name }}</el-tag>
            <el-tag v-else-if="row.status === 'unbound'" size="small" type="info" effect="plain">已移出花园</el-tag>
          </div>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="提醒日期" width="120">
      <template #default="{ row }">{{ formatDate(row.remind_date) }}</template>
    </el-table-column>
    <el-table-column label="频率" width="90">
      <template #default="{ row }">{{ frequencyText(row.frequency) }}</template>
    </el-table-column>
    <el-table-column label="状态" width="100">
      <template #default="{ row }">
        <el-tag :type="statusTagType(row.status)">{{ statusText(row.status) }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="230">
      <template #default="{ row }">
        <template v-if="row.status === 'pending' || row.status === 'overdue'">
          <el-button size="small" type="success" :loading="completingId === row.id" @click="markDone(row)">完成</el-button>
          <el-button size="small" @click="openEdit(row)">改期</el-button>
          <el-button size="small" type="danger" @click="$emit('remove', row.id)">删除</el-button>
        </template>
        <template v-else-if="row.status === 'unbound'">
          <el-button size="small" type="warning" @click="openTransfer(row)">转养</el-button>
          <el-button size="small" @click="cancel(row)">取消</el-button>
        </template>
        <template v-else>
          <el-button size="small" type="danger" @click="$emit('remove', row.id)">删除</el-button>
        </template>
      </template>
    </el-table-column>

    <!-- 改期 / 改频率：下一期失效并按新计划重算 -->
    <el-dialog v-model="editVisible" title="修改养护计划（下一期将重算）" width="420px">
      <el-form label-width="80px">
        <el-form-item label="任务">
          <el-input v-model="editForm.task_title" />
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="editForm.remind_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="频率">
          <el-select v-model="editForm.frequency" style="width: 100%">
            <el-option v-for="f in REMINDER_FREQUENCIES" :key="f" :label="ReminderFrequencyMap[f]" :value="f" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingEdit" @click="saveEdit">保存并重算</el-button>
      </template>
    </el-dialog>

    <!-- 转养给同品种的另一盆 -->
    <el-dialog v-model="transferVisible" title="转养给同品种的另一盆" width="420px">
      <p v-if="!transferTargets.length" class="muted">没有其他同品种盆栽，可直接取消该提醒。</p>
      <el-radio-group v-else v-model="transferTargetId" class="target-group">
        <el-radio v-for="g in transferTargets" :key="g.id" :label="g.id" class="target-item">
          {{ g.nickname || g.plant_name || `盆栽#${g.id}` }}
          <span class="muted">（{{ g.location || '未设置位置' }}）</span>
        </el-radio>
      </el-radio-group>
      <template #footer>
        <el-button @click="transferVisible = false">关闭</el-button>
        <el-button v-if="transferTargets.length" type="primary" :loading="transferring" :disabled="!transferTargetId" @click="confirmTransfer">转养</el-button>
      </template>
    </el-dialog>
  </el-table>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/dateFormat'
import {
  REMINDER_FREQUENCIES,
  ReminderFrequencyMap,
  frequencyText,
  statusTagType,
  statusText,
} from '@/constants/reminder'
import {
  cancelReminder,
  listTransferTargets,
  transferReminder,
  updateReminder,
  updateReminderStatus,
} from '@/api/reminder'
import type { CareReminder, ReminderFrequency } from '@/types/api'

withDefaults(defineProps<{ reminders: CareReminder[]; showSource?: boolean }>(), {
  showSource: true,
})
const emit = defineEmits<{
  (e: 'done', id: number): void
  (e: 'remove', id: number): void
  (e: 'changed'): void
}>()

const completingId = ref<number | null>(null)
async function markDone(row: CareReminder) {
  completingId.value = row.id
  try {
    const res = await updateReminderStatus(row.id, 'done')
    // processed=true 即重复完成：已处理，不会再生成第二份下一期
    ElMessage.success(res.message)
    emit('done', row.id)
    emit('changed')
  } finally {
    completingId.value = null
  }
}

async function cancel(row: CareReminder) {
  try {
    await ElMessageBox.confirm('取消后该待确认提醒将被删除，确定吗？', '取消待确认提醒', { type: 'warning' })
  } catch {
    return
  }
  const res = await cancelReminder(row.id)
  ElMessage.success(res.message)
  emit('changed')
}

// ---- 改期 / 改频率 ----
const editVisible = ref(false)
const savingEdit = ref(false)
const editForm = reactive<{ id: number; task_title: string; remind_date: string; frequency: ReminderFrequency | string }>({
  id: 0,
  task_title: '',
  remind_date: '',
  frequency: 'once',
})
function openEdit(row: CareReminder) {
  editForm.id = row.id
  editForm.task_title = row.task_title
  editForm.remind_date = formatDate(row.remind_date)
  editForm.frequency = row.frequency || 'once'
  editVisible.value = true
}
async function saveEdit() {
  savingEdit.value = true
  try {
    await updateReminder(editForm.id, {
      task_title: editForm.task_title,
      remind_date: editForm.remind_date,
      frequency: editForm.frequency,
    })
    ElMessage.success('计划已更新，下一期已重算')
    editVisible.value = false
    emit('changed')
  } finally {
    savingEdit.value = false
  }
}

// ---- 转养 ----
const transferVisible = ref(false)
const transferring = ref(false)
const transferTargets = ref<{ id: number; nickname?: string; plant_name?: string; location?: string }[]>([])
const transferTargetId = ref<number>(0)
const transferReminderId = ref<number>(0)

async function openTransfer(row: CareReminder) {
  transferReminderId.value = row.id
  transferTargetId.value = 0
  transferTargets.value = await listTransferTargets(row.id)
  transferVisible.value = true
}
async function confirmTransfer() {
  transferring.value = true
  try {
    const res = await transferReminder(transferReminderId.value, transferTargetId.value)
    ElMessage.success(res.message)
    transferVisible.value = false
    emit('changed')
  } finally {
    transferring.value = false
  }
}
</script>

<style scoped>
.task-cell { display: flex; flex-direction: column; gap: 4px; }
.task-title { font-weight: 600; }
.source { display: flex; gap: 6px; flex-wrap: wrap; }
.muted { color: #999; font-size: 12px; }
.target-group { display: flex; flex-direction: column; gap: 8px; }
.target-item { margin-right: 0; }
</style>
