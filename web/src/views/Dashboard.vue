<template>
  <div>
    <el-row :gutter="14">
      <el-col :span="6" v-for="card in cards" :key="card.label">
        <el-card shadow="never" class="kpi">
          <div class="kpi-icon" :style="{ background: card.bg }"><el-icon :size="20" color="#fff"><component :is="card.icon" /></el-icon></div>
          <div>
            <div class="kpi-num">{{ card.value }}</div>
            <div class="kpi-label">{{ card.label }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" style="margin-top:14px">
      <template #header>
        <div class="panel-head">
          <span>我的邮箱</span>
          <el-button link type="primary" @click="$router.push('/mailboxes')">管理地址 →</el-button>
        </div>
      </template>
      <div class="addr-grid">
        <div v-for="mb in mailboxes" :key="mb.id" class="addr-item" @click="goInbox(mb)">
          <div class="addr-type" :class="mb.type">{{ mb.type === 'local' ? '本地' : '外部' }}</div>
          <div class="addr-text">
            <div class="addr-address">{{ mb.address }}</div>
            <div class="addr-status" :class="mb.last_error ? 'bad' : 'ok'">{{ mb.last_error || '正常' }}</div>
          </div>
        </div>
        <div class="addr-item add" @click="$router.push('/mailboxes')">
          <el-icon size="22"><Plus /></el-icon>
          <span>添加邮箱地址</span>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'

const router = useRouter()
const stats = ref({})
const mailboxes = ref([])

const cards = computed(() => [
  { label: '邮箱地址', icon: 'Postcard', value: stats.value.mailboxes ?? '-', bg: 'linear-gradient(135deg,#4c6fff,#7a5cff)' },
  { label: '收件总数', icon: 'MessageBox', value: stats.value.messages ?? '-', bg: 'linear-gradient(135deg,#00a870,#4cd9a0)' },
  { label: '未读邮件', icon: 'Bell', value: stats.value.unread ?? '-', bg: 'linear-gradient(135deg,#f59e0b,#fbbf60)' },
  { label: '已发送', icon: 'Promotion', value: stats.value.sent ?? '-', bg: 'linear-gradient(135deg,#e4574c,#ff8a7a)' }
])

async function load() {
  const [st, mbs] = await Promise.all([api.get('/stats'), api.get('/mailboxes')])
  stats.value = st
  mailboxes.value = mbs
}
onMounted(load)
function goInbox() { router.push('/inbox') }
</script>

<style scoped>
.kpi { display: flex; }
.kpi :deep(.el-card__body) { display: flex; align-items: center; gap: 14px; }
.kpi-icon { width: 44px; height: 44px; border-radius: 11px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.kpi-num { font-size: 26px; font-weight: 700; line-height: 1.1; }
.kpi-label { font-size: 13px; color: var(--cy-text-2); margin-top: 3px; }
.panel-head { display: flex; justify-content: space-between; align-items: center; }
.addr-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.addr-item {
  display: flex; align-items: center; gap: 12px;
  border: 1px solid var(--cy-border); border-radius: 10px;
  padding: 14px 16px; cursor: pointer; transition: all .15s;
}
.addr-item:hover { border-color: var(--cy-primary); box-shadow: 0 2px 10px rgba(76,111,255,.1); }
.addr-type { font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 6px; flex-shrink: 0; }
.addr-type.local { background: #eef2ff; color: #4c6fff; }
.addr-type.external { background: #fff7e8; color: #d97706; }
.addr-address { font-weight: 600; font-size: 14px; }
.addr-status { font-size: 12px; margin-top: 2px; }
.addr-status.ok { color: #00a870; }
.addr-status.bad { color: #e4574c; }
.addr-item.add { justify-content: center; gap: 8px; color: var(--cy-text-2); border-style: dashed; }
.addr-item.add:hover { color: var(--cy-primary); }
</style>
