<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled, Upload } from '@element-plus/icons-vue'
import KnowledgeSelector from '../../components/KnowledgeSelector.vue'

const processingInfo = ref(null)
const indexResult = ref(null)
const knowledgeSelectorRef = ref(null)

function beforeUpload(file) {
  // 检查文件类型
  // const allowedTypes = ['application/pdf', 'text/markdown', 'text/html', 'text/plain']
  const allowedTypes = ['text/markdown', 'text/html', 'text/plain']
  const isAllowed = allowedTypes.includes(file.type)

  if (!isAllowed) {
    ElMessage.error('Only supports Markdown, HTML and text files!')
    return false
  }

  // 显示处理中信息
  processingInfo.value = {
    title: 'Document Processing',
    type: 'info',
    description: `Processing file: ${file.name}, please wait...`,
  }

  return true
}

function handleUploadSuccess(response) {

  processingInfo.value = {
    title: 'Document Processing Completed',
    type: 'success',
    description: 'The document has been successfully indexed into the system'
  }
  // 显示索引结果
  indexResult.value = {
    chunks: response.data?.doc_ids?.length || 0,
    status: 'success'
  }

  ElMessage.success('Document indexing successful!')
}

function handleUploadError(error) {
  processingInfo.value = {
    title: 'Document Processing Failed',
    type: 'error',
    description: 'An error occurred during document indexing, please try again',
  }

  indexResult.value = {
    chunks: 0,
    status: 'error',
  }

  ElMessage.error('Document indexing failed!')
  console.error('Upload error:', error)
}

function getUpdateData() {
  const selectedKnowledgeId = knowledgeSelectorRef.value?.getSelectedKnowledgeId()
  return {
    knowledge_name: selectedKnowledgeId || 'default',
  }
}

onMounted(() => {
  // 组件挂载后的初始化逻辑
})
</script>

<template>
  <div class="indexer-container">
    <el-card class="indexer-card">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon class="header-icon"><Upload /></el-icon>
            <span>Document Index</span>
          </div>
          <div class="header-actions">
            <KnowledgeSelector ref="knowledgeSelectorRef" class="knowledge-selector" />
          </div>
        </div>
      </template>
      <div class="upload-area">
        <el-upload
            class="upload-component"
            drag
            action="/api/v1/indexer"
            :on-success="handleUploadSuccess"
            :on-error="handleUploadError"
            :before-upload="beforeUpload"
            :data="getUpdateData"
            :show-file-list="true"
            multiple>
          <el-icon class="el-icon--upload"><Upload /></el-icon>
          <div class="el-upload__text">
            Drag and Drop Files Here or <em>Click to Upload</em>
          </div>
          <template #tip>
            <div class="el-upload__tip">
              Support Uploading PDF, Markdown, HTML and Other Document Files
            </div>
          </template>
        </el-upload>
      </div>

      <div class="process-info" v-if="processingInfo">
        <el-alert
            :title="processingInfo.title"
            :type="processingInfo.type"
            :description="processingInfo.description"
            show-icon
            :closable="false">
        </el-alert>
      </div>
    </el-card>

    <el-card class="indexer-info-card" v-if="indexResult">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon class="header-icon"><InfoFilled /></el-icon>
            <span>Index Results</span>
          </div>
        </div>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="Number of Document Fragments">{{ indexResult.chunks }}</el-descriptions-item>
        <el-descriptions-item label="Index Status">
          <el-tag :type="indexResult.status === 'success' ? 'success' : 'danger'">
            {{ indexResult.status === 'success' ? 'Success' : 'Failed' }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<style scoped>
.indexer-container {
  margin: 20px;
}

.indexer-card {
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 18px;
}

.header-actions {
  display: flex;
  align-items: center;
}

.knowledge-selector {
  min-width: 200px;
}

.upload-area {
  padding: 20px 0;
}

.upload-component {
  width: 100%;
}

.process-info {
  margin-top: 24px;
}

.indexer-info-card {
  margin-top: 24px;
}
</style>