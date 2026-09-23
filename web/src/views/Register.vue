<template>
  <div class="auth-page">
    <div class="auth-brand">
      <div class="brand-logo">📮</div>
      <h1>草鼬 CaoYou</h1>
      <p class="brand-sub">注册即开通 用户名@域名 专属邮箱</p>
      <ul class="brand-feats">
        <li>注册自动获得专属邮箱地址</li>
        <li>可绑定QQ/Gmail等外部邮箱</li>
        <li>收件、发件、附件完整支持</li>
      </ul>
      <div class="brand-foot">© 2026 CaoYou Mail</div>
    </div>
    <div class="auth-panel">
      <el-card class="auth-card">
        <h2 class="auth-title">创建账号</h2>
        <p class="auth-desc">填写以下信息完成注册</p>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top" size="large">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" placeholder="3-32位字母数字下划线">
              <template #prefix><el-icon><User /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="至少6位">
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="备用邮箱（选填）">
            <el-input v-model="form.email" placeholder="用于找回账号" />
          </el-form-item>
          <el-button type="primary" class="auth-btn" :loading="loading" @click="submit">注 册</el-button>
          <div class="auth-footer">已有账号？<router-link to="/login">直接登录</router-link></div>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../api'
import { store } from '../store'

const router = useRouter()
const formRef = ref()
const loading = ref(false)
const form = reactive({ username: '', password: '', email: '' })
const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]{3,32}$/, message: '3-32位字母数字下划线', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '至少6位', trigger: 'blur' }
  ]
}

async function submit() {
  await formRef.value.validate()
  loading.value = true
  try {
    await api.post('/register', form)
    await store.init()
    ElMessage.success('注册成功')
    router.push('/mailboxes')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page { height: 100%; display: flex; }
.auth-brand {
  flex: 1.1;
  background: linear-gradient(160deg, #141b2e 0%, #1d2a4a 60%, #24356b 100%);
  color: #fff;
  padding: 8vh 7vw;
  display: flex; flex-direction: column; justify-content: center;
}
.brand-logo {
  width: 56px; height: 56px; border-radius: 14px;
  background: rgba(255,255,255,.1);
  display: flex; align-items: center; justify-content: center;
  font-size: 28px; margin-bottom: 24px;
}
.auth-brand h1 { font-size: 30px; font-weight: 700; }
.brand-sub { margin-top: 8px; color: #a9b6d8; font-size: 15px; }
.brand-feats { list-style: none; margin: 5vh 0 0; padding: 0; }
.brand-feats li { color: #c6d0e8; line-height: 2.4; padding-left: 26px; position: relative; }
.brand-feats li::before {
  content: '✓'; position: absolute; left: 0; top: 0;
  width: 18px; height: 18px; border-radius: 50%;
  background: var(--cy-primary); color: #fff; font-size: 11px;
  display: flex; align-items: center; justify-content: center; margin-top: 8px;
}
.brand-foot { margin-top: auto; color: #5d6b8f; font-size: 12px; }
.auth-panel { flex: 1; display: flex; align-items: center; justify-content: center; background: #fff; }
.auth-card { width: 400px; border: none !important; box-shadow: none !important; }
.auth-title { font-size: 24px; font-weight: 700; margin-bottom: 6px; }
.auth-desc { color: var(--cy-text-2); margin-bottom: 28px; }
.auth-btn { width: 100%; margin-top: 8px; height: 44px; font-size: 15px; }
.auth-footer { text-align: center; margin-top: 18px; font-size: 14px; color: var(--cy-text-2); }
.auth-footer a { color: var(--cy-primary); text-decoration: none; font-weight: 500; }
@media (max-width: 900px) { .auth-brand { display: none; } }
</style>
