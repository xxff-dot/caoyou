<template>
  <div>
    <el-card shadow="never" class="new-card">
      <template #header>绑定外部邮箱（IMAP/SMTP）</template>
      <el-form :model="form" label-width="110px" style="max-width:520px">
        <el-form-item label="快速填入">
          <el-select placeholder="选择服务商自动填入" style="width:100%" @change="preset">
            <el-option label="QQ邮箱" value="qq" />
            <el-option label="163邮箱" value="163" />
            <el-option label="Gmail" value="gmail" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮箱地址">
          <el-input v-model="form.address" placeholder="name@qq.com" />
        </el-form-item>
        <el-form-item label="密码/授权码">
          <el-input v-model="form.password" type="password" show-password placeholder="QQ/163需使用授权码" />
        </el-form-item>
        <el-form-item label="IMAP服务器">
          <el-input v-model="form.imap_host" placeholder="imap.qq.com" style="width:220px" />
          <el-input-number v-model="form.imap_port" :min="1" :max="65535" style="margin-left:8px;width:130px" />
        </el-form-item>
        <el-form-item label="SMTP服务器">
          <el-input v-model="form.smtp_host" placeholder="smtp.qq.com" style="width:220px" />
          <el-input-number v-model="form.smtp_port" :min="1" :max="65535" style="margin-left:8px;width:130px" />
        </el-form-item>
        <el-button type="primary" size="large" :loading="saving" @click="bind">绑 定</el-button>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>已绑定的外部邮箱</template>
      <el-table :data="externals" v-loading="loading">
        <el-table-column label="地址" min-width="220">
          <template #default="{ row }"><b>{{ row.address }}</b></template>
        </el-table-column>
        <el-table-column label="状态" min-width="220">
          <template #default="{ row }">
            <el-text v-if="row.last_error" type="danger" size="small">{{ row.last_error }}</el-text>
            <el-text v-else type="success" size="small">正常</el-text>
          </template>
        </el-table-column>
        <el-table-column width="90" label="操作">
          <template #default="{ row }">
            <el-button link type="danger" @click="del(row)">解绑</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const mailboxes = ref([])
const loading = ref(false)
const saving = ref(false)
const form = reactive({
  address: '', password: '',
  imap_host: 'imap.qq.com', imap_port: 993,
  smtp_host: 'smtp.qq.com', smtp_port: 465
})
const externals = computed(() => mailboxes.value.filter((m) => m.type === 'external'))

const presets = {
  qq: { imap_host: 'imap.qq.com', imap_port: 993, smtp_host: 'smtp.qq.com', smtp_port: 465 },
  '163': { imap_host: 'imap.163.com', imap_port: 993, smtp_host: 'smtp.163.com', smtp_port: 465 },
  gmail: { imap_host: 'imap.gmail.com', imap_port: 993, smtp_host: 'smtp.gmail.com', smtp_port: 465 }
}

function preset(name) {
  Object.assign(form, presets[name])
}

async function load() {
  loading.value = true
  try {
    mailboxes.value = await api.get('/mailboxes')
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function bind() {
  saving.value = true
  try {
    await api.post('/external', form)
    ElMessage.success('绑定成功')
    form.address = ''
    form.password = ''
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function del(row) {
  await ElMessageBox.confirm(`解绑 ${row.address}？其已同步邮件将一并删除。`, '提示', { type: 'warning' })
  await api.del(`/mailboxes/${row.id}`)
  ElMessage.success('已解绑')
  load()
}
</script>

<style scoped>
.new-card { margin-bottom: 16px; }
</style>
