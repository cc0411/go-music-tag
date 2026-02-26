<template>
  <!-- ✅ 根元素直接 fixed 定位 -->
  <div class="player-bar-fixed">
    <!-- 左侧：歌曲信息 -->
    <div class="track-info">
      <img :src="currentTrack?.coverUrl || ''" class="cover" @error="onCoverError" />
      <div class="text">
        <div class="title">{{ currentTrack?.title || currentTrack?.file_name || '未播放' }}</div>
        <div class="artist">{{ currentTrack?.artist || '-' }}</div>
      </div>
    </div>

    <!-- 中间：控制按钮 -->
    <div class="controls">
      <el-button circle @click="store.playPrev"><el-icon><Back /></el-icon></el-button>
      <el-button text circle @click="store.toggleMode" :title="modeText">
        <el-icon><component :is="modeIcon" /></el-icon>
      </el-button>
      <el-button circle size="large" @click="store.togglePlay">
        <el-icon><component :is="store.isPlaying ? 'VideoPause' : 'VideoPlay'" /></el-icon>
      </el-button>
      <el-button circle @click="store.playNext"><el-icon><Right /></el-icon></el-button>
    </div>

    <!-- 右侧：进度条 + 列表按钮 -->
    <div class="progress-area">
      <span class="time-text">{{ formatTime(currentTime) }}</span>
      <el-slider v-model="progress" :format-tooltip="formatTime" @change="onSeek" class="custom-slider" />
      <span class="time-text">{{ formatTime(duration) }}</span>
      
      <el-button text circle class="playlist-btn" @click="togglePlaylist">
        <el-icon><List /></el-icon>
        <span class="badge" v-if="store.playlist.length > 0">{{ store.playlist.length }}</span>
      </el-button>
    </div>
  </div>

  <!-- 🎵 播放列表抽屉 (Fixed 定位到视口) -->
  <transition name="slide-up">
    <div v-if="showPlaylist" class="playlist-overlay" @click="showPlaylist = false">
      <div class="playlist-drawer" @click.stop>
        <!-- 头部 -->
        <div class="drawer-header">
          <div class="header-left">
            <span class="title-text">播放队列</span>
            <span class="count-tag">共 {{ store.playlist.length }} 首</span>
          </div>
          <div class="header-right">
            <el-button link type="danger" size="small" @click="store.clearPlaylist" v-if="store.playlist.length">
              <el-icon><Delete /></el-icon>
            </el-button>
            <el-button link size="small" @click="showPlaylist = false">
              <el-icon><Close /></el-icon>
            </el-button>
          </div>
        </div>

        <!-- 列表 -->
        <div class="drawer-body">
          <div v-for="(track, index) in store.playlist" :key="track.id || index" class="track-item" :class="{ active: currentTrack?.id === track.id }" @click="playTrack(track)">
            <div class="item-cover-box">
              <img :src="track.coverUrl || defaultCover" class="item-cover" @error="(e)=>e.target.src=defaultCover" />
              <div class="playing-icon" v-if="currentTrack?.id === track.id && store.isPlaying">
                <el-icon class="is-loading"><VideoPlay /></el-icon>
              </div>
            </div>
            <div class="item-info">
              <div class="item-title">{{ track.title || track.file_name || '未知曲目' }}</div>
              <div class="item-artist">{{ track.artist || '未知艺术家' }}</div>
            </div>
            <div class="item-time">{{ formatTime(track.duration || 0) }}</div>
          </div>
          <el-empty v-if="!store.playlist.length" description="列表为空" :image-size="60" />
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Back, Right, VideoPlay, VideoPause, List, Connection, Refresh, Lock, Delete, Close } from '@element-plus/icons-vue'
import { usePlayerStore } from '@/stores/player'

const store = usePlayerStore()
const currentTime = ref(0)
const duration = ref(0)
const progress = ref(0)
const showPlaylist = ref(false)

const defaultCover = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjY2NjIiBzdHJva2Utd2lkdGg9IjIiPjxyZWN0IHg9IjMiIHk9IjMiIHdpZHRoPSIxOCIgaGVpZ2h0PSIxOCIgcng9IjIiIHJ5PSIyIi8+PGNpcmNsZSBjeD0iOC41IiBjeT0iOC41IiByPSIxLjUiLz48cG9seWxpbmUgcG9pbnRzPSIyMSAxNSAxNiAxMCA1IDIxIi8+PC9zdmc+'

const currentTrack = computed(() => store.currentTrack)
const modeIcon = computed(() => store.playMode === 'random' ? Connection : store.playMode === 'single' ? Lock : Refresh)
const modeText = computed(() => store.playMode === 'random' ? '随机播放' : store.playMode === 'single' ? '单曲循环' : '顺序播放')

