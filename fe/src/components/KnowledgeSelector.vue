<template>
  <div class="knowledge-selector">
    <el-popover
        placement="bottom"
        :width="360"
        trigger="click"
        v-model:visible="popoverVisible"
        :close-on-click-outside="false"
    >
      <template #reference>
        <el-button type="info" plain>
          <el-icon><Edit /></el-icon>
          Current knowledge base: {{ selectedKnowledgeId }}
        </el-button>
      </template>

      <div class="selector-content">
        <div class="selector-header">
          <h4>Knowledge base settings</h4>
        </div>

        <el-form label-position="top">
          <el-form-item label="Select Knowledge Base">
            <el-select
                v-model="selectedKnowledgeId"
                placeholder="Select Knowledge Base"
                filterable
                :loading="loading"
                @change="handleKnowledgeChange"
                style="width: 100%"
                :popper-append-to-body="false"
                :teleported="false"
                popper-class="knowledge-select-dropdown"
            >
              <el-option
                  v-for="item in knowledgeBaseList"
                  :key="item.name"
                  :label="item.name"
                  :value="item.name"
                  :disabled="item.status === 2"
              >
                <div class="knowledge-option">
                  <span class="knowledge-name">{{ item.name }}</span>
                  <el-tag size="small" :type="item.status === 2 ? 'danger' : 'success'">
                    {{ item.status === 2 ? 'Disabled' : 'Enabled' }}
                  </el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item v-if="selectedKnowledge" label="Knowledge Base Information">
            <div class="knowledge-info">
              <div class="info-item">
                <span class="info-label">Description:</span>
                <span class="info-value">{{ selectedKnowledge.description }}</span>
              </div>
              <div class="info-item" v-if="selectedKnowledge.category">
                <span class="info-label">Category:</span>
                <span class="info-value">{{ selectedKnowledge.category }}</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item class="form-actions">
            <el-button type="primary" @click="saveKnowledgeSelection">
              Confirm
            </el-button>
            <el-button @click="popoverVisible = false">
              Cancel
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-popover>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { Edit } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '../utils/request'

// 定义事件
const emit = defineEmits(['knowledge-changed', 'change'])

// 组件状态
const popoverVisible = ref(false)
const knowledgeBaseList = ref([])
const selectedKnowledgeId = ref('')
const loading = ref(false)

// 计算属性：获取当前选中的知识库详细信息
const selectedKnowledge = computed(() => {
  return knowledgeBaseList.value.find(item => item.name === selectedKnowledgeId.value) || null
})

// 本地存储键名
const STORAGE_KEY = 'go_rag_selected_knowledge_id'

// 获取知识库列表
function fetchKnowledgeBaseList() {
  loading.value = true
  return new Promise((resolve) => {
    request.get('/v1/kb')
        .then((response) => {
          knowledgeBaseList.value = response.data?.list || []
          // 如果当前选中的知识库不在列表中，清空选择
          if (selectedKnowledgeId.value) {
            const selected = knowledgeBaseList.value.find(
                item => item.name === selectedKnowledgeId.value && item.status !== 2,
            )
            if (!selected) {
              selectedKnowledgeId.value = ''
              localStorage.removeItem(STORAGE_KEY)
            }
          }
          // 如果没有选中的知识库，且列表中有可用的知识库，则自动选择第一个
          if (!selectedKnowledgeId.value && knowledgeBaseList.value.length > 0) {
            const firstAvailable = knowledgeBaseList.value.find(item => item.status !== 2)
            if (firstAvailable) {
              selectedKnowledgeId.value = firstAvailable.name
              // 自动选择后触发change事件
              emit('change', selectedKnowledgeId.value)
            }
          }
          resolve()
        })
        .catch((error) => {
          ElMessage.error('Failed to retrieve knowledge base list')
          resolve()
        })
        .finally(() => {
          loading.value = false
        })
  })
}

// 处理知识库选择变化
function handleKnowledgeChange(value) {
  // 知识库选择变化处理
}

// 保存知识库选择
function saveKnowledgeSelection() {
  if (!selectedKnowledgeId.value) {
    ElMessage.warning('Please select a knowledge base')
    return
  }

  // 保存到本地存储
  localStorage.setItem(STORAGE_KEY, selectedKnowledgeId.value)

  ElMessage.success(`Knowledge base selected: ${selectedKnowledgeId.value}`)
  popoverVisible.value = false

  // 触发自定义事件，通知父组件
  emit('knowledge-changed', selectedKnowledgeId.value)
}

// 初始化：获取知识库列表和已保存的选择
onMounted(async () => {
  // 从本地存储获取已保存的知识库ID
  const savedKnowledgeId = localStorage.getItem(STORAGE_KEY)

  if (savedKnowledgeId) {
    selectedKnowledgeId.value = savedKnowledgeId
  }

  // 获取知识库列表
  await fetchKnowledgeBaseList()
})

// 暴露方法给父组件
defineExpose({
  fetchKnowledgeBaseList,
  getSelectedKnowledgeId: () => selectedKnowledgeId.value,
})
</script>

<style scoped>
.knowledge-selector {
  display: inline-block;
}

.selector-content {
  padding: 4px;
}

.selector-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  margin-bottom: 16px;
}

.selector-header h4 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.selector-content :deep(.el-form) {
  padding: 0 16px 12px;
}

.selector-content :deep(.el-form-item) {
  margin-bottom: 20px;
}

.selector-content :deep(.el-form-item__label) {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-regular);
  margin-bottom: 8px;
  padding: 0;
}

.selector-content :deep(.el-select) {
  width: 100%;
}

.knowledge-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  gap: 12px;
}

.knowledge-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.knowledge-info {
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 12px;
}

.info-item {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
  line-height: 1.6;
}

.info-item:last-child {
  margin-bottom: 0;
}

.info-label {
  font-weight: 600;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

.info-value {
  color: var(--el-text-color-secondary);
  flex: 1;
  word-break: break-word;
}

.form-actions {
  margin-bottom: 0 !important;
  padding-top: 8px;
}

.form-actions :deep(.el-form-item__content) {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

:deep(.knowledge-select-dropdown) {
  z-index: 9999 !important;
}
</style>