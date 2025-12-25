<script setup>
import { onMounted, ref } from 'vue'
import { Document, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate, getStatusText, getStatusType } from '~/utils/format.js'
import request from '~/utils/request.js'
import KnowledgeSelector from '../../components/KnowledgeSelector.vue'

const knowledgeSelectorRef = ref(null)
const documentsList = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)

async function onKnowledgeChange() {
  currentPage.value = 1
  await fetchDocumentsList()
}

function fetchDocumentsList() {
  if (knowledgeSelectorRef.value?.getSelectedKnowledgeId() === '') {
    documentsList.value = []
    total.value = 0
    return
  }
  loading.value = true

  request.get('/v1/documents', {
    params: {
      knowledge_name: knowledgeSelectorRef.value?.getSelectedKnowledgeId(),
      page: currentPage.value,
      size: pageSize.value,
    },
  })
      .then((response) => {
        documentsList.value = response.data.data || []
        total.value = response.data.total || 0
      })
      .catch((error) => {
        const errorMessage = error.response?.data?.message || 'Unknown error'
        ElMessage.error(`Failed to retrieve document list: ${errorMessage}`)
      })
      .finally(() => {
        loading.value = false
      })
}

function confirmDelete(document) {
  ElMessageBox.confirm(
      `Are you sure you want to delete the document "${document.fileName}"? This operation will delete all data blocks under this document, and the deletion is irreversible.`,
      'Confirm Deletion',
      {
        confirmButtonText: 'Confirm Deletion',
        cancelButtonText: 'Cancel',
        type: 'warning',
      },
  )
      .then(async () => {
        try {
          await request.delete('/v1/documents', { params: { document_id: document.id } })
          ElMessage.success(`Document "${document.fileName}" deleted successfully`)
          await fetchDocumentsList()
        }
        catch (error) {
          if (error !== 'cancel') {
            // 错误消息已由 request 拦截器统一处理
          }
        }
      })
      .catch(() => {
        // 用户取消删除
      })
}

async function handleSizeChange(size) {
  pageSize.value = size
  currentPage.value = 1
  await fetchDocumentsList()
}

async function handleCurrentChange(page) {
  currentPage.value = page
  await fetchDocumentsList()
}

function setDocument(row) {
  localStorage.setItem(`document-${row.id}`, JSON.stringify(row))
}

onMounted(async () => {
  await knowledgeSelectorRef.value.fetchKnowledgeBaseList?.()
  // 如果已经有选中的知识库,直接加载文档列表
  if (knowledgeSelectorRef.value.getSelectedKnowledgeId()) {
    await onKnowledgeChange()
  }
})
</script>

<template>
  <div class="knowledge-documents">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon class="header-icon"><Search /></el-icon>
            <span>Knowledge Document Management</span>
          </div>
          <div class="header-actions">
            <KnowledgeSelector @change="onKnowledgeChange" ref="knowledgeSelectorRef" />
          </div>
        </div>
      </template>
      <el-table
          v-loading="loading"
          :data="documentsList"
          style="width: 100%; margin-top: 20px;"
          empty-text="Please Select the Knowledge Base First"
      >
        <el-table-column prop="id" label="ID" width="80" />

        <el-table-column prop="fileName" label="File Name" min-width="220">
          <template #default="scope">
            <div class="file-info">
              <el-icon class="file-icon">
                <Document />
              </el-icon>
              <span class="file-name">{{ scope.row.fileName }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="Status" width="120">
          <template #default="scope">
            <el-tag :type="getStatusType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="updatedAt" label="Update Time" width="200">
          <template #default="scope">
            {{ formatDate(scope.row.updatedAt) }}
          </template>
        </el-table-column>

        <el-table-column label="Operations" width="220">
          <template #default="scope">
            <router-link :to="`/chunk-details/${scope.row.id}`">
              <el-button
                  type="primary"
                  size="small"
                  style="margin-right: 10px;"
                  @click="setDocument(scope.row)"
              >
                Check Details
              </el-button>
            </router-link>
            <el-button
                type="danger"
                size="small"
                @click="confirmDelete(scope.row)"
            >
              Delete
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="total > 0" class="pagination">
        <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.knowledge-documents {
  margin: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
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

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-icon {
  font-size: 16px;
  color: #409eff;
}

.file-name {
  word-break: break-all;
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}
</style>