import { createRouter, createWebHistory } from 'vue-router'
import { store } from './store'

const routes = [
  { path: '/login', component: () => import('./views/Login.vue') },
  { path: '/register', component: () => import('./views/Register.vue') },
  {
    path: '/',
    component: () => import('./views/Layout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', component: () => import('./views/Dashboard.vue') },
      { path: 'inbox', component: () => import('./views/Inbox.vue') },
      { path: 'compose', component: () => import('./views/Compose.vue') },
      { path: 'mailboxes', component: () => import('./views/Mailboxes.vue') },
      { path: 'external', component: () => import('./views/External.vue') },
      { path: 'admin', component: () => import('./views/Admin.vue'), meta: { admin: true } }
    ]
  }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach(async (to) => {
  if (store.user === null && to.path !== '/login' && to.path !== '/register') {
    await store.init()
    if (!store.user) return '/login'
  }
  if (to.meta.admin && !store.user?.is_admin) return '/dashboard'
})

export default router
