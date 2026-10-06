<template>
  <div class="page">
    <h1>季节养护日历</h1>
    <el-row :gutter="16">
      <el-col :xs="24" :md="14">
        <el-timeline>
          <el-timeline-item v-for="t in seasonTasks" :key="t.month" :timestamp="t.label" :type="t.month === currentMonth ? 'primary' : ''">
            {{ t.task }}
            <el-tag v-if="t.month === currentMonth" size="small" type="success">本月</el-tag>
          </el-timeline-item>
        </el-timeline>
      </el-col>
      <el-col :xs="24" :md="10">
        <el-card>
          <template #header>本月养护提醒</template>
          <el-form inline>
            <el-form-item label="任务"><el-input v-model="form.task_title" placeholder="如：给月季施肥" /></el-form-item>
            <el-form-item label="日期"><el-date-picker v-model="form.remind_date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
            <el-form-item label="频率">
              <el-select v-model="form.frequency" style="width: 110px">
                <el-option v-for="f in REMINDER_FREQUENCIES" :key="f" :label="ReminderFrequencyMap[f]" :value="f" />
              </el-select>
            </el-form-item>
            <el-form-item><el-button type="primary" @click="create">创建提醒</el-button></el-form-item>
          </el-form>
          <ReminderList
            :reminders="reminders"
            @remove="remove"
            @changed="reload"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import ReminderList from '@/components/common/ReminderList.vue'
import { useReminderStore } from '@/stores/reminderStore'
import { getSeasonTasks } from '@/utils/season'
import { deleteReminder } from '@/api/reminder'
import { REMINDER_FREQUENCIES, ReminderFrequencyMap } from '@/constants/reminder'
import type { CareReminder } from '@/types/api'

const store = useReminderStore()
const seasonTasks = getSeasonTasks()
const currentMonth = new Date().getMonth() + 1
const reminders = ref<CareReminder[]>([])
const form = reactive<{ task_title: string; remind_date: string; frequency: string }>({
  task_title: '',
  remind_date: '',
  frequency: 'once',
})

onMounted(reload)

async function reload() {
  await store.load()
  reminders.value = store.reminders
}

async function create() {
  if (!form.task_title || !form.remind_date) {
    ElMessage.warning('请填写任务与日期')
    return
  }
  // 日历页创建的是未绑定具体盆栽的通用提醒（garden_id=0），按频率只占一个下一期位置。
  await store.create({ task_title: form.task_title, remind_date: form.remind_date, frequency: form.frequency })
  reminders.value = store.reminders
  form.task_title = ''
  form.remind_date = ''
  form.frequency = 'once'
  ElMessage.success('养护提醒已创建')
}
async function remove(id: number) {
  await deleteReminder(id)
  await reload()
  ElMessage.success('已删除提醒')
}
</script>

<style scoped>
.page { max-width: 1200px; margin: 0 auto; }
</style>
