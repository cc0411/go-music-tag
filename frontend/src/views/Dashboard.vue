<template>
  <div class="dashboard-container">
    <!-- 顶部标题栏 -->
    <div class="page-header">
      <h2 class="page-title">仪表盘</h2>
      <el-button circle @click="loadStats">
        <el-icon><Refresh /></el-icon>
      </el-button>
    </div>

    <!-- 1. 统计卡片区 -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <!-- ✅ 修复：使用 Headset (耳机) -->
            <div class="icon-box" style="background: linear-gradient(135deg, #7F7FD5, #86A8E7);">
              <el-icon :size="32" color="#fff"><Headset /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ stats.total || 0 }}</div>
              <div class="label">音乐总数</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <!-- ✅ 修复：使用 User (用户) -->
            <div class="icon-box" style="background: linear-gradient(135deg, #11998e, #38ef7d);">
              <el-icon :size="32" color="#fff"><User /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ stats.top_artists?.length || 0 }}</div>
              <div class="label">艺术家</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <!-- ✅ 修复：使用 Collection (收藏/专辑) -->
            <div class="icon-box" style="background: linear-gradient(135deg, #F2994A, #F2C94C);">
              <el-icon :size="32" color="#fff"><Collection /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ stats.top_albums?.length || 0 }}</div>
              <div class="label">专辑</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <!-- ✅ 修复：使用 PictureFilled (图片/流派) -->
            <div class="icon-box" style="background: linear-gradient(135deg, #FF6B6B, #FFE66D);">
              <el-icon :size="32" color="#fff"><PictureFilled /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ stats.top_genres?.length || 0 }}</div>
              <div class="label">流派</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 2. 热门列表区 -->
    <el-row :gutter="20" class="lists-section">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span class="header-title">🏆 热门艺术家</span>
            </div>
          </template>
          <div v-if="stats.top_artists?.length" class="list-container">
            <div v-for="(item, i) in stats.top_artists" :key="i" class="list-item">
              <span class="rank" :class="{ 'top-3': i < 3 }">{{ i + 1 }}</span>
              <span class="name">{{ item.name }}</span>
              <span class="count">{{ item.count }} 首</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="80" />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span class="header-title">🎨 热门流派</span>
            </div>
          </template>
          <div v-if="stats.top_genres?.length" class="list-container">
            <div v-for="(item, i) in stats.top_genres" :key="i" class="list-item">
              <span class="rank" :class="{ 'top-3': i < 3 }">{{ i + 1 }}</span>
              <span class="name">{{ item.name }}</span>
              <span class="count">{{ item.count }} 首</span>
            </div>
          </div>
          <el-empty v-else description="暂无数据" :image-size="80" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 3. 快速操作区 -->
    <el-card shadow="never" class="quick-actions">
      <template #header>
        <div class="card-header">
          <span class="header-title">🚀 快速操作</span>
        </div>
      </template>
      <el-row :gutter="20">
        <el-col :span="6">
          <div class="action-btn" @click="goToScan">
            <div class="action-icon" style="background: #e0f2fe;">
              <el-icon :size="32" color="#0ea5e9"><Promotion /></el-icon>
            </div>
            <span class="action-text">开始扫描</span>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="action-btn" @click="goToWebDAV">
            <div class="action-icon" style="background: #f3f4f6;">
              <el-icon :size="32" color="#4b5563"><Cloudy /></el-icon>
            </div>
            <span class="action-text">配置 WebDAV</span>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="action-btn" @click="goToMusic">
            <div class="action-icon" style="background: #ede9fe;">
              <el-icon :size="32" color="#7c3aed"><Headset /></el-icon>
            </div>
            <span class="action-text">查看音乐</span>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="action-btn" @click="goToPlayer">
            <div class="action-icon" style="background: #f3f4f6;">
              <el-icon :size="32" color="#4b5563"><Headset /></el-icon>
            </div>
            <span class="action-text">打开播放器</span>
          </div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Headset, User, Collection, PictureFilled, Refresh, 
  Promotion, Cloudy
} from '@element-plus/icons-vue'
import { api } from '@/api'

const router = useRouter()
const stats = ref({})

const loadStats = async () => {
  try {
    const res = await api.getStats()
    if (res.code === 0) {
      stats.value = res.data || {}
    }
  } catch (e) {
    console.error('加载统计失败:', e)
  }
}

const goToScan = () => router.push('/scan')
const goToWebDAV = () => router.push('/webdav')
const goToMusic = () => router.push('/music')
const goToPlayer = () => router.push('/player')

onMounted(() => {
  loadStats()
})
</script>

<style scoped>
.dashboard-container {
  padding: 20px;
  background-color: #f8fafc;
  min-height: calc(100vh - 140px);
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}
.stat-cards { margin-bottom: 24px; }
.stat-card {
  border: none;
  border-radius: 12px;
  transition: transform 0.2s;
}
.stat-card:hover { transform: translateY(-4px); }
.card-content {
  display: flex;
  align-items: center;
  gap: 16px;
}
.icon-box {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.info { flex: 1; }
.value {
  font-size: 32px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1.2;
}
.label {
  font-size: 14px;
  color: #64748b;
  margin-top: 4px;
}
.lists-section { margin-bottom: 24px; }
.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}
.list-container {
  max-height: 300px;
  overflow-y: auto;
}
.list-item {
  display: flex;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
  transition: background 0.2s;
}
.list-item:last-child { border-bottom: none; }
.list-item:hover {
  background-color: #f8fafc;
  padding-left: 8px;
  border-radius: 6px;
}
.rank {
  width: 24px;
  height: 24px;
  background-color: #f1f5f9;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  margin-right: 12px;
  flex-shrink: 0;
}
.rank.top-3 {
  background-color: #fef3c7;
  color: #d97706;
}
.name {
  flex: 1;
  font-size: 14px;
  color: #334155;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.count {
  font-size: 13px;
  color: #94a3b8;
  flex-shrink: 0;
}
.quick-actions {
  border: none;
  border-radius: 12px;
}
.action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}
.action-btn:hover {
  background-color: #fff;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  border-color: #e2e8f0;
  transform: translateY(-2px);
}
.action-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
}
.action-text {
  font-size: 14px;
  color: #475569;
  font-weight: 500;
}
.list-container::-webkit-scrollbar { width: 6px; }
.list-container::-webkit-scrollbar-track {
  background: #f1f5f9;
  border-radius: 3px;
}
.list-container::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 3px;
}
.list-container::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}
</style>