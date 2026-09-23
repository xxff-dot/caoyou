<template>
  <div>
    <el-card shadow="never" class="new-card">
      <template #header>
        <div class="mb-head">
          <div>
            <div class="mb-title">创建邮箱地址</div>
            <div class="mb-sub">创建后即可通过 自建SMTP 或 网页 收取发往该地址的邮件</div>
          </div>
        </div>
      </template>
      <div class="new-row">
        <el-input v-model="local" placeholder="邮箱前缀，如 zhangsan" style="width:260px" size="large" @keyup.enter="create">
          <template #append>@{{ store.domain }}</template>
        </el-input>
        <el-button type="primary" size="large" :loading="creating" @click="create">创建</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>我的邮箱地址</template>
      <el-table :data="locals" v-loading="loading">
        <el-table-column label="地址" min-width="240">
          <template #default="{ row }"><b>{{ row.address }}</b></template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column width="90" label="操作">
          <template #default="{ row }">
            <el-button link type="danger" @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'
import { store } from '../store'

const mailboxes = ref([])
const local = ref('')
const creating = ref(false)
const loading = ref(false)
const locals = computed(() => mailboxes.value.filter((m) => m.type === 'local'))

async function load() {
  loading.value = true
  try {
    mailboxes.value = await api.get('/mailboxes')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function create() {
  if (!local.value.trim()) {
    ElMessage.warning('请输入前缀')
    return
  }
  creating.value = true
  try {
    await api.post('/mailboxes', { local: local.value.trim() })
    ElMessage.success('创建成功')
    local.value = ''
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    creating.value = false
  }
}

async function del(row) {
  await ElMessageBox.confirm(`确定删除 ${row.address} 及其全部邮件？`, '危险操作', { type: 'warning' })
  await api.del(`/mailboxes/${row.id}`)
  ElMessage.success('已删除')
  load()
}

function fmtTime(s) {
  return new Date(s).toLocaleString('zh-CN')
}
</script>

<style scoped>
.new-card { margin-bottom: 16px; }
.new-row { display: flex; gap: 12px; }
.mb-title { font-size: 16px; font-weight: 700; }
.mb-sub { font-size: 12px; color: var(--cy-text-3); margin-top: 2px; }
</style>
