<template>
  <div>
    <el-card shadow="never" class="toolbar">
      <div class="toolbar-row">
        <el-select v-model="mailboxId" placeholder="选择邮箱" style="width:270px">
          <el-option v-for="mb in mailboxes" :key="mb.id" :label="mb.label || mb.address" :value="mb.id" />
        </el-select>
        <el-radio-group v-model="folder">
          <el-radio-button value="inbox">收件</el-radio-button>
          <el-radio-button value="sent">已发送</el-radio-button>
        </el-radio-group>
        <el-button :icon="'Refresh'" circle @click="load" title="刷新" />
        <div class="flex-sp"></div>
        <el-button type="primary" :icon="'EditPen'" @click="$router.push('/compose')">写邮件</el-button>
      </div>
    </el-card>

    <el-card shadow="never" v-loading="loading">
      <div v-if="items.length === 0" class="mail-empty">
        <el-empty description="暂无邮件" :image-size="90" />
      </div>
      <div v-else class="mail-list">
        <div v-for="m in items" :key="m.id" class="mail-row" :class="{ unread: !m.is_read && folder === 'inbox' }" @click="openMsg(m)">
          <div class="mail-avatar" :style="{ background: avatarColor(m.from) }">{{ avatarChar(m.from) }}</div>
          <div class="mail-main">
            <div class="mail-line1">
              <span class="mail-from">{{ displayName(folder === 'inbox' ? m.from : m.to) }}</span>
              <span class="mail-time">{{ fmtTime(m.sent_at) }}</span>
            </div>
            <div class="mail-line2">
              <span class="mail-subject">{{ m.subject || '(无主题)' }}</span>
              <span v-if="m.attachment_count" class="mail-att">📎{{ m.attachment_count }}</span>
            </div>
            <div class="mail-snippet">{{ m.snippet || '　' }}</div>
          </div>
          <el-icon class="mail-del" title="删除" @click.stop="delMsg(m)"><Delete /></el-icon>
        </div>
      </div>
      <el-pagination layout="total, prev, pager, next" :total="total" :page-size="size"
        v-model:current-page="page" @current-change="load" class="pager" />
    </el-card>

    <el-drawer v-model="drawer" :title="current?.subject || '邮件详情'" size="56%">
      <div v-if="current" class="msg-detail">
        <div class="msg-head">
          <div class="mail-avatar big" :style="{ background: avatarColor(current.from) }">{{ avatarChar(current.from) }}</div>
          <div class="msg-head-meta">
            <div class="msg-from">{{ displayName(current.from) }} <span class="msg-addr">{{ rawAddr(current.from) }}</span></div>
            <div class="msg-to">收件人：{{ displayName(current.to) }} · {{ fmtTime(current.sent_at) }}</div>
          </div>
        </div>
        <div v-if="current.attachments?.length" class="atts">
          <el-tag v-for="a in current.attachments" :key="a.id" class="att-tag" @click="dl(a)" type="info" effect="plain">
            📎 {{ a.filename }} ({{ fmtSize(a.size) }})
          </el-tag>
        </div>
        <iframe v-if="current.html_body" :key="current.id" :srcdoc="current.html_body" class="mail-frame" sandbox=""></iframe>
        <pre v-else class="mail-text">{{ current.text_body }}</pre>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'
import { store } from '../store'

const mailboxes = ref([])
const mailboxId = ref(null)
const folder = ref('inbox')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const loading = ref(false)
const drawer = ref(false)
const current = ref(null)

async function loadMailboxes() {
  if (store.user?.is_admin) {
    const r = await api.get('/admin/mailboxes', { page: 1, size: 100 })
    mailboxes.value = (r.items || []).map((m) => ({ ...m, label: `[${m.owner}] ${m.address}` }))
  } else {
    mailboxes.value = await api.get('/mailboxes')
  }
  if (!mailboxId.value && mailboxes.value.length) {
    mailboxId.value = mailboxes.value[0].id
  }
}

async function load() {
  if (!mailboxId.value) return
  loading.value = true
  try {
    const r = await api.get(`/mailboxes/${mailboxId.value}/messages`, { folder: folder.value, page: page.value, size })
    items.value = r.items || []
    total.value = r.total
  } finally {
    loading.value = false
  }
}

async function openMsg(row) {
  const r = await api.get(`/messages/${row.id}`)
  current.value = { ...r.message, attachments: r.attachments || [] }
  row.is_read = true
  drawer.value = true
}

