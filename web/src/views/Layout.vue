<template>
  <el-container class="layout">
    <el-aside width="228px" class="aside">
      <div class="brand">
        <span class="brand-mark">📮</span>
        <span class="brand-name">草鼬 <em>CaoYou</em></span>
      </div>
      <div class="menu-group-title">邮件</div>
      <el-menu :default-active="$route.path" router background-color="transparent" text-color="#9aa5bd" active-text-color="#ffffff">
        <el-menu-item index="/dashboard"><el-icon><Odometer /></el-icon>工作台</el-menu-item>
        <el-menu-item index="/inbox"><el-icon><MessageBox /></el-icon>收件箱</el-menu-item>
        <el-menu-item index="/compose"><el-icon><EditPen /></el-icon>写邮件</el-menu-item>
      </el-menu>
      <div class="menu-group-title">管理</div>
      <el-menu :default-active="$route.path" router background-color="transparent" text-color="#9aa5bd" active-text-color="#ffffff">
        <el-menu-item index="/mailboxes"><el-icon><Postcard /></el-icon>邮箱地址</el-menu-item>
        <el-menu-item index="/external"><el-icon><Link /></el-icon>外部邮箱</el-menu-item>
        <el-menu-item v-if="store.user?.is_admin" index="/admin"><el-icon><Setting /></el-icon>系统管理</el-menu-item>
      </el-menu>
      <div class="aside-user">
        <el-avatar :size="34" class="user-avatar">{{ avatarChar }}</el-avatar>
        <div class="user-meta">
          <div class="user-name">{{ store.user?.username }}</div>
          <div class="user-role">{{ store.user?.is_admin ? '管理员' : '成员' }}</div>
        </div>
        <el-tooltip content="退出登录" placement="top">
          <el-icon class="logout-btn" @click="logout"><SwitchButton /></el-icon>
        </el-tooltip>
      </div>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-title">{{ title }}</div>
        <el-button v-if="$route.path === '/inbox'" type="primary" :icon="'EditPen'" @click="$router.push('/compose')">写邮件</el-button>
      </el-header>
      <el-main class="main"><router-view /></el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api'
import { store } from '../store'

const route = useRoute()
const router = useRouter()

const titles = {
  '/dashboard': '工作台',
  '/inbox': '收件箱',
  '/compose': '写邮件',
  '/mailboxes': '邮箱地址',
  '/external': '外部邮箱',
  '/admin': '系统管理'
}
const title = computed(() => titles[route.path] || '草鼬')
const avatarChar = computed(() => (store.user?.username || '?').charAt(0).toUpperCase())

async function logout() {
  await api.post('/logout')
  store.user = null
  router.push('/login')
}
</script>

<style scoped>
.layout { height: 100%; }
.aside {
  background: var(--cy-sidebar);
  display: flex; flex-direction: column;
  padding: 0 0 12px;
}
.brand {
  display: flex; align-items: center; gap: 10px;
  padding: 20px 22px 18px;
}
.brand-mark {
  width: 34px; height: 34px; border-radius: 9px;
  background: linear-gradient(135deg, var(--cy-primary), #7a5cff);
  display: flex; align-items: center; justify-content: center; font-size: 17px;
}
.brand-name { color: #fff; font-size: 17px; font-weight: 700; letter-spacing: .02em; }
.brand-name em { font-style: normal; color: #6d7da3; font-size: 12px; font-weight: 500; margin-left: 4px; }
.menu-group-title {
  color: #57637f; font-size: 11px; font-weight: 600; letter-spacing: .08em;
  padding: 14px 24px 6px; text-transform: uppercase;
}
.aside .el-menu { border-right: none; }
.aside-user {
  margin-top: auto;
  display: flex; align-items: center; gap: 10px;
  margin-left: 12px; margin-right: 12px;
  padding: 10px 12px;
  background: #1c2538; border-radius: 10px;
}
.user-avatar { background: var(--cy-primary); font-weight: 600; flex-shrink: 0; }
.user-meta { flex: 1; min-width: 0; }
.user-name { color: #e8ecf5; font-size: 13px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.user-role { color: #6d7da3; font-size: 11px; }
.logout-btn { color: #6d7da3; cursor: pointer; font-size: 16px; }
.logout-btn:hover { color: #ff7d7d; }
.header {
  background: #fff;
  display: flex; align-items: center; justify-content: space-between;
  border-bottom: 1px solid var(--cy-border);
  height: 56px;
}
.header-title { font-size: 16px; font-weight: 700; }
.main { padding: 20px 24px; overflow: auto; }
</style>
