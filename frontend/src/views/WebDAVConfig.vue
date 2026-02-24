<template>
    <el-card style="max-width: 600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="URL">
          <el-input v-model="form.url" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="save">保存</el-button>
          <el-button @click="testConn">测试连接</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </template>
  
  <script setup>
  import { ref, onMounted } from 'vue'
  import { ElMessage } from 'element-plus'
  import { api } from '@/api'
  
  const form = ref({ url: '', username: '', password: '' })
  
  onMounted(async () => {
    const res = await api.getWebDAVConfig()
    if (res.code === 0 && res.data) form.value = res.data
  })
  
  const save = async () => {
    await api.saveWebDAVConfig(form.value)
    ElMessage.success('保存成功')
  }
  
  const testConn = async () => {
    try {
      await api.testWebDAVConfig(form.value)
      ElMessage.success('连接成功')
    } catch {
      ElMessage.error('连接失败')
    }
  }
  </script>