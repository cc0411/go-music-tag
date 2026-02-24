<!-- src/views/PlayerView.vue -->
<template>
    <div class="player-view">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>🎵 全屏播放器</span>
          </div>
        </template>
        <div style="text-align: center; padding: 40px;">
          <el-icon :size="60" color="#409EFF"><Headset /></el-icon>
          <h2>请使用底部全局播放器</h2>
          <p>当前页面主要用于展示播放列表和歌词详情。</p>
          <p style="color: #999; margin-top: 10px;">
            正在播放: {{ currentTrack?.title || '无' }}
          </p>
        </div>
      </el-card>
    </div>
  </template>
  
  <script setup>
  import { computed } from 'vue'
  import { Headset } from '@element-plus/icons-vue'
  import { usePlayerStore } from '@/stores/player'
  
  const playerStore = usePlayerStore()
  const currentTrack = computed(() => playerStore.currentTrack)
  </script>
  
  <style scoped>
  .player-view {
    max-width: 800px;
    margin: 0 auto;
  }
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  </style>