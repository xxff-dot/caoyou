<template>
  <div class="auth-page">
    <div class="auth-brand">
      <div class="brand-logo">📮</div>
      <h1>草鼬 CaoYou</h1>
      <p class="brand-sub">企业级自建邮件服务平台</p>
      <ul class="brand-feats">
        <li>自有域名，专属邮箱地址</li>
        <li>自建SMTP收信，数据完全自持</li>
        <li>外部邮箱聚合，收发一站式</li>
      </ul>
      <div class="brand-foot">© 2026 CaoYou Mail</div>
    </div>
    <div class="auth-panel">
      <el-card class="auth-card">
        <h2 class="auth-title">欢迎回来</h2>
        <p class="auth-desc">登录你的账号，继续使用草鼬</p>
        <el-form :model="form" :rules="rules" ref="formRef" label-position="top" size="large">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" placeholder="请输入用户名">
              <template #prefix><el-icon><User /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" @keyup.enter="submit">
              <template #prefix><el-icon><Lock /></el-icon></template>
            </el-input>
          </el-form-item>
          <el-button type="primary" class="auth-btn" :loading="loading" @click="submit">登 录</el-button>
          <div class="auth-footer">还没有账号？<router-link to="/register">立即注册</router-link></div>
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
const form = reactive({ username: '', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function submit() {
  await formRef.value.validate()
  loading.value = true
  try {
    await api.post('/login', form)
    await store.init()
    router.push('/dashboard')
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
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.brand-logo {
  width: 56px; height: 56px; border-radius: 14px;
  background: rgba(255,255,255,.1);
  display: flex; align-items: center; justify-content: center;
  font-size: 28px; margin-bottom: 24px;
}
.auth-brand h1 { font-size: 30px; font-weight: 700; letter-spacing: .02em; }
.brand-sub { margin-top: 8px; color: #a9b6d8; font-size: 15px; }
.brand-feats { list-style: none; margin: 5vh 0 0; padding: 0; }
.brand-feats li {
  color: #c6d0e8; font-size: 14px; line-height: 2.4;
  padding-left: 26px; position: relative;
}
.brand-feats li::before {
  content: '✓'; position: absolute; left: 0; top: 0;
  width: 18px; height: 18px; border-radius: 50%;
  background: var(--cy-primary); color: #fff; font-size: 11px;
  display: flex; align-items: center; justify-content: center; margin-top: 8px;
}
.brand-foot { margin-top: auto; color: #5d6b8f; font-size: 12px; }
.auth-panel {
  flex: 1;
  display: flex; align-items: center; justify-content: center;
  background: #fff;
}
.auth-card { width: 400px; border: none !important; box-shadow: none !important; }
.auth-title { font-size: 24px; font-weight: 700; margin-bottom: 6px; }
.auth-desc { color: var(--cy-text-2); margin-bottom: 28px; }
.auth-btn { width: 100%; margin-top: 8px; height: 44px; font-size: 15px; }
.auth-footer { text-align: center; margin-top: 18px; font-size: 14px; color: var(--cy-text-2); }
.auth-footer a { color: var(--cy-primary); text-decoration: none; font-weight: 500; }
@media (max-width: 900px) { .auth-brand { display: none; } }
</style>
