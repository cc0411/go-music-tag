<template>
    <el-card>
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索..." style="width: 300px" @keyup.enter="loadData" />
        <el-button type="primary" @click="loadData">搜索</el-button>
        <el-button type="success" @click="handleBatch('lyrics')">批量歌词</el-button>
        <el-button type="warning" @click="handleBatch('covers')">批量封面</el-button>
      </div>
  
      <el-table :data="tableData" stripe v-loading="loading" style="margin-top: 20px">
        <el-table-column prop="file_name" label="文件名" />
        <el-table-column prop="title" label="标题" />
        <el-table-column prop="artist" label="艺术家" />
        <el-table-column prop="album" label="专辑" />
        <el-table-column label="状态" width="150">
          <template #default="{ row }">
            <el-tag v-if="row.has_lyrics" type="success" size="small">歌词</el-tag>
            <el-tag v-if="row.has_cover" type="primary" size="small" style="margin-left:5px">封面</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="play(row)">播放</el-button>
            <el-button size="small" type="danger" @click="del(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
  
      <div class="pagination">
        <el-pagination v-model:current-page="page" :total="total" :page-size="20" @current-change="loadData" layout="prev, pager, next" />
      </div>
    </el-card>
  </template>
  
  <script setup>
  import { ref, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { api } from '@/api'
  import { usePlayerStore } from '@/stores/player'
  
  const keyword = ref('')
  const tableData = ref([])
  const loading = ref(false)
  const page = ref(1)
  const total = ref(0)
  const playerStore = usePlayerStore()
  
  const loadData = async () => {
    loading.value = true
    try {
      const res = await api.getMusicList(page.value, keyword.value)
      if (res.code === 0) {
        tableData.value = res.data.list || res.data || []
        total.value = res.total || tableData.value.length
      }
    } finally {
      loading.value = false
    }
  }
  
  const play = (row) => playerStore.playTrack(row, tableData.value)
  
  const del = async (id) => {
    await ElMessageBox.confirm('确定删除？', '提示')
    await api.deleteMusic(id)
    ElMessage.success('删除成功')
    loadData()
  }
  
  const handleBatch = async (type) => {
    try {
      await (type === 'lyrics' ? api.batchFetchLyrics() : api.batchFetchCovers())
      ElMessage.success('任务已启动')
    } catch (e) {
      ElMessage.error('启动失败')
    }
  }
  
  onMounted(loadData)
  </script>
  
  <style scoped>
  .toolbar { display: flex; gap: 10px; }
  .pagination { margin-top: 20px; justify-content: flex-end; display: flex; }
  </style>