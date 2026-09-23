import { reactive } from 'vue'
import api from './api'

export const store = reactive({
  user: null,
  domain: 'localhost',
  async init() {
    try {
      const me = await api.get('/me')
      this.user = me
      this.domain = (await api.get('/config')).domain || this.domain
    } catch {
      this.user = null
    }
  },
  async refresh() {
    this.user = (await api.get('/me')) || null
  }
})
