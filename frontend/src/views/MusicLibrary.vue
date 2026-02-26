<template>
  <div class="music-library-page">
    <el-card class="box-card">
      <!-- 1. 顶部工具栏 (保持不变) -->
      <div class="toolbar">
        <div class="search-group">
          <el-input 
            v-model="keyword" 
            placeholder="搜索音乐、艺术家、专辑..." 
            style="width: 300px" 
            clearable
            @keyup.enter="loadData"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" @click="loadData" :icon="Search">搜索</el-button>
        </div>

        <div class="batch-group">
          <el-button type="success" :icon="Document" @click="handleBatch('lyrics')">批量歌词</el-button>
          <el-button type="warning" :icon="Picture" @click="handleBatch('covers')">批量封面</el-button>
          <el-button type="primary" :icon="FolderChecked" @click="handleBatch('all')">批量获取全部</el-button>
          
          <el-divider direction="vertical" />

          <el-button type="danger" :icon="Delete" @click="batchDelete" :disabled="selectedIds.length === 0">
            批量删除 ({{ selectedIds.length }})
          </el-button>
          
          <el-button type="info" :icon="Plus" @click="batchAddToPlaylist" :disabled="selectedIds.length === 0">
            添加到列表
          </el-button>
        </div>
      </div>

      <!-- 2. 数据表格 -->
      <el-table 
        ref="tableRef"
        :data="tableData" 
        stripe 
        v-loading="loading" 
        style="margin-top: 20px"
        header-cell-class-name="table-header-gray"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="title" label="标题" min-width="200">
          <template #default="{ row }">
            <div class="song-title">
              <span class="name">{{ row.title || row.file_name }}</span>
              <el-tag v-if="row.has_lyrics" size="small" type="success" effect="plain" style="margin-left: 5px; font-size: 10px">词</el-tag>
              <el-tag v-if="row.has_cover" size="small" type="primary" effect="plain" style="margin-left: 2px; font-size: 10px">图</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="artist" label="艺术家" width="150" show-overflow-tooltip />
        <el-table-column prop="album" label="专辑" width="150" show-overflow-tooltip />

        <!-- ✅ 优化：时长显示逻辑 -->
        <el-table-column label="时长" width="100" align="center">
          <template #default="{ row }">
            <span :class="['duration-text', { 'is-zero': !row.duration || row.duration <= 0 }]">
              {{ formatDuration(row.duration) }}
            </span>
            <!-- 如果时长为 0，显示一个小提示图标 -->
            <el-tooltip v-if="!row.duration || row.duration <= 0" content="未检测到时长信息，可尝试重新扫描或编辑" placement="top">
              <el-icon class="warning-icon"><Warning /></el-icon>
            </el-tooltip>
          </template>
        </el-table-column>

        <!-- ✅ 优化：操作栏使用下拉菜单，超过 3 个自动隐藏 -->
        <el-table-column label="操作" width="240" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <!-- 1. 播放 (仅图标) -->
              <el-tooltip content="播放" placement="top">
                <el-button 
                  link 
                  type="primary" 
                  size="small" 
                  @click="play(row)" 
                  :icon="VideoPlay" 
                  circle 
                />
              </el-tooltip>

              <!-- 2. 添加到列表 (仅图标) -->
              <el-tooltip content="添加到播放列表" placement="top">
                <el-button 
                  link 
                  type="info" 
                  size="small" 
                  @click="addToPlaylist(row)" 
                  :icon="Plus" 
                  circle 
                />
              </el-tooltip>

              <!-- 3. 查看 (图标 + 文字，因为这是新增的核心功能) -->
              <el-tooltip content="查看详情" placement="top">
                <el-button 
                  link 
                  type="success" 
                  size="small" 
                  @click="viewDetail(row)" 
                >
                  <el-icon><View /></el-icon>
                  <span style="margin-left: 4px">查看</span>
                </el-button>
              </el-tooltip>

              <!-- 4. 更多 (下拉菜单) -->
              <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, row)">
                <el-button link type="info" size="small">
                  更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="edit" :icon="Edit">编辑信息</el-dropdown-item>
                    <el-dropdown-item command="fetch_lyrics" :icon="Document" :disabled="row.has_lyrics">
                      {{ row.has_lyrics ? '已有歌词' : '获取歌词' }}
                    </el-dropdown-item>
                    <el-dropdown-item command="fetch_cover" :icon="Picture" :disabled="row.has_cover">
                      {{ row.has_cover ? '已有封面' : '获取封面' }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" :icon="Delete" divided style="color: #F56C6C">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 3. 分页 -->
      <div class="pagination">
        <el-pagination 
          v-model:current-page="page" 
          :total="total" 
          :page-size="pageSize" 
          @current-change="loadData" 
          layout="prev, pager, next, jumper, total" 
        />
      </div>
    </el-card>

    <!-- 4. 编辑对话框 (保持不变) -->
    <el-dialog v-model="editDialogVisible" title="编辑音乐信息" width="500px">
      <el-form :model="editForm" label-width="80px" label-position="left">
        <el-form-item label="标题"><el-input v-model="editForm.title" /></el-form-item>
        <el-form-item label="艺术家"><el-input v-model="editForm.artist" /></el-form-item>
        <el-form-item label="专辑"><el-input v-model="editForm.album" /></el-form-item>
        <el-form-item label="年份"><el-input-number v-model="editForm.year" :min="1900" :max="2100" style="width: 100%" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveEdit" :loading="saving">保存</el-button>
      </template>
    </el-dialog>

    <!-- 5. ✅ 新增：详情查看对话框 (支持封面大图) -->
    <el-dialog v-model="detailVisible" title="音乐详情" width="600px" class="detail-dialog">
      <div v-if="currentRow" class="detail-content">
        <div class="detail-header">
          <!-- 封面展示区域 -->
          <div class="cover-area">
            <img 
              v-if="currentRow.has_cover" 
              :src="getCoverUrl(currentRow.id)" 
              alt="Cover" 
              class="cover-image"
              @error="handleImageError"
            />
            <div v-else class="cover-placeholder">
              <el-icon :size="50"><Picture /></el-icon>
              <span>无封面</span>
            </div>
          </div>
          
          <div class="info-area">
            <h3 class="song-name">{{ currentRow.title || currentRow.file_name }}</h3>
            <p><strong>艺术家:</strong> {{ currentRow.artist || '-' }}</p>
            <p><strong>专辑:</strong> {{ currentRow.album || '-' }}</p>
            <p><strong>时长:</strong> <span :class="{ 'text-warning': !currentRow.duration }">{{ formatDuration(currentRow.duration) }}</span></p>
            <p><strong>格式:</strong> {{ currentRow.format || 'MP3' }}</p>
            <p><strong>比特率:</strong> {{ currentRow.bit_rate ? currentRow.bit_rate + ' kbps' : '-' }}</p>
          </div>
        </div>

        <el-divider />

        <div class="detail-grid">
          <div class="detail-item"><strong>文件名:</strong> {{ currentRow.file_name }}</div>
          <div class="detail-item"><strong>文件大小:</strong> {{ (currentRow.file_size / 1024 / 1024).toFixed(2) }} MB</div>
          <div class="detail-item"><strong>路径:</strong> <span class="text-ellipsis">{{ currentRow.file_path }}</span></div>
          <div class="detail-item"><strong>扫描时间:</strong> {{ currentRow.scanned_at ? formatDate(currentRow.scanned_at) : '-' }}</div>
          <div class="detail-item"><strong>状态:</strong> 
            <el-tag :type="currentRow.scan_status === 'success' ? 'success' : 'warning'" size="small">
              {{ currentRow.scan_status }}
            </el-tag>
          </div>
        </div>
        
        <div v-if="currentRow.scan_error" class="error-msg">
          <el-alert title="扫描错误" type="error" :closable="false" show-icon>
            {{ currentRow.scan_error }}
          </el-alert>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button type="primary" @click="openEditFromDetail">编辑信息</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Search, Document, Picture, FolderChecked, Delete, Plus, VideoPlay, Edit, View, 
  Warning, ArrowDown 
} from '@element-plus/icons-vue'
import { api } from '@/api'
import { usePlayerStore } from '@/stores/player'

