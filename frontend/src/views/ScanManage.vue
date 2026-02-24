<template>
    <el-card>
      <el-button type="primary" @click="startScan" :loading="scanning">开始扫描</el-button>
      
      <div style="margin-top: 20px; max-height: 400px; overflow-y: auto; background: #f5f7fa; padding: 10px;">
        <div v-for="(log, i) in logs" :key="i" class="log-line">
          <span :class="log.level">{{ log.created_at }}</span> - {{ log.message }}
        </div>
      </div>
    </el-card>
  </template>
  
  <script setup>
  import { ref, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { api } from '@/api'
  
  const scanning = ref(false)
  const logs = ref([])
  
  const startScan = async () => {
    scanning.value = true
    await api.startScan()
    ElMessage.success('扫描已开始')
    loadLogs()
  }
  
  const loadLogs = async () => {
    const res = await api.getScanLogs(1)
    if (res.code === 0) logs.value = res.data || []
    scanning.value = false
  }
  
  onMounted(loadLogs)
  </script>
  
  <style scoped>
  .log-line { font-family: monospace; font-size: 13px; margin-bottom: 5px; }
  .info { color: #409EFF; }
  .error { color: #F56C6C; }
  </style>