async function delMsg(row) {
  await ElMessageBox.confirm('确定删除这封邮件？', '提示', { type: 'warning' })
  await api.del(`/messages/${row.id}`)
  ElMessage.success('已删除')
  load()
}

function dl(a) {
  window.open(`/api/attachments/${a.id}`, '_blank')
}

function displayName(s) {
  if (!s) return '(未知)'
  let name = s.includes('<') ? s.slice(0, s.indexOf('<')) : s.split('@')[0]
  name = name.trim().replace(/^"|"$/g, '')
  return name || rawAddr(s)
}
function rawAddr(s) {
  if (!s) return ''
  const m = s.match(/<([^>]+)>/)
  return m ? m[1] : s
}
function avatarChar(s) {
  return displayName(s).charAt(0).toUpperCase()
}
function avatarColor(s) {
  const hues = ['#4c6fff', '#00a870', '#f59e0b', '#e4574c', '#8f5cff', '#0e9bb5', '#d9527f']
  let h = 0
  for (const c of s || '') h = (h * 31 + c.charCodeAt(0)) % 997
  return hues[h % hues.length]
}
function fmtTime(s) {
  const d = new Date(s)
  const now = new Date()
  if (d.toDateString() === now.toDateString()) return d.toTimeString().slice(0, 5)
  return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
function fmtSize(n) {
  if (n > 1048576) return (n / 1048576).toFixed(1) + 'MB'
  if (n > 1024) return (n / 1024).toFixed(1) + 'KB'
  return n + 'B'
}

watch([mailboxId, folder], () => { page.value = 1; load() })
onMounted(loadMailboxes)
</script>

<style scoped>
.toolbar { margin-bottom: 14px; }
.toolbar-row { display: flex; gap: 12px; align-items: center; }
.flex-sp { flex: 1; }
.mail-list { --el-card-padding: 0; }
.mail-row {
  display: flex; align-items: center; gap: 14px;
  padding: 13px 16px; border-bottom: 1px solid var(--cy-border);
  cursor: pointer; transition: background .12s;
  position: relative;
}
.mail-row:hover { background: #f7f9fc; }
.mail-row.unread { background: #f4f7ff; }
.mail-row.unread .mail-from, .mail-row.unread .mail-subject { font-weight: 700; }
.mail-row.unread::before {
  content: ''; position: absolute; left: 4px; top: 50%; transform: translateY(-50%);
  width: 6px; height: 6px; border-radius: 50%; background: var(--cy-primary);
}
.mail-avatar {
  width: 38px; height: 38px; border-radius: 50%; flex-shrink: 0;
  color: #fff; font-weight: 700; font-size: 15px;
  display: flex; align-items: center; justify-content: center;
}
.mail-avatar.big { width: 46px; height: 46px; font-size: 18px; }
.mail-main { flex: 1; min-width: 0; }
.mail-line1 { display: flex; justify-content: space-between; align-items: baseline; }
.mail-from { font-size: 14px; }
.mail-time { font-size: 12px; color: var(--cy-text-3); flex-shrink: 0; margin-left: 12px; }
.mail-line2 { display: flex; align-items: center; gap: 8px; margin-top: 2px; }
.mail-subject { font-size: 13.5px; color: var(--cy-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.mail-att { font-size: 12px; color: var(--cy-text-2); flex-shrink: 0; }
.mail-snippet {
  font-size: 12.5px; color: var(--cy-text-3); margin-top: 2px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.mail-del {
  flex-shrink: 0; padding: 8px; border-radius: 6px;
  color: var(--cy-text-3); opacity: 0; transition: all .12s;
}
.mail-row:hover .mail-del { opacity: 1; }
.mail-del:hover { color: #e4574c; background: #fdecec; }
.mail-empty { padding: 30px 0; }
.pager { margin-top: 14px; justify-content: flex-end; }
.msg-detail { padding: 2px; }
.msg-head { display: flex; gap: 14px; align-items: center; padding-bottom: 14px; border-bottom: 1px solid var(--cy-border); margin-bottom: 14px; }
.msg-from { font-size: 15px; font-weight: 700; }
.msg-addr { color: var(--cy-text-3); font-weight: 400; font-size: 13px; margin-left: 6px; }
.msg-to { color: var(--cy-text-2); font-size: 13px; margin-top: 3px; }
.atts { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 12px; }
.att-tag { cursor: pointer; }
.mail-frame { width: 100%; min-height: 55vh; border: 1px solid var(--cy-border); border-radius: 8px; background: #fff; }
.mail-text { white-space: pre-wrap; word-break: break-word; font-family: inherit; line-height: 1.8; font-size: 14px; }
</style>
