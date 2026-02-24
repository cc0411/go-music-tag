<template>
    <div class="app-container">
      <el-aside width="220px" class="sidebar">
        <div class="logo">🎵 音乐标签</div>
        <el-menu :default-active="$route.path" router background-color="#304156" text-color="#bfcbd9" active-text-color="#409EFF">
          <el-menu-item v-for="route in $router.options.routes[0].children" :key="route.path" :index="route.path">
            <el-icon><component :is="route.meta.icon" /></el-icon>
            <span>{{ route.meta.title }}</span>
          </el-menu-item>
        </el-menu>
      </el-aside>
  
      <el-main class="main-content">
        <router-view />
      </el-main>
  
      <AudioPlayer class="global-player" />
    </div>
  </template>
  
  <script setup>
  import AudioPlayer from '../components/AudioPlayer.vue'
  </script>
  
  <style scoped>
  .app-container { display: flex; height: 100vh; overflow: hidden; }
  .sidebar { background-color: #304156; color: white; }
  .logo { height: 60px; display: flex; align-items: center; justify-content: center; font-size: 20px; font-weight: bold; background-color: #2b3a4b; }
  .main-content { flex: 1; overflow-y: auto; padding-bottom: 100px; background-color: #f0f2f5; }
  .global-player { position: fixed; bottom: 0; left: 220px; right: 0; height: 90px; background: white; box-shadow: 0 -2px 10px rgba(0,0,0,0.1); z-index: 100; }
  </style>