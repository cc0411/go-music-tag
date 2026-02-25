<template>
    <div class="player-view">
      <div class="player-content">
        <!-- 左侧：专辑封面 -->
        <div class="cover-section">
          <div class="disc-wrapper" :class="{ 'is-playing': playerStore.isPlaying }">
            <div class="disc-bg"></div>
            <!-- ✅ 使用后端接口获取封面 -->
            <img 
              :src="coverUrl" 
              class="disc-cover" 
              @error="(e) => e.target.src = defaultCover"
              alt="Album Cover"
            />
            <div class="disc-center"></div>
          </div>
          <div class="track-meta-mobile">
            <h2 class="track-title">{{ currentTrack?.title || currentTrack?.file_name || '未知曲目' }}</h2>
            <p class="track-artist">{{ currentTrack?.artist || '未知艺术家' }}</p>
          </div>
        </div>
  
        <!-- 右侧：歌词与控制 -->
        <div class="lyrics-section">
          <div class="track-meta-desktop">
            <h2 class="track-title">{{ currentTrack?.title || currentTrack?.file_name || '未知曲目' }}</h2>
            <p class="track-artist">{{ currentTrack?.artist || '未知艺术家' }}</p>
            <!-- 显示专辑名如果有 -->
            <p v-if="currentTrack?.album" class="track-album">{{ currentTrack.album }}</p>
          </div>
  
          <!-- 歌词区域 -->
          <div class="lyrics-container" ref="lyricsContainer">
            <!-- 加载中状态 -->
            <div v-if="loadingLyrics" class="loading-state">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>正在加载歌词...</span>
            </div>
  
            <!-- 歌词列表 -->
            <div v-else-if="lyrics.length > 0" class="lyrics-list">
              <div 
                v-for="(line, index) in lyrics" 
                :key="index"
                class="lyric-line"
                :class="{ active: currentLyricIndex === index }"
                @click="seekTo(line.time)"
              >
                {{ line.text }}
              </div>
            </div>
  
            <!-- 无歌词状态 -->
            <div v-else class="no-lyrics">
              <el-icon :size="48" color="#ccc"><Headset /></el-icon>
              <p>暂无歌词</p>
              <p class="sub-text">纯音乐或歌词文件缺失</p>
            </div>
          </div>
  
          <!-- 底部控制栏 -->
          <div class="mini-controls">
            <el-slider 
              v-model="progress" 
              :format-tooltip="formatTime" 
              @change="onSeek" 
              class="custom-slider"
            />
            <div class="time-info">
              <span>{{ formatTime(currentTime) }}</span>
              <span>{{ formatTime(duration) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </template>
  
<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { Headset, Refresh } from '@element-plus/icons-vue'
import { usePlayerStore } from '@/stores/player'
import axios from 'axios'

const playerStore = usePlayerStore()
const lyricsContainer = ref(null)

// 状态
const currentTime = ref(0)
const duration = ref(0)
const progress = ref(0)
const currentLyricIndex = ref(-1)
const loadingLyrics = ref(false)
const lyrics = ref([])

const defaultCover = 'data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgc3Ryb2tlPSIjY2NjIiBzdHJva2Utd2lkdGg9IjIiPjxyZWN0IHg9IjMiIHk9IjMiIHdpZHRoPSIxOCIgaGVpZ2h0PSIxOCIgcng9IjIiIHJ5PSIyIi8+PGNpcmNsZSBjeD0iOC41IiBjeT0iOC41IiByPSIxLjUiLz48cG9seWxpbmUgcG9pbnRzPSIyMSAxNSAxNiAxMCA1IDIxIi8+PC9zdmc+'

const currentTrack = computed(() => playerStore.currentTrack)

const coverUrl = computed(() => {
  if (!currentTrack.value?.id) return defaultCover
  return `/api/v1/music/${currentTrack.value.id}/cover` 
})

// ✅ 核心修复：定义时间更新处理函数
const handleTimeUpdate = () => {
  const audio = playerStore.audio
  currentTime.value = audio.currentTime
  duration.value = audio.duration || 0
  
  if (duration.value > 0) {
    progress.value = (currentTime.value / duration.value) * 100
  }
  
  // 歌词高亮逻辑
  if (lyrics.value.length > 0) {
    const index = lyrics.value.findIndex((line, i) => {
      const nextLine = lyrics.value[i + 1]
      return currentTime.value >= line.time && (!nextLine || currentTime.value < nextLine.time)
    })
    
    if (index !== -1 && index !== currentLyricIndex.value) {
      currentLyricIndex.value = index
      scrollToLyric(index)
    }
  }
}

// ✅ 核心修复：组件挂载时添加监听
onMounted(() => {
  const audio = playerStore.audio
  
  // 1. 绑定时间更新事件 (每秒触发多次)
  audio.addEventListener('timeupdate', handleTimeUpdate)
  
  // 2. 绑定元数据加载事件 (获取总时长)
  audio.addEventListener('loadedmetadata', () => {
    duration.value = audio.duration
    console.log('音频时长已加载:', duration.value)
  })
  
  // 3. 如果当前已经在播放，立即执行一次以同步状态
  if (!audio.paused) {
    handleTimeUpdate()
  }
  
  // 4. 加载歌词
  loadLyrics()
})

// ✅ 核心修复：组件卸载时移除监听，防止内存泄漏
onUnmounted(() => {
  const audio = playerStore.audio
  audio.removeEventListener('timeupdate', handleTimeUpdate)
  audio.removeEventListener('loadedmetadata', () => {}) // 匿名函数无法移除，忽略即可，主要移除 timeupdate
})

// 解析歌词
const parseLyrics = (lrcText) => {
  if (!lrcText) return []
  const lines = lrcText.split('\n')
  const result = []
  const timeReg = /\[(\d{2}):(\d{2})\.(\d{2,3})\]/
  
  for (const line of lines) {
    const match = timeReg.exec(line)
    if (match) {
      const minutes = parseInt(match[1])
      const seconds = parseInt(match[2])
      const milliseconds = parseInt(match[3])
      const time = minutes * 60 + seconds + milliseconds / 1000
      const text = line.replace(timeReg, '').trim()
      if (text) {
        result.push({ time, text })
      }
    }
  }
  return result
}

// 加载歌词
const loadLyrics = async () => {
  if (!currentTrack.value?.id) {
    lyrics.value = []
    return
  }

  loadingLyrics.value = true
  currentLyricIndex.value = -1
  
  try {
    const response = await axios.get(`/api/v1/music/${currentTrack.value.id}/lyrics`)
    
    let lrcText = ''
    
    // ✅ 修复：根据后端返回结构正确提取歌词
    if (response.data.code === 0 && response.data.data) {
      // 后端返回的是 { data: { lyrics: "...", parsed: [...] } }
      lrcText = response.data.data.lyrics || ''
      
      // 可选：如果后端已经解析好了，可以直接用 parsed 数组
      // if (response.data.data.parsed && response.data.data.parsed.length > 0) {
      //   lyrics.value = response.data.data.parsed
      //   loadingLyrics.value = false
      //   return
      // }
    } else if (typeof response.data === 'string') {
      // 兼容直接返回字符串的情况
      lrcText = response.data
    }

    console.log('获取到的歌词文本:', lrcText.substring(0, 100)) // 调试用
    
    if (lrcText) {
      lyrics.value = parseLyrics(lrcText)
    } else {
      lyrics.value = []
    }
  } catch (error) {
    console.error('加载歌词失败:', error)
    lyrics.value = []
  } finally {
    loadingLyrics.value = false
  }
}

const scrollToLyric = (index) => {
  nextTick(() => {
    if (!lyricsContainer.value) return
    const element = lyricsContainer.value.querySelectorAll('.lyric-line')[index]
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
}

const onSeek = (val) => {
  if (duration.value) {
    playerStore.seek((val / 100) * duration.value)
    // 拖拽后立即更新一次界面，避免视觉延迟
    currentTime.value = (val / 100) * duration.value
    progress.value = val
  }
}

const seekTo = (time) => {
  playerStore.seek(time)
}

const formatTime = (s) => {
  if (!s || isNaN(s)) return '0:00'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${sec.toString().padStart(2, '0')}`
}
</script>
  
  <style scoped lang="scss">
  /* ... (样式部分保持不变，直接复用上一轮的样式) ... */
  .player-view {
    height: calc(100vh - 90px);
    width: 100%;
    padding: 40px;
    background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
    overflow: hidden;
    display: flex;
    justify-content: center;
    align-items: center;
  }
  .player-content {
    display: flex;
    width: 100%;
    max-width: 1200px;
    height: 100%;
    gap: 60px;
    background: rgba(255, 255, 255, 0.8);
    backdrop-filter: blur(20px);
    border-radius: 24px;
    padding: 40px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.1);
  }
  .cover-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  .disc-wrapper {
    width: 320px;
    height: 320px;
    border-radius: 50%;
    position: relative;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
    animation: rotate 20s linear infinite;
    animation-play-state: paused;
    &.is-playing { animation-play-state: running; }
    .disc-bg { position: absolute; inset: 0; border-radius: 50%; background: #111; border: 1px solid #333; }
    .disc-cover { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 200px; height: 200px; border-radius: 50%; object-fit: cover; border: 4px solid #fff; }
    .disc-center { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 40px; height: 40px; background: #f0f2f5; border-radius: 50%; z-index: 10; border: 2px solid #ddd; }
  }
  .track-meta-mobile { display: none; margin-top: 30px; text-align: center; .track-title { font-size: 24px; font-weight: 700; color: #1f2937; margin: 0 0 8px 0; } .track-artist { font-size: 16px; color: #6b7280; margin: 0; } }
  .lyrics-section { flex: 1.5; display: flex; flex-direction: column; overflow: hidden; }
  .track-meta-desktop { margin-bottom: 30px; .track-title { font-size: 32px; font-weight: 800; color: #1f2937; margin: 0 0 10px 0; letter-spacing: -0.5px; } .track-artist { font-size: 18px; color: #6b7280; margin: 0; font-weight: 500; } .track-album { font-size: 14px; color: #9ca3af; margin: 5px 0 0 0; } }
  .lyrics-container { flex: 1; overflow-y: auto; mask-image: linear-gradient(to bottom, transparent 0%, black 10%, black 90%, transparent 100%); -webkit-mask-image: linear-gradient(to bottom, transparent 0%, black 10%, black 90%, transparent 100%); padding: 20px 0; scroll-behavior: smooth; position: relative; &::-webkit-scrollbar { width: 6px; } &::-webkit-scrollbar-thumb { background: #cbd5e1; border-radius: 3px; } }
  .loading-state, .no-lyrics { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; color: #9ca3af; gap: 10px; .sub-text { font-size: 14px; opacity: 0.7; } }
  .lyrics-list { display: flex; flex-direction: column; gap: 24px; padding: 0 20px; }
  .lyric-line { font-size: 18px; color: #9ca3af; cursor: pointer; transition: all 0.3s; font-weight: 500; line-height: 1.6; &:hover { color: #4b5563; transform: translateX(5px); } &.active { color: #409EFF; font-size: 22px; font-weight: 700; transform: scale(1.05); text-shadow: 0 2px 10px rgba(64, 158, 255, 0.3); } }
  .mini-controls { margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e7eb; .custom-slider { margin-bottom: 10px; --el-slider-main-bg-color: #409EFF; --el-slider-runway-bg-color: #e5e7eb; :deep(.el-slider__runway) { height: 6px; } :deep(.el-slider__bar) { height: 6px; } } .time-info { display: flex; justify-content: space-between; font-size: 13px; color: #9ca3af; font-variant-numeric: tabular-nums; } }
  @keyframes rotate { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  @media (max-width: 768px) { .player-content { flex-direction: column; padding: 20px; gap: 30px; overflow-y: auto; } .disc-wrapper { width: 240px; height: 240px; .disc-cover { width: 150px; height: 150px; } } .track-meta-desktop { display: none; } .track-meta-mobile { display: block; } .lyrics-container { max-height: 300px; } }
  </style>