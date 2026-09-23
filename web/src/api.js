import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 30000 })

http.interceptors.response.use(
  (resp) => resp.data?.data ?? resp.data,
  (errResp) => {
    const msg = errResp.response?.data?.error || '请求失败'
    if (errResp.response?.status === 401 && location.pathname !== '/login') {
      location.href = '/login'
    }
    return Promise.reject(new Error(msg))
  }
)

export default {
  async get(url, params) {
    return http.get(url, { params })
  },
  async post(url, data) {
    return http.post(url, data)
  },
  async put(url, data) {
    return http.put(url, data)
  },
  async upload(url, formData) {
    return http.post(url, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  async del(url) {
    return http.delete(url)
  }
}
