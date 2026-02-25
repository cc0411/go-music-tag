<template>
  <div class="webdav-page">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span class="title">WebDAV 配置</span>
          <el-button link type="primary" :icon="Refresh" @click="loadConfig"></el-button>
        </div>
      </template>

      <div class="section-title">
        <el-icon><Cloudy /></el-icon>
        <span>WebDAV 服务器配置</span>
      </div>

      <el-form :model="form" label-width="100px" class="webdav-form">
        
        <el-form-item label="服务器地址" required>
          <el-input v-model="form.url" placeholder="http://192.168.1.100:8081" />
        </el-form-item>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="用户名">
              <el-input v-model="form.username" placeholder="admin" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="密码">
              <el-input v-model="form.password" type="password" placeholder="password" show-password />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="根路径">
          <el-input v-model="form.rootPath" placeholder="/dav" />
          <div class="form-tip" style="font-size: 12px; color: #999; margin-top: 5px;">
            提示：通常为 /dav 或 /，请根据实际服务器配置填写
          </div>
        </el-form-item>

        <el-form-item label=" ">
          <el-checkbox v-model="form.enabled">启用 WebDAV</el-checkbox>
        </el-form-item>

        <el-form-item label=" ">
          <div class="button-group">
            <el-button type="primary" @click="save" :icon="FolderChecked" :loading="saving">
              保存配置
            </el-button>
            
            <el-button @click="testConn" :icon="Connection" :loading="testing">
              测试连接
            </el-button>
            
            <el-button type="danger" plain @click="deleteConfig" :icon="Delete">
              删除配置
            </el-button>
          </div>
        </el-form-item>

      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Cloudy, Refresh, FolderChecked, Connection, Delete 
} from '@element-plus/icons-vue'
import { api } from '@/api'

const saving = ref(false)
const testing = ref(false)

const form = ref({
  url: 'http://192.168.1.3:9000',
  username: 'music',
  password: 'music',
  rootPath: '/dav', 
  enabled: false
})

const loadConfig = async () => {
  try {
    const res = await api.getWebDAVConfig()
    if (res && res.code === 0 && res.data) {
      const data = res.data
      form.value = {
        url: data.url || '',
        username: data.username || '',
        password: data.password || '', 
        rootPath: data.rootPath || data.root_path || '/dav', 
        enabled: data.enabled !== undefined ? data.enabled : true
      }
    }
  } catch (error) {
    console.error('加载配置失败', error)
  }
}

const save = async () => {
  if (!form.value.url) {
    ElMessage.warning('请填写服务器地址')
    return
  }
  
  if (!form.value.rootPath) {
    form.value.rootPath = '/dav'
  }

  saving.value = true
  try {
    await api.saveWebDAVConfig(form.value)
    ElMessage.success('保存成功')
  } catch (error) {
    ElMessage.error('保存失败：' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

const testConn = async () => {
  if (!form.value.url) {
    ElMessage.warning('请先填写服务器地址')
    return
  }
  
  const payload = { ...form.value }
  if (!payload.rootPath) {
    payload.rootPath = '/dav'
  }

  testing.value = true
  try {
    // ✅ 关键修改策略：
    // 由于后端 Test 接口可能依赖数据库中的配置 (返回 404 如果不存在)，
    // 我们在测试前先调用一次保存，确保数据库里有记录。
    
    // 1. 先保存 (静默保存，不弹窗打扰用户)
    try {
      await api.saveWebDAVConfig(payload)
    } catch (saveErr) {
      console.warn('预保存失败，尝试直接测试...', saveErr)
      // 即使保存失败，也继续尝试测试，万一后端支持直接传参呢
    }

    // 2. 再测试
    await api.testWebDAVConfig(payload)
    ElMessage.success('连接成功！服务器可用')
    
  } catch (error) {
    const errMsg = error.response?.data?.message || error.response?.data?.error || error.message
    console.error('测试失败详情:', error.response?.data)
    
    // 特殊处理 404 错误，给用户更友好的提示
    if (error.response?.status === 404 && errMsg.includes('not found')) {
      ElMessage.error('测试失败：未找到配置。请先点击“保存配置”，然后再试。')
    } else {
      ElMessage.error('连接失败：' + errMsg)
    }
  } finally {
    testing.value = false
  }
}

const deleteConfig = async () => {
  try {
    await ElMessageBox.confirm('确定要删除当前的 WebDAV 配置吗？此操作不可恢复。', '警告', { 
      confirmButtonText: '确定', 
      cancelButtonText: '取消', 
      type: 'warning' 
    })
    
    await api.deleteWebDAVConfig()
    
    form.value = { 
      url: '', 
      username: '', 
      password: '', 
      rootPath: '/dav', 
      enabled: false 
    }
    ElMessage.success('配置已删除')
  } catch (action) {
    // 用户取消
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped lang="scss">
.webdav-page {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: 100%;
}

.box-card {
  max-width: 800px;
  margin: 0 auto;
  border-radius: 8px;

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    
    .title {
      font-size: 20px;
      font-weight: 600;
      color: #1f2937;
    }
  }
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #374151;
  margin: 20px 0;
  padding-bottom: 10px;
  border-bottom: 1px solid #ebeef5;

  .el-icon {
    color: #409EFF;
    font-size: 18px;
  }
}

.webdav-form {
  :deep(.el-input__wrapper) {
    background-color: #ecf5ff; 
    box-shadow: 0 0 0 1px #d9ecff inset;
    
    &:hover, &:focus-within {
      box-shadow: 0 0 0 1px #409EFF inset;
      background-color: #fff;
    }
  }

  .button-group {
    display: flex;
    gap: 12px;
    margin-top: 10px;
  }
}
</style>