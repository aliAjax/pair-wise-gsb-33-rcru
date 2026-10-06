<template>
  <el-card class="block awaiting-card">
    <template #header>
      <div class="hdr">
        <span>待确认提醒（植物已移出）</span>
        <el-tag type="info">{{ groups.length }} 盆</el-tag>
      </div>
    </template>
    <el-empty v-if="!groups.length" description="没有待确认的提醒" :image-size="60" />
    <el-collapse v-else v-model="active">
      <el-collapse-item v-for="g in groups" :key="g.garden.id" :name="g.garden.id">
        <template #title>
          <div class="pot-line">
            <el-tag type="info" size="small">已移出</el-tag>
            <span class="pot-name">{{ g.garden.display_label || g.garden.nickname || g.garden.plant_name }}</span>
            <el-tag size="small" type="warning">{{ g.reminders.length }} 条待确认</el-tag>
          </div>
        </template>
        <el-table :data="g.reminders" size="small" border>
          <el-table-column prop="task_title" label="任务" min-width="140" />
          <el-table-column label="日期" width="110">
            <template #default="{ row }">{{ formatDate(row.remind_date) }}</template>
          </el-table-column>
          <el-table-column label="频率" width="80">
            <template #default="{ row }">{{ frequencyText(row.frequency) }}</template>
          </el-table-column>
        </el-table>
        <div class="actions">
          <el-button size="small" type="danger" @click="emit('cancel', g.garden.id)">
            取消这些提醒
          </el-button>
          <el-select
            :model-value="targetMap[g.garden.id] ?? ''"
            size="small"
            :placeholder="g.targets.length ? '转给同品种另一盆' : '无同品种其他盆'"
            :disabled="!g.targets.length"
            style="width: 220px"
            @update:model-value="(v) => pickTarget(g.garden.id, v)"
          >
            <el-option
              v-for="t in g.targets"
              :key="t.id"
              :label="t.display_label || t.nickname || t.plant_name"
              :value="t.id"
            />
          </el-select>
          <el-button
            size="small"
            type="primary"
            :disabled="!targetMap[g.garden.id]"
            @click="emit('transfer', g.garden.id, targetMap[g.garden.id])"
          >
            转移
          </el-button>
        </div>
      </el-collapse-item>
    </el-collapse>
  </el-card>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { RemovedGardenGroup } from '@/types/api'
import { formatDate } from '@/utils/dateFormat'
import { frequencyText } from '@/constants/reminder'

const props = defineProps<{ groups: RemovedGardenGroup[] }>()
const emit = defineEmits<{
  (e: 'cancel', gardenId: number): void
  (e: 'transfer', gardenId: number, targetGardenId: number): void
}>()

const active = ref<number[]>([])
const targetMap = reactive<Record<number, number>>({})

watch(
  () => props.groups,
  (gs) => {
    active.value = gs.map((g) => g.garden.id)
  },
  { immediate: true },
)

function pickTarget(gardenId: number, value: number | string) {
  targetMap[gardenId] = Number(value)
}
</script>

<style scoped>
.awaiting-card { margin-top: 16px; }
.hdr { display: flex; align-items: center; gap: 8px; }
.pot-line { display: flex; align-items: center; gap: 8px; }
.pot-name { font-weight: 600; }
.actions { margin-top: 10px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
</style>
