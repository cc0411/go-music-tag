import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '仪表盘', icon: 'DataLine' } },
      { path: 'music', name: 'MusicLibrary', component: () => import('../views/MusicLibrary.vue'), meta: { title: '音乐库', icon: 'Music' } },
      { path: 'player', name: 'PlayerView', component: () => import('../views/PlayerView.vue'), meta: { title: '播放器', icon: 'Headset' } },
      { path: 'webdav', name: 'WebDAVConfig', component: () => import('../views/WebDAVConfig.vue'), meta: { title: 'WebDAV', icon: 'Cloud' } },
      { path: 'scan', name: 'ScanManage', component: () => import('../views/ScanManage.vue'), meta: { title: '扫描管理', icon: 'Search' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router