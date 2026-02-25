<template>
  <div class="scan-page">
    <!-- 1. 扫描控制卡片 -->
    <el-card class="box-card mb-4">
      <template #header>
        <div class="card-header">
          <el-icon><Search /></el-icon>
          <span>扫描控制</span>
        </div>
      </template>

      <el-form :inline="true" class="scan-form">
        <el-form-item>
          <el-checkbox v-model="scanOptions.recursive" border>
            递归扫描子目录
          </el-checkbox>
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="primary" 
            @click="startScan" 
            :loading="scanning" 
            :disabled="scanning"
            icon="VideoPlay"
          >
            {{ scanning ? '扫描进行中...' : '开始扫描' }}
          </el-button>
          
          <el-button 
            v-if="scanning" 
            @click="checkStatus" 
            icon="Refresh" 
            circle 
            title="刷新状态"
          />
        </el-form-item>
      </el-form>

      <!-- 进度条 (仅在扫描中显示) -->
      <div v-if="scanning" class="progress-section">
        <el-progress 
          :percentage="progressPercentage" 
          :status="progressStatus"
          :format="progressFormat"
        />
        <div class="status-text">{{ statusMessage }}</div>
      </div>
    </el-card>

    <!-- 2. 扫描日志卡片 -->
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <el-icon><Document /></el-icon>
          <span>扫描日志</span>
          <el-button link type="primary" size="small" @click="loadLogs" :loading="loadingLogs">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
      </template>

      <div class="log-container" ref="logContainer">
        <div v-if="logs.length === 0" class="empty-log">
          <el-empty description="暂无扫描日志" :image-size="60" />
        </div>
        
        <div v-else>
          <div v-for="(log, index) in logs" :key="index" class="log-line" :class="getLogClass(log)">
            <span class="log-time">{{ formatTime(log.created_at) }}</span>
            <span class="log-level" :class="log.level.toLowerCase()">{{ log.level.toUpperCase() }}</span>
            <span class="log-msg">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, VideoPlay, Refresh, Document } from '@element-plus/icons-vue'
import { api } from '@/api'

// 状态
const scanning = ref(false)
const loadingLogs = ref(false)
const logs = ref([])
const logContainer = ref(null)

// 扫描选项
const scanOptions = ref({
  recursive: true
})

// 进度状态
const progressPercentage = ref(0)
const progressStatus = ref('') // '' | 'exception' | 'success'
const statusMessage = ref('准备开始...')

// 轮询定时器
let statusTimer = null

// 开始扫描
const startScan = async () => {
  try {
    const res = await api.startScan(scanOptions.value)
    if (res.code === 0) {
      ElMessage.success('扫描任务已启动')
      scanning.value = true
      progressPercentage.value = 0
      progressStatus.value = ''
      statusMessage.value = '正在初始化...'
      
      // 启动轮询
      checkStatus()
      if (statusTimer) clearInterval(statusTimer)
      statusTimer = setInterval(checkStatus, 2000) // 每 2 秒查询一次状态
    }
  } catch (error) {
    ElMessage.error('启动扫描失败：' + (error.response?.data?.message || error.message))
  }
}

// 检查扫描状态
const checkStatus = async () => {
  try {
    const res = await api.getScanStatus()
    if (res.code === 0 && res.data) {
      const data = res.data
      
      // 更新进度条逻辑 (根据后端返回的具体字段调整)
      // 假设后端返回 last_log 包含 "Processing [20/29]" 这样的信息
      if (data.last_log) {
        statusMessage.value = data.last_log
        const match = data.last_log.match(/\[(\d+)\/(\d+)\]/)
        if (match) {
          const current = parseInt(match[1])
          const total = parseInt(match[2])
          progressPercentage.value = Math.floor((current / total) * 100)
        } else {
           // 如果没有进度信息，使用动画进度
           progressPercentage.value = 50 // 或者保持上一个值
        }
      }

      // 如果后端明确返回了 running: false，则停止
      if (data.running === false) {
        stopScanning()
      }
    }
  } catch (error) {
    console.error('获取状态失败', error)
  }
}

// 停止扫描状态
const stopScanning = () => {
  scanning.value = false
  progressPercentage.value = 100
  progressStatus.value = 'success'
  statusMessage.value = '扫描完成！'
  
  if (statusTimer) {
    clearInterval(statusTimer)
    statusTimer = null
  }
  
  ElMessage.success('扫描任务已完成')
  loadLogs() // 加载最新日志
}

// 加载日志
const loadLogs = async () => {
  loadingLogs.value = true
  try {
    // 假设 API 支持分页，这里先取第一页 50 条
    const res = await api.getScanLogs({ page: 1, page_size: 50 })
    if (res.code === 0) {
      // 如果是列表结构 (res.data.list) 还是直接数组 (res.data)，做兼容处理
      logs.value = res.data.list || res.data || []
      
      // 滚动到底部
      await nextTick()
      scrollToBottom()
    }
  } catch (error) {
    console.error('加载日志失败', error)
  } finally {
    loadingLogs.value = false
  }
}

// 滚动到底部
const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

// 格式化时间
const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', { hour12: false })
}

// 获取日志样式类
const getLogClass = (log) => {
  const msg = log.message.toLowerCase()
  if (msg.includes('error') || log.level === 'error') return 'log-error'
  if (msg.includes('success') || msg.includes('completed')) return 'log-success'
  return 'log-info'
}

const getLogColor = (log) => {
  const msg = log.message.toLowerCase()
  if (msg.includes('error')) return '#F56C6C'
  if (msg.includes('success') || msg.includes('completed')) return '#67C23A'
  return '#606266'
}

onMounted(() => {
  loadLogs()
  // 可选：页面加载时检查是否正在扫描
  checkStatus()
})

onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
})
</script>

<style scoped lang="scss">
.scan-page {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: 100%;
}

.mb-4 {
  margin-bottom: 20px;
}

.box-card {
  border-radius: 8px;
  
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 16px;
    color: #1f2937;
    
    .el-icon {
      color: #409EFF;
    }
  }
}

.scan-form {
  .el-form-item {
    margin-bottom: 0;
  }
}

.progress-section {
  margin-top: 20px;
  
  .status-text {
    margin-top: 8px;
    font-size: 13px;
    color: #606266;
    font-family: monospace;
  }
}

.log-container {
  height: 400px;
  overflow-y: auto;
  background-color: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 10px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  
  &::-webkit-scrollbar {
    width: 6px;
  }
  &::-webkit-scrollbar-thumb {
    background: #c0c4cc;
    border-radius: 3px;
  }
}

.empty-log {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: #909399;
}

.log-line {
  margin-bottom: 6px;
  word-break: break-all;
  white-space: pre-wrap;
  
  .log-time {
    color: #909399;
    margin-right: 10px;
    font-size: 12px;
  }
  
  .log-level {
    font-weight: bold;
    margin-right: 8px;
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 11px;
    
    &.info {
      background-color: #ecf5ff;
      color: #409EFF;
    }
    &.error {
      background-color: #fef0f0;
      color: #F56C6C;
    }
    &.warning {
      background-color: #fdf6ec;
      color: #E6A23C;
    }
  }
  
  .log-msg {
    color: #303133;
  }
}

.log-error .log-msg { color: #F56C6C; }
.log-success .log-msg { color: #67C23A; }
.log-info .log-msg { color: #606266; }
</style>