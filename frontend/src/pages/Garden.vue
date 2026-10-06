<template>
  <div class="page">
    <h1>我的花园</h1>
    <el-row :gutter="16">
      <el-col :xs="24" :md="16">
        <el-card>
          <template #header>花园清单</template>
          <el-table :data="gardenItems" empty-text="花园还是空的，去品种库添加吧" row-key="id">
            <el-table-column label="植物" min-width="180">
              <template #default="{ row }">
                <div class="plant-cell">
                  <span class="garden-name">{{ row.nickname || row.plant_name || `盆栽#${row.id}` }}</span>
                  <span class="species">品种：{{ row.plant_name || `#${row.plant_species_id}` }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="来源" width="140">
              <template #default="{ row }">
                <el-tag v-if="row.plant_type" size="small" effect="plain">{{ PlantTypeMap[row.plant_type as PlantType] || row.plant_type }}</el-tag>
                <div class="origin">{{ row.origin || '来源未知' }}</div>
              </template>
            </el-table-column>
            <el-table-column label="位置" prop="location" width="110" />
            <el-table-column label="拥有时间" width="110">
              <template #default="{ row }">{{ formatDate(row.owned_since) }}</template>
            </el-table-column>
            <el-table-column label="养护状态" width="150">
              <template #default="{ row }">
                <el-tag size="small" type="warning">待办 {{ row.pending_reminder_count ?? 0 }}</el-tag>
                <el-tag v-if="row.unbound_reminder_count" size="small" type="info" class="unbound-tag">
                  待确认 {{ row.unbound_reminder_count }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="170">
              <template #default="{ row }">
                <el-button size="small" type="primary" @click="openCreateReminder(row)">提醒</el-button>
                <el-button size="small" type="danger" @click="remove(row)">移出</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-card class="block">
          <template #header>
            <div class="card-header">
              <span>我的养护提醒</span>
              <el-radio-group v-model="statusFilter" size="small" @change="reloadReminders">
                <el-radio-button label="">全部</el-radio-button>
                <el-radio-button label="unbound">待确认</el-radio-button>
                <el-radio-button label="pending">养护中</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <ReminderList :reminders="reminders" @remove="removeReminder" @changed="reloadAll" />
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

    <!-- 给某一盆创建提醒：按频率只占一个下一期位置 -->
    <el-dialog v-model="createVisible" title="添加养护提醒" width="420px">
      <el-form label-width="80px">
        <el-form-item label="盆栽">
          <el-input :model-value="createForm.gardenLabel" disabled />
        </el-form-item>
        <el-form-item label="任务">
          <el-input v-model="createForm.task_title" placeholder="如：浇水、施肥" />
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="createForm.remind_date" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="频率">
          <el-select v-model="createForm.frequency" style="width: 100%">
            <el-option v-for="f in REMINDER_FREQUENCIES" :key="f" :label="ReminderFrequencyMap[f]" :value="f" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitReminder">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ReminderList from '@/components/common/ReminderList.vue'
import { listGardens, removeGarden } from '@/api/garden'
import { listFavorites } from '@/api/favorite'
import { createReminder, deleteReminder, listReminders } from '@/api/reminder'
import { FavoriteTargetTypeMap, type Favorite, type FavoriteTargetType } from '@/constants/favorite'
import { PlantTypeMap, type PlantType } from '@/constants/plant'
import { REMINDER_FREQUENCIES, ReminderFrequencyMap } from '@/constants/reminder'
import { formatDate } from '@/utils/dateFormat'
import type { CareReminder, UserGarden } from '@/types/api'

const gardenItems = ref<UserGarden[]>([])
const favorites = ref<Favorite[]>([])
const reminders = ref<CareReminder[]>([])
const statusFilter = ref('')

onMounted(reloadAll)

async function reloadAll() {
  const [gardens, favs] = await Promise.all([listGardens(), listFavorites()])
  gardenItems.value = gardens
  favorites.value = favs
  await reloadReminders()
}

async function reloadReminders() {
  reminders.value = await listReminders(statusFilter.value || undefined)
}

// 移出植物：未完成提醒停在待确认；有同品种另一盆时提示可转养。保存失败时
// 后端事务回滚，原花园关系与提醒绑定都会恢复。
async function remove(row: UserGarden) {
  try {
    await ElMessageBox.confirm(
      `确定将「${row.nickname || row.plant_name || row.id}」移出花园吗？未完成的养护提醒会停在“待确认”，可取消或转养给同品种的另一盆。`,
      '移出植物',
      { type: 'warning', confirmButtonText: '移出' },
    )
  } catch {
    return
  }
  const res = await removeGarden(row.id)
  if (res.transfer_targets?.length) {
    try {
      await ElMessageBox.confirm(
        `有 ${res.transfer_targets.length} 盆同品种植物可接收待确认提醒（如「${res.transfer_targets[0].nickname || res.transfer_targets[0].plant_name || '#' + res.transfer_targets[0].id}」）。是否立即在“待确认”列表中转养或取消？`,
        res.message,
        { confirmButtonText: '去处理', cancelButtonText: '稍后', type: 'info' },
      )
      statusFilter.value = 'unbound'
    } catch {
      // 用户选择稍后处理，提醒仍停在待确认状态
    }
  } else {
    ElMessage.success('植物已移出，未完成提醒已停在待确认')
  }
  await reloadAll()
  await reloadReminders()
}

async function removeReminder(id: number) {
  await deleteReminder(id)
  await reloadAll()
}

// ---- 给某一盆添加提醒 ----
const createVisible = ref(false)
const creating = ref(false)
const createForm = reactive<{
  gardenId: number
  gardenLabel: string
  plantSpeciesId: number
  task_title: string
  remind_date: string
  frequency: string
}>({
  gardenId: 0,
  gardenLabel: '',
  plantSpeciesId: 0,
  task_title: '',
  remind_date: '',
  frequency: 'once',
})

function openCreateReminder(row: UserGarden) {
  createForm.gardenId = row.id
  createForm.gardenLabel = row.nickname || row.plant_name || `盆栽#${row.id}`
  createForm.plantSpeciesId = row.plant_species_id
  createForm.task_title = ''
  createForm.remind_date = formatDate(new Date())
  createForm.frequency = 'once'
  createVisible.value = true
}

async function submitReminder() {
  if (!createForm.task_title || !createForm.remind_date) {
    ElMessage.warning('请填写任务与日期')
    return
  }
  creating.value = true
  try {
    await createReminder({
      garden_id: createForm.gardenId,
      plant_species_id: createForm.plantSpeciesId,
      task_title: createForm.task_title,
      remind_date: createForm.remind_date,
      frequency: createForm.frequency,
    })
    ElMessage.success('养护提醒已创建')
    createVisible.value = false
    await reloadAll()
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
.block { margin-top: 16px; }
.garden-name { font-weight: 600; }
.plant-cell { display: flex; flex-direction: column; gap: 2px; }
.species, .origin { color: #999; font-size: 12px; }
.unbound-tag { margin-left: 6px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
</style>
