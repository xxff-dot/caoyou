<template>
  <div>
    <el-row :gutter="16" style="margin-bottom:16px">
      <el-col :span="4" v-for="card in cards" :key="card.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-num">{{ card.value }}</div>
          <div class="stat-label">{{ card.label }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <el-tabs v-model="tab">
        <el-tab-pane label="用户管理" name="users">
          <div class="toolbar-row">
            <el-input v-model="userQ" placeholder="搜索用户名" clearable style="width:220px" @keyup.enter="loadUsers" />
            <el-button @click="loadUsers">搜索</el-button>
          </div>
          <el-table :data="users" v-loading="loadingUsers" size="large">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="username" label="用户名" min-width="140">
              <template #default="{ row }">
                {{ row.username }}
                <el-tag v-if="row.is_admin" size="small" type="warning">管理员</el-tag>
                <el-tag v-if="row.disabled" size="small" type="danger">已禁用</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="mailbox_count" label="邮箱数" width="90" />
            <el-table-column label="注册时间" width="170">
              <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="230">
              <template #default="{ row }">
                <el-button link :type="row.disabled ? 'success' : 'warning'" @click="toggleDisable(row)">
                  {{ row.disabled ? '启用' : '禁用' }}
                </el-button>
                <el-button link type="primary" @click="openReset(row)">重置密码</el-button>
                <el-button link type="danger" @click="delUser(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination layout="total, prev, pager, next" :total="userTotal" :page-size="userSize"
            v-model:current-page="userPage" @current-change="loadUsers" style="margin-top:14px; justify-content:flex-end" />
        </el-tab-pane>

        <el-tab-pane label="邮箱管理" name="mailboxes">
          <el-table :data="mailboxes" v-loading="loadingMbs" size="large">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column label="地址" min-width="200">
              <template #default="{ row }"><b>{{ row.address }}</b></template>
            </el-table-column>
            <el-table-column label="类型" width="90">
              <template #default="{ row }">
                <el-tag effect="plain" :type="row.type === 'local' ? 'primary' : 'warning'">
                  {{ row.type === 'local' ? '本地' : '外部' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="owner" label="属主" width="140" />
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }">{{ fmtTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button link type="danger" @click="delMailbox(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination layout="total, prev, pager, next" :total="mbTotal" :page-size="mbSize"
            v-model:current-page="mbPage" @current-change="loadMailboxes" style="margin-top:14px; justify-content:flex-end" />
        </el-tab-pane>

        <el-tab-pane label="系统统计" name="stats">
          <el-descriptions :column="2" border size="large" style="max-width:560px">
            <el-descriptions-item label="总用户数">{{ stats.users }}</el-descriptions-item>
            <el-descriptions-item label="总邮箱数">{{ stats.mailboxes }}</el-descriptions-item>
            <el-descriptions-item label="收件总量">{{ stats.messages }}</el-descriptions-item>
            <el-descriptions-item label="发信总量">{{ stats.sent }}</el-descriptions-item>
            <el-descriptions-item label="今日新增邮件">{{ stats.today }}</el-descriptions-item>
          </el-descriptions>
          <el-alert type="info" :closable="false" style="margin-top:16px"
            title="查看任意邮箱的邮件：请到「收件箱」页面，管理员模式下邮箱下拉框会列出全站所有邮箱。" />
        </el-tab-pane>

        <el-tab-pane label="发信设置" name="mail">
          <el-form :model="ms" label-width="140px" style="max-width:640px" v-loading="loadingMs">
            <el-form-item label="邮件域名">
              <el-input v-model="ms.domain" placeholder="example.com" />
              <div class="field-tip">保存后，所有本地邮箱地址会自动改为新域名后缀（外部邮箱不受影响；新地址已被占用时该地址跳过）。</div>
            </el-form-item>
            <el-form-item label="HELO域名">
              <el-input v-model="ms.helo_domain" placeholder="默认同邮件域名" />
            </el-form-item>
            <el-form-item label="发信模式">
              <el-radio-group v-model="ms.mode">
                <el-radio value="relay">relay 经外部SMTP发信（推荐）</el-radio>
                <el-radio value="direct">direct 解析MX直投</el-radio>
              </el-radio-group>
            </el-form-item>

            <template v-if="ms.mode === 'relay'">
              <el-form-item label="SMTP服务器">
                <el-input v-model="ms.relay_host" placeholder="smtp.qq.com" style="width:240px" />
                <el-input-number v-model="ms.relay_port" :min="1" :max="65535" style="margin-left:8px;width:130px" />
              </el-form-item>
              <el-form-item label="加密方式">
                <el-radio-group v-model="ms.relay_tls">
                  <el-radio value="ssl">SSL(465)</el-radio>
                  <el-radio value="starttls">STARTTLS(587)</el-radio>
                  <el-radio value="none">无加密</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="账号">
                <el-input v-model="ms.relay_user" placeholder="yourname@qq.com" />
              </el-form-item>
              <el-form-item label="密码/授权码">
                <el-input v-model="ms.relay_pass" type="password" show-password
                  :placeholder="ms.has_relay_pass ? '已保存，留空则不修改' : '授权码'" />
              </el-form-item>
            </template>

            <el-form-item label="DKIM签名">
              <el-switch v-model="ms.dkim_enabled" />
            </el-form-item>
            <template v-if="ms.dkim_enabled">
              <el-form-item label="DKIM Selector">
                <el-input v-model="ms.dkim_selector" placeholder="caoyou" style="width:240px" />
              </el-form-item>
              <el-form-item label="">
                <el-button @click="genDKIM" :loading="genning">生成新密钥对</el-button>
                <el-tag v-if="ms.has_dkim_key" type="success" style="margin-left:10px">已保存密钥</el-tag>
              </el-form-item>
              <el-form-item v-if="dkimDns" label="DNS记录">
                <div class="dkim-dns">
                  <div>添加TXT记录，名称：</div>
                  <code>{{ dkimDns.name }}</code>
                  <div>内容：</div>
                  <code class="dns-txt">{{ dkimDns.txt }}</code>
                </div>
              </el-form-item>
            </template>

            <el-form-item>
              <el-button type="primary" size="large" :loading="savingMs" @click="saveMs">保存设置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>


    <el-dialog v-model="resetVisible" title="重置密码" width="400px">
      <el-input v-model="resetPassword" placeholder="新密码（至少6位）" show-password />
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" @click="doReset">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const tab = ref('users')
const stats = ref({})
const cards = computed(() => [
  { label: '用户', value: stats.value.users ?? '-' },
  { label: '邮箱', value: stats.value.mailboxes ?? '-' },
  { label: '收件', value: stats.value.messages ?? '-' },
  { label: '发信', value: stats.value.sent ?? '-' },
  { label: '今日新增', value: stats.value.today ?? '-' }
])

const users = ref([])
const userTotal = ref(0)
const userPage = ref(1)
const userSize = 20
const userQ = ref('')
const loadingUsers = ref(false)

async function loadUsers() {
  loadingUsers.value = true
  try {
    const r = await api.get('/admin/users', { page: userPage.value, size: userSize, q: userQ.value })
    users.value = r.items || []
    userTotal.value = r.total
  } finally {
    loadingUsers.value = false
  }
}

const mailboxes = ref([])
const mbTotal = ref(0)
const mbPage = ref(1)
const mbSize = 20
const loadingMbs = ref(false)

async function loadMailboxes() {
  loadingMbs.value = true
  try {
    const r = await api.get('/admin/mailboxes', { page: mbPage.value, size: mbSize })
    mailboxes.value = r.items || []
    mbTotal.value = r.total
  } finally {
    loadingMbs.value = false
  }
}

async function toggleDisable(row) {
  await api.post(`/admin/users/${row.id}/disable`, { disabled: !row.disabled })
  ElMessage.success(row.disabled ? '已启用' : '已禁用')
  loadUsers()
}

const resetVisible = ref(false)
const resetPassword = ref('')
let resetUser = null

function openReset(row) {
  resetUser = row
  resetPassword.value = ''
  resetVisible.value = true
}

async function doReset() {
  if (resetPassword.value.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  await api.post(`/admin/users/${resetUser.id}/resetpw`, { password: resetPassword.value })
  ElMessage.success('密码已重置')
  resetVisible.value = false
}

async function delUser(row) {
  await ElMessageBox.confirm(`删除用户 ${row.username} 将同时删除其全部邮箱和邮件，确定？`, '危险操作', { type: 'warning' })
  await api.del(`/admin/users/${row.id}`)
  ElMessage.success('已删除')
  loadUsers()
  loadStats()
}

async function delMailbox(row) {
  await ElMessageBox.confirm(`删除邮箱 ${row.address} 及其全部邮件？`, '危险操作', { type: 'warning' })
  await api.del(`/admin/mailboxes/${row.id}`)
  ElMessage.success('已删除')
  loadMailboxes()
  loadStats()
}

async function loadStats() {
  stats.value = await api.get('/admin/stats')
}

// 发信设置
const ms = ref({})
const loadingMs = ref(false)
const savingMs = ref(false)
const genning = ref(false)
const dkimDns = ref(null)

async function loadMs() {
  loadingMs.value = true
  try {
    ms.value = await api.get('/admin/settings/mail')
  } finally {
    loadingMs.value = false
  }
}

async function saveMs() {
  savingMs.value = true
  try {
    await api.put('/admin/settings/mail', ms.value)
    ElMessage.success('已保存，立即生效')
  } finally {
    savingMs.value = false
  }
}

async function genDKIM() {
  genning.value = true
  try {
    const r = await api.post('/admin/dkim/generate', { selector: ms.value.dkim_selector })
    ms.value.dkim_selector = r.selector
    ms.value.has_dkim_key = true
    dkimDns.value = { name: r.dns_name, txt: r.dns_txt }
    ElMessage.success('密钥已生成，请按提示添加DNS记录')
  } finally {
    genning.value = false
  }
}

onMounted(() => {
  loadStats()
  loadUsers()
  loadMailboxes()
  loadMs()
})

function fmtTime(s) {
  return new Date(s).toLocaleString('zh-CN')
}
</script>

<style scoped>
.stat-card { border-radius: 10px; }
.stat-num { font-size: 26px; font-weight: 700; color: #303133; }
.stat-label { font-size: 13px; color: #909399; }
.toolbar-row { display: flex; gap: 12px; margin-bottom: 14px; }
.field-tip { font-size: 12px; color: #909399; line-height: 1.6; margin-top: 4px; }
.dkim-dns { background: #f5f7fa; border-radius: 6px; padding: 10px 12px; font-size: 13px; width: 100%; }
.dkim-dns code { display: block; word-break: break-all; background: #eceef3; padding: 4px 8px; border-radius: 4px; margin: 4px 0 8px; }
.dkim-dns .dns-txt { min-height: 20px; }
</style>
