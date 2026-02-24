import axios from 'axios'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

// 拦截器处理错误
request.interceptors.response.use(
  response => response.data,
  error => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export const api = {
  // 统计
  getStats: () => request.get('/statistics'),
  
  // 音乐
  getMusicList: (page, keyword) => request.get('/music/search', { params: { page, page_size: 20, keyword } }),
  deleteMusic: (id) => request.delete(`/music/${id}`),
  fetchLyrics: (id) => request.post(`/music/${id}/fetch-lyrics`),
  fetchCover: (id) => request.post(`/music/${id}/fetch-cover`),
  batchFetchLyrics: () => request.post('/music/batch-fetch-lyrics'),
  batchFetchCovers: () => request.post('/music/batch-fetch-covers'),
  getLyrics: (id) => request.get(`/music/${id}/lyrics`),
  getCoverUrl: (id) => `/api/v1/music/${id}/cover`,
  getPlayUrl: (id) => `/api/v1/music/${id}/play`,
  
  // WebDAV
  getWebDAVConfig: () => request.get('/webdav/config'),
  saveWebDAVConfig: (data) => request.post('/webdav/config', data),
  testWebDAVConfig: (data) => request.post('/webdav/test', data),
  
  // 扫描
  startScan: () => request.post('/scan'),
  getScanStatus: () => request.get('/scan/status'),
  getScanLogs: (page) => request.get('/scan/logs', { params: { page, page_size: 50 } })
}

export default request