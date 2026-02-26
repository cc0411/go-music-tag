// src/api/index.js
import axios from 'axios'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

// 拦截器处理错误
request.interceptors.response.use(
  response => response.data, // 自动解包，直接返回 data 部分
  error => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export const api = {
  // --- 统计 ---
  getStats: () => request.get('/statistics'),
  
  // --- 音乐管理 ---
  
  // 搜索/列表 (匹配后端 /music/search)
  searchMusic: (params) => request.get('/music/search', { 
    params: { 
      keyword: params.keyword || '', 
      page: params.page || 1, 
      page_size: params.page_size || 20 
    } 
  }),
  
  // 兼容旧版调用
  getMusicList: (page, keyword) => api.searchMusic({ page, keyword }),

  // 获取单曲详情
  getMusicDetail: (id) => request.get(`/music/${id}`),

  // ✅ 新增：更新单曲信息 (匹配后端 PUT /music/:id)
  updateMusic: (id, data) => request.put(`/music/${id}`, data),

  // ✅ 新增：批量更新 (匹配后端 POST /music/batch)
  batchUpdateMusic: (data) => request.post('/music/batch', data),

  // 删除单曲
  deleteMusic: (id) => request.delete(`/music/${id}`),

  // ✅ 新增：删除所有音乐 (匹配后端 DELETE /music)
  deleteAllMusic: () => request.delete('/music'),

  // 获取歌词内容
  getLyrics: (id) => request.get(`/music/${id}/lyrics`),
  
  // 获取封面图片 URL (静态资源不需要经过 axios 拦截器)
  getCoverUrl: (id) => `/api/v1/music/${id}/cover`,
  
  // 获取播放地址
  getPlayUrl: (id) => `/api/v1/music/${id}/play`,

  // 获取播放列表
  getPlaylist: (params) => request.get('/music/playlist', { params }),
  
  // --- 歌词和封面获取 ---
  
  // 单条获取
  fetchLyrics: (id) => request.post(`/music/${id}/fetch-lyrics`),
  fetchCover: (id) => request.post(`/music/${id}/fetch-cover`),
  
  // 批量获取
  batchFetchLyrics: () => request.post('/music/batch-fetch-lyrics'),
  batchFetchCovers: () => request.post('/music/batch-fetch-covers'),
  
  // 批量获取全部 (匹配后端 POST /music/batch-fetch-all)
  batchFetchAll: () => request.post('/music/batch-fetch-all'),

  // 获取批量任务状态
  getBatchStatus: () => request.get('/music/batch-status'),
  
  // --- WebDAV 配置 ---
  
  getWebDAVConfig: () => request.get('/webdav/config'),
  saveWebDAVConfig: (data) => request.post('/webdav/config', data),
  testWebDAVConfig: (data) => request.post('/webdav/test', data),
  deleteWebDAVConfig: () => request.delete('/webdav/config'),
  
  // --- 扫描管理 ---
  
  // ✅ 修复：支持传入递归参数
  startScan: (options = {}) => request.post('/scan', options),
  
  getScanStatus: () => request.get('/scan/status'),
  
  getScanLogs: (params) => request.get('/scan/logs', { 
    params: { 
      page: params.page || 1, 
      page_size: params.page_size || 50,
      task_id: params.task_id || ''
    } 
  }),

  // ✅ 新增：实时日志流 (SSE)
  getScanLogsStream: () => {
    return `${request.defaults.baseURL}/scan/logs/stream`;
  }
}

export default request