const tableRef = ref(null)
const keyword = ref('')
const tableData = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const playerStore = usePlayerStore()

const selectedIds = ref([])
const selectedRows = ref([])

// 弹窗控制
const editDialogVisible = ref(false)
const detailVisible = ref(false)
const saving = ref(false)
const currentRow = ref(null) // 当前选中的行数据

const editForm = ref({ id: null, title: '', artist: '', album: '', year: 0 })

// ✅ 优化：时长格式化
const formatDuration = (seconds) => {
  if (!seconds && seconds !== 0) return '--:--'
  const num = Number(seconds)
  if (isNaN(num) || num <= 0) return '--:--' // 显示为 --:-- 表示未知
  const m = Math.floor(num / 60)
  const s = Math.floor(num % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 获取封面 URL
const getCoverUrl = (id) => {
  return api.getCoverUrl(id) // 确保 api.js 中返回的是完整 URL 字符串
}

const handleImageError = (e) => {
  e.target.style.display = 'none'
  ElMessage.warning('封面图片加载失败')
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.searchMusic({ keyword: keyword.value, page: page.value, page_size: pageSize.value })
    if (res.code === 0) {
      if (Array.isArray(res.data)) {
        tableData.value = res.data
        total.value = res.total || res.data.length
      } else if (res.data && Array.isArray(res.data.list)) {
        tableData.value = res.data.list
        total.value = res.data.total || res.total || 0
      } else {
        tableData.value = []
        total.value = 0
      }
      if (tableRef.value) tableRef.value.clearSelection()
    }
  } catch (error) {
    ElMessage.error('加载失败：' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection) => {
  selectedIds.value = selection.map(item => item.id)
  selectedRows.value = selection
}

const play = (row) => playerStore.playTrack(row, tableData.value)

const addToPlaylist = (row) => {
  playerStore.addTrackToQueue(row)
  ElMessage.success(`已添加 "${row.title || row.file_name}"`)
}

const batchAddToPlaylist = () => {
  if (selectedRows.value.length === 0) return
  selectedRows.value.forEach(row => playerStore.addTrackToQueue(row))
  ElMessage.success(`已添加 ${selectedRows.value.length} 首歌曲`)
  tableRef.value?.clearSelection()
}

// ✅ 查看详情
const viewDetail = (row) => {
  currentRow.value = row
  detailVisible.value = true
}

// 从详情页打开编辑
const openEditFromDetail = () => {
  detailVisible.value = false
  openEditDialog(currentRow.value)
}

// ✅ 处理下拉菜单命令
const handleCommand = (cmd, row) => {
  switch (cmd) {
    case 'edit':
      openEditDialog(row)
      break
    case 'delete':
      del(row.id)
      break
    case 'fetch_lyrics':
      fetchSingle(row, 'lyrics')
      break
    case 'fetch_cover':
      fetchSingle(row, 'cover')
      break
  }
}

const fetchSingle = async (row, type) => {
  try {
    const apiFunc = type === 'lyrics' ? api.fetchLyrics : api.fetchCover
    await apiFunc(row.id)
    ElMessage.success(`已启动获取${type === 'lyrics' ? '歌词' : '封面'}任务`)
    loadData()
  } catch (e) {
    ElMessage.error('启动失败：' + (e.response?.data?.message || e.message))
  }
}

const openEditDialog = (row) => {
  editForm.value = {
    id: row.id,
    title: row.title || '',
    artist: row.artist || '',
    album: row.album || '',
    year: row.year || new Date().getFullYear()
  }
  editDialogVisible.value = true
}

const saveEdit = async () => {
  saving.value = true
  try {
    await api.updateMusic(editForm.value.id, editForm.value)
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    if (detailVisible.value) {
      // 如果是在详情页编辑，刷新当前行数据
      loadData().then(() => {
        const updated = tableData.value.find(i => i.id === editForm.value.id)
        if (updated) currentRow.value = updated
      })
    } else {
      loadData()
    }
  } catch (e) {
    ElMessage.error('保存失败：' + (e.response?.data?.message || e.message))
  } finally {
    saving.value = false
  }
}

const del = async (id) => {
  try {
    await ElMessageBox.confirm('确定删除？此操作不可恢复。', '警告', { type: 'warning' })
    await api.deleteMusic(id)
    ElMessage.success('删除成功')
    loadData()
  } catch {}
}

const batchDelete = async () => {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 首音乐？`, '警告', { type: 'warning' })
    await Promise.all(selectedIds.value.map(id => api.deleteMusic(id)))
    ElMessage.success(`成功删除 ${selectedIds.value.length} 首`)
    loadData()
  } catch {}
}

const handleBatch = async (type) => {
  try {
    const map = {
      'lyrics': { fn: api.batchFetchLyrics, msg: '批量歌词' },
      'covers': { fn: api.batchFetchCovers, msg: '批量封面' },
      'all': { fn: api.batchFetchAll, msg: '批量获取全部' }
    }
    await map[type].fn()
    ElMessage.success(`${map[type].msg}任务已启动`)
  } catch (e) {
    if (e.response?.status === 409) ElMessage.warning('任务正在运行中')
    else ElMessage.error('启动失败：' + (e.response?.data?.message || e.message))
  }
}

onMounted(loadData)
</script>

<style scoped lang="scss">
.music-library-page {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: 100%;
}

.box-card {
  border-radius: 8px;
  
  /* ✅ 关键：增加卡片内部padding，让内容不贴边 */
  :deep(.el-card__body) {
    padding: 24px;
  }
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 15px;
  margin-bottom: 10px; /* 增加工具栏和表格的间距 */

  .search-group {
    display: flex;
    gap: 10px;
  }

  .batch-group {
    display: flex;
    gap: 10px;
    align-items: center;
    flex-wrap: wrap;
  }
}

/* ✅ 关键：全局调整表格样式 */
:deep(.el-table) {
  /* 增加行高 */
  --el-table-row-height: 54px; 
  
  .el-table__row {
    height: 54px; /* 强制行高 */
    
    .cell {
      /* 增加单元格内边距，让文字不贴边 */
      padding: 0 12px; 
      line-height: 24px;
      
      /* 优化字体颜色和大小 */
      color: #606266;
      font-size: 14px; 
    }
  }

  /* 表头样式优化 */
  th.el-table__cell {
    background-color: #fafafa !important; /* 更淡的背景色 */
    color: #909399;
    font-weight: 600;
    font-size: 14px;
    height: 46px; /* 表头高度 */
  }
}

.table-header-gray {
  background-color: #fafafa !important;
  color: #909399;
  font-weight: 600;
}

.song-title {
  display: flex;
  align-items: center;
  gap: 6px; /* 标题和标签的间距 */
  
  .name {
    font-weight: 500;
    color: #303133;
    font-size: 14px;
  }
  
  /* 优化标签大小，使其与文字更协调 */
  :deep(.el-tag) {
    transform: scale(0.9); /* 稍微缩小标签 */
    margin-left: 2px;
  }
}

/* 时长样式 */
.duration-text { 
  font-family: 'Consolas', 'Monaco', monospace; 
  color: #909399; /* 灰色表示未解析 */
  font-size: 13px; 
  margin-right: 5px; 
  
  &.is-zero {
    font-style: italic;
  }
}

.warning-icon { 
  color: #E6A23C; 
  vertical-align: middle; 
  font-size: 15px; 
  cursor: help;
  margin-left: 2px;
}

/* 操作按钮样式优化 */
.action-buttons { 
  display: flex; 
  align-items: center; 
  justify-content: center; 
  gap: 6px; /* 按钮间距 */
  
  :deep(.el-button) {
    /* 如果是圆形按钮，不需要额外 padding，Element Plus 会自动处理 */
    /* 如果是带文字的按钮，保持原有 padding */
    font-size: 13px;
    
    .el-icon {
      margin-right: 0; /* 圆形按钮不需要右边距 */
    }
  }
  
  /* 针对带文字的“查看”按钮特殊处理 */
  :deep(.el-button:not([circle])) {
    padding: 6px 10px;
    .el-icon {
      margin-right: 4px;
    }
  }
}

.pagination {
  margin-top: 24px; /* 增加分页上方的间距 */
  display: flex;
  justify-content: flex-end;
  
  :deep(.el-pagination) {
    --el-pagination-font-size: 14px;
  }
}

/* 详情页样式 (保持不变或微调) */
.detail-content { line-height: 1.8; }
.detail-header { display: flex; gap: 24px; margin-bottom: 20px; }
.cover-area { 
  flex-shrink: 0; 
  width: 160px; /* 稍微加大封面 */
  height: 160px; 
  display: flex; 
  align-items: center; 
  justify-content: center; 
  background: #f5f7fa; 
  border-radius: 8px; 
  overflow: hidden;
  box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05); /* 增加阴影更有质感 */
}
.cover-image { width: 100%; height: 100%; object-fit: cover; }
.cover-placeholder { text-align: center; color: #909399; display: flex; flex-direction: column; align-items: center; }
.info-area { flex: 1; h3 { margin: 0 0 15px 0; font-size: 20px; color: #303133; font-weight: 600; } p { margin: 10px 0; color: #606266; font-size: 14px; } }
.detail-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; font-size: 13px; color: #606266; background: #fafafa; padding: 15px; border-radius: 6px; }
.text-ellipsis { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; display: block; max-width: 300px; }
.text-warning { color: #E6A23C; font-weight: bold; }
.error-msg { margin-top: 15px; }
</style>

<style>
/* 全局样式微调 */
.detail-dialog .el-dialog__body { padding: 20px 24px; }
</style>