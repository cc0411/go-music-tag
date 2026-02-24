<template>
  <div class="player-bar">
    <!-- 左侧：歌曲信息 -->
    <div class="track-info">
      <img 
        :src="currentTrack?.coverUrl || ''" 
        class="cover" 
        @error="onCoverError"
      />
      <div class="text">
        <div class="title">{{ currentTrack?.title || currentTrack?.file_name || '未播放' }}</div>
        <div class="artist">{{ currentTrack?.artist || '-' }}</div>
      </div>
    </div>

    <!-- 中间：控制按钮 -->
    <div class="controls">
      <el-button circle @click="store.playPrev">
        <el-icon><Back /></el-icon>
      </el-button>
      <el-button circle size="large" @click="store.togglePlay">
        <el-icon>
          <component :is="store.isPlaying ? 'VideoPause' : 'VideoPlay'" />
        </el-icon>
      </el-button>
      <el-button circle @click="store.playNext">
        <el-icon><Right /></el-icon>
      </el-button>
    </div>

    <!-- 右侧：进度条 -->
    <div class="progress-area">
      <span class="time-text">{{ formatTime(currentTime) }}</span>
      <el-slider 
        v-model="progress" 
        :format-tooltip="formatTime" 
        @change="onSeek" 
        style="width: 300px; margin: 0 10px;" 
      />
      <span class="time-text">{{ formatTime(duration) }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { Back, Right, VideoPlay, VideoPause } from '@element-plus/icons-vue'
import { usePlayerStore } from '@/stores/player'

const store = usePlayerStore()
const currentTime = ref(0)
const duration = ref(0)
const progress = ref(0)

// ✅ 关键：通过 computed 响应式获取当前歌曲
const currentTrack = computed(() => {
  const track = store.currentTrack
  // 调试用：每次变化都会打印
  if (track) {
    console.log('🖼️ AudioPlayer 渲染封面 URL:', track.coverUrl)
  }
  return track
})

// 默认占位图 (Base64 SVG)
const defaultCover = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjY2NjIiBzdHJva2Utd2lkdGg9IjIiPjxyZWN0IHg9IjMiIHk9IjMiIHdpZHRoPSIxOCIgaGVpZ2h0PSIxOCIgcng9IjIiIHJ5PSIyIi8+PGNpcmNsZSBjeD0iOC41IiBjeT0iOC41IiByPSIxLjUiLz48cG9seWxpbmUgcG9pbnRzPSIyMSAxNSAxNiAxMCA1IDIxIi8+PC9zdmc+'

// 图片加载失败处理
const onCoverError = (e) => {
  e.target.src = defaultCover
}

onMounted(() => {
  // 1. 初始化加载播放列表
  store.loadPlaylist()
  
  // 2. 监听音频时间更新
  store.audio.addEventListener('timeupdate', () => {
    currentTime.value = store.audio.currentTime
    duration.value = store.audio.duration || 0
    if (duration.value > 0) {
      progress.value = (currentTime.value / duration.value) * 100
    }
  })

  // 3. 监听当前歌曲变化 (调试用)
  watch(() => store.currentTrack, (newTrack, oldTrack) => {
    console.log('👀 监听到歌曲变化:', oldTrack?.title, '->', newTrack?.title)
  }, { immediate: true })
})

// 拖拽进度条
const onSeek = (val) => {
  if (duration.value) {
    store.seek((val / 100) * duration.value)
  }
}

// 格式化时间
const formatTime = (s) => {
  if (!s || isNaN(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}
</script>

<style scoped>
.player-bar {
  display: flex;
  align-items: center;
  justify-content: space-between; /* 改为 space-between 确保两端对齐 */
  height: 100%;
  padding: 0 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.track-info {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 240px;
  overflow: hidden;
}

.cover {
  width: 56px;
  height: 56px;
  border-radius: 8px;
  object-fit: cover;
  background-color: #f0f0f0;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.text {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0; /* 允许 flex 子项收缩 */
}

.title {
  font-weight: 600;
  font-size: 15px;
  color: #1f2937;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.artist {
  font-size: 13px;
  color: #6b7280;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.progress-area {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  max-width: 500px;
  font-size: 13px;
  color: #6b7280;
}

.time-text {
  min-width: 40px;
  text-align: center;
  font-variant-numeric: tabular-nums; /* 等宽数字，防止跳动 */
}

/* 移动端适配 */
@media (max-width: 768px) {
  .player-bar {
    padding: 0 10px;
  }
  .track-info {
    width: 120px;
  }
  .cover {
    width: 40px;
    height: 40px;
  }
  .progress-area {
    max-width: 200px;
  }
  .time-text {
    font-size: 11px;
    min-width: 30px;
  }
}
</style>