const onCoverError = (e) => { e.target.src = defaultCover }
const togglePlaylist = () => { showPlaylist.value = !showPlaylist.value }
const playTrack = (track) => { store.playTrack(track) }

onMounted(() => {
  store.loadPlaylist()
  store.audio.addEventListener('timeupdate', () => {
    currentTime.value = store.audio.currentTime
    duration.value = store.audio.duration || 0
    if (duration.value > 0) progress.value = (currentTime.value / duration.value) * 100
  })
})

const onSeek = (val) => { if (duration.value) store.seek((val / 100) * duration.value) }
const formatTime = (s) => {
  if (!s || isNaN(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}
</script>

<style scoped lang="scss">
.player-bar-fixed {
  position: fixed;
  bottom: 0;
  left: 220px; /* 与侧边栏对齐 */
  right: 0;
  height: 90px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: #fff;
  box-shadow: 0 -2px 8px rgba(0,0,0,0.05);
  box-sizing: border-box;
  z-index: 100;
  overflow: hidden;
}

.track-info {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 240px;
  overflow: hidden;
  flex-shrink: 0;
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
  min-width: 0;
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
  flex-shrink: 0;
}
.progress-area {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  max-width: 800px;
  font-size: 13px;
  color: #6b7280;
  justify-content: flex-end;
  overflow: hidden;
}
.time-text {
  min-width: 45px;
  text-align: center;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.custom-slider {
  flex: 1;
  margin: 0 8px;
  --el-slider-main-bg-color: #409EFF;
  --el-slider-runway-bg-color: #e5e7eb;
  :deep(.el-slider__runway) { height: 4px; }
  :deep(.el-slider__bar) { height: 4px; }
}
.playlist-btn {
  position: relative;
  margin-left: 8px;
  flex-shrink: 0;
  .badge {
    position: absolute;
    top: -2px;
    right: -2px;
    background: #f56c6c;
    color: white;
    font-size: 10px;
    padding: 2px 5px;
    border-radius: 10px;
    font-weight: bold;
    transform: scale(0.9);
    z-index: 10;
  }
}

/* --- 播放列表抽屉 --- */
.playlist-overlay {
  position: fixed;
  bottom: 100px; /* 播放器高度 90px + 10px 间距 */
  right: 30px;
  width: 360px;
  height: 500px;
  z-index: 2000;
  pointer-events: none;
  display: flex;
  flex-direction: column;
}

.playlist-drawer {
  width: 100%;
  height: 100%;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  box-shadow: 0 5px 25px rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(229, 231, 235, 0.6);
  display: flex;
  flex-direction: column;
  pointer-events: auto;
  overflow: hidden;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f3f4f6;
  background: #fafafa;
  flex-shrink: 0;
  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;
    .title-text { font-weight: 600; font-size: 15px; color: #1f2937; }
    .count-tag { font-size: 12px; color: #6b7280; background: #e5e7eb; padding: 2px 8px; border-radius: 10px; }
  }
  .header-right { display: flex; gap: 4px; }
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  &::-webkit-scrollbar { width: 6px; }
  &::-webkit-scrollbar-thumb { background: #d1d5db; border-radius: 3px; }
  &::-webkit-scrollbar-track { background: transparent; }
}

.track-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
  &:hover { background-color: #f9fafb; }
  &.active { 
    background-color: rgba(64, 158, 255, 0.08); 
    .item-title { color: #409EFF; font-weight: 600; } 
  }
  .item-cover-box {
    position: relative;
    width: 40px;
    height: 40px;
    border-radius: 6px;
    overflow: hidden;
    flex-shrink: 0;
    margin-right: 12px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    .item-cover { width: 100%; height: 100%; object-fit: cover; }
    .playing-icon {
      position: absolute; inset: 0; background: rgba(0,0,0,0.5);
      display: flex; align-items: center; justify-content: center; color: white;
      .el-icon { animation: pulse 1.5s infinite; }
    }
  }
  .item-info {
    flex: 1; min-width: 0; display: flex; flex-direction: column; justify-content: center; overflow: hidden;
    .item-title { font-size: 14px; color: #1f2937; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 2px; }
    .item-artist { font-size: 12px; color: #9ca3af; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  }
  .item-time {
    flex-shrink: 0; margin-left: 12px; font-size: 12px; color: #9ca3af; width: 45px; text-align: right; font-variant-numeric: tabular-nums;
  }
}

.slide-up-enter-active, .slide-up-leave-active { transition: all 0.3s cubic-bezier(0.25, 0.8, 0.5, 1); }
.slide-up-enter-from, .slide-up-leave-to { opacity: 0; transform: translateY(20px) scale(0.95); }
@keyframes pulse { 0% { opacity: 0.6; transform: scale(0.9); } 50% { opacity: 1; transform: scale(1.1); } 100% { opacity: 0.6; transform: scale(0.9); } }
</style>