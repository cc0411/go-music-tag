// src/stores/player.js
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'

export const usePlayerStore = defineStore('player', () => {
  const playlist = ref([])
  const currentTrackIndex = ref(-1)
  const isPlaying = ref(false)
  const audio = new Audio()

  // ✅ 计算属性：安全地获取当前歌曲对象
  const currentTrack = computed(() => {
    if (currentTrackIndex.value >= 0 && currentTrackIndex.value < playlist.value.length) {
      return playlist.value[currentTrackIndex.value]
    }
    return null
  })

  // 加载播放列表
  const loadPlaylist = async () => {
    try {
      const res = await api.getMusicList(1, '') // 加载第一页作为初始列表
      if (res.code === 0) {
        const list = res.data.list || res.data || []
        playlist.value = list.map(item => ({
          ...item,
          playUrl: api.getPlayUrl(item.id),
          coverUrl: api.getCoverUrl(item.id)
        }))
        console.log('📋 播放列表加载完成，共', playlist.value.length, '首')
      }
    } catch (e) {
      console.error('❌ 加载播放列表失败:', e)
    }
  }

  // ✅ 核心修复：播放歌曲
  const playTrack = (track, list = null) => {
    console.log('🎵 准备播放:', track.title || track.file_name, 'ID:', track.id)

    // 情况 A: 传入了新列表 (例如从音乐库页面点击播放)
    if (list) {
      playlist.value = list.map(item => ({
        ...item,
        playUrl: api.getPlayUrl(item.id),
        coverUrl: api.getCoverUrl(item.id)
      }))
      const idx = playlist.value.findIndex(t => t.id === track.id)
      currentTrackIndex.value = idx >= 0 ? idx : 0
      console.log('🔄 使用新列表，找到索引:', currentTrackIndex.value)
    } 
    // 情况 B: 使用当前播放列表
    else {
      const idx = playlist.value.findIndex(t => t.id === track.id)
      if (idx >= 0) {
        currentTrackIndex.value = idx
        console.log('✅ 在当前列表中找到索引:', currentTrackIndex.value)
      } else {
        // 如果当前列表没有这首歌，临时添加
        console.log('⚠️ 歌曲不在列表中，临时添加')
        playlist.value.push({
          ...track,
          playUrl: api.getPlayUrl(track.id),
          coverUrl: api.getCoverUrl(track.id)
        })
        currentTrackIndex.value = playlist.value.length - 1
      }
    }

    // 执行播放
    if (currentTrackIndex.value >= 0 && currentTrack.value) {
      const targetTrack = currentTrack.value
      console.log('🔊 实际播放对象:', targetTrack.title)
      
      audio.src = targetTrack.playUrl
      audio.load() // 必须调用 load 重新加载
      
      audio.play().then(() => {
        isPlaying.value = true
        console.log('▶️ 开始播放成功')
      }).catch(err => {
        console.error('❌ 播放失败:', err)
        isPlaying.value = false
      })
    } else {
      console.error('❌ 错误：索引无效或找不到歌曲对象')
    }
  }

  // 切换播放/暂停
  const togglePlay = () => {
    if (!currentTrack.value && playlist.value.length > 0) {
      playTrack(playlist.value[0])
      return
    }
    
    if (isPlaying.value) {
      audio.pause()
      isPlaying.value = false
    } else {
      audio.play().catch(e => console.error('播放出错:', e))
      isPlaying.value = true
    }
  }

  // 下一首
  const playNext = () => {
    if (playlist.value.length === 0) return
    let nextIdx = currentTrackIndex.value + 1
    if (nextIdx >= playlist.value.length) nextIdx = 0 // 循环
    console.log('⏭️ 下一首索引:', nextIdx)
    playTrack(playlist.value[nextIdx])
  }

  // 上一首
  const playPrev = () => {
    if (playlist.value.length === 0) return
    let prevIdx = currentTrackIndex.value - 1
    if (prevIdx < 0) prevIdx = playlist.value.length - 1 // 循环
    playTrack(playlist.value[prevIdx])
  }

  // 拖拽进度
  const seek = (time) => {
    if (audio.duration) {
      audio.currentTime = time
    }
  }

  // 绑定音频原生事件
  audio.addEventListener('ended', () => {
    console.log('🏁 播放结束，自动下一首')
    playNext()
  })
  
  audio.addEventListener('pause', () => {
    isPlaying.value = false
  })
  
  audio.addEventListener('playing', () => {
    isPlaying.value = true
  })

  audio.addEventListener('error', (e) => {
    console.error('💥 音频错误事件:', e)
    isPlaying.value = false
  })

  return {
    playlist,
    currentTrack,
    currentTrackIndex,
    isPlaying,
    audio,
    loadPlaylist,
    playTrack,
    togglePlay,
    playNext,
    playPrev,
    seek
  }
})