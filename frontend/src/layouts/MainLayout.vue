<template>
  <div class="app-container">
    <!-- 侧边栏 -->
    <el-aside width="220px" class="sidebar">
      <div class="logo">🎵 音乐标签</div>
      <el-menu :default-active="$route.path" router background-color="#304156" text-color="#bfcbd9" active-text-color="#409EFF">
        <el-menu-item v-for="route in $router.options.routes[0].children" :key="route.path" :index="route.path">
          <el-icon><component :is="route.meta.icon" /></el-icon>
          <span>{{ route.meta.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 主内容区 -->
    <el-main class="main-content">
      <router-view />
    </el-main>

    <!-- 底部播放器-->
    <AudioPlayer />
  </div>
</template>

<script setup>
import AudioPlayer from '../components/AudioPlayer.vue'
</script>

<style scoped lang="scss">
.app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background-color: #f0f2f5;
}

.sidebar {
  background-color: #304156;
  color: white;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  height: 100%;
  z-index: 10;

  .logo {
    height: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: bold;
    background-color: #2b3a4b;
    flex-shrink: 0;
  }

  :deep(.el-menu) {
    border-right: none;
    overflow-y: auto;
    flex: 1;
  }
}

.main-content {
  flex: 1;
  padding: 20px;
  background-color: #f0f2f5;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  
  /* 留出播放器高度 */
  padding-bottom: 110px; 
}

/* 滚动条美化 */
.main-content::-webkit-scrollbar { width: 6px; }
.main-content::-webkit-scrollbar-thumb { background: #ccc; border-radius: 3px; }
.main-content::-webkit-scrollbar-track { background: transparent; }
</style>