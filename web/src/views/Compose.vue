<template>
  <el-card shadow="never">
    <template #header>
      <div class="compose-head">
        <div>
          <div class="compose-title">写邮件</div>
          <div class="compose-sub">支持多收件人与附件</div>
        </div>
      </div>
    </template>
    <el-form :model="form" :rules="rules" ref="formRef" label-position="top" style="max-width:760px">
      <el-form-item label="发件地址" prop="mailbox_id">
        <el-select v-model="form.mailbox_id" placeholder="选择发件地址" style="width:340px">
          <el-option v-for="mb in mailboxes" :key="mb.id" :label="mb.address" :value="mb.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="收件人" prop="to">
        <el-input v-model="form.to" placeholder="多个收件人用逗号分隔" />
      </el-form-item>
      <el-form-item label="主题" prop="subject">
        <el-input v-model="form.subject" placeholder="邮件主题" />
      </el-form-item>
      <el-form-item label="正文">
        <el-input v-model="form.text" type="textarea" :rows="12" placeholder="邮件正文" />
      </el-form-item>
      <el-form-item label="附件">
        <el-upload v-model:file-list="fileList" :auto-upload="false" multiple drag class="upload-box">
          <el-icon size="32" color="#98a1b2"><UploadFilled /></el-icon>
          <div class="el-upload__text">拖拽文件到这里，或<em>点击上传</em></div>
        </el-upload>
      </el-form-item>
      <el-button type="primary" size="large" :loading="sending" @click="send" class="send-btn">
        <el-icon><Promotion /></el-icon>&nbsp;发 送
      </el-button>
    </el-form>
  </el-card>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const formRef = ref()
const mailboxes = ref([])
const fileList = ref([])
const sending = ref(false)
const form = reactive({ mailbox_id: null, to: '', subject: '', text: '', html: '' })
const rules = {
  mailbox_id: [{ required: true, message: '请选择发件地址', trigger: 'change' }],
  to: [
    { required: true, message: '请输入收件人', trigger: 'blur' },
    { pattern: /^[^@\s,;]+@[^@\s,;]+\.[^@\s,;]+(,|;|\s|$)/, message: '地址格式不正确', trigger: 'blur' }
  ],
  subject: [{ required: true, message: '请输入主题', trigger: 'blur' }]
}

onMounted(async () => {
  mailboxes.value = await api.get('/mailboxes')
  if (mailboxes.value.length) form.mailbox_id = mailboxes.value[0].id
})

async function send() {
  await formRef.value.validate()
  const fd = new FormData()
  fd.append('mailbox_id', form.mailbox_id)
  fd.append('to', form.to)
  fd.append('subject', form.subject)
  fd.append('text', form.text)
  for (const f of fileList.value) {
    fd.append('files', f.raw)
  }
  sending.value = true
  try {
    const r = await api.upload('/send', fd)
    const errs = r?.errors || {}
    if (Object.keys(errs).length) {
      ElMessage.error('部分失败: ' + Object.entries(errs).map(([k, v]) => `${k}: ${v}`).join('; '))
    } else {
      ElMessage.success('发送成功')
      form.to = ''
      form.subject = ''
      form.text = ''
      fileList.value = []
    }
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.compose-title { font-size: 16px; font-weight: 700; }
.compose-sub { font-size: 12px; color: var(--cy-text-3); margin-top: 2px; }
.upload-box { width: 100%; }
.send-btn { min-width: 120px; }
</style>
