<script setup>
import { Delete, Edit, CopyDocument } from '@element-plus/icons-vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import '~/styles/markdown.css';
import { formatDate } from '~/utils/format.js';
import { renderMarkdown } from '~/utils/markdown.js';
import request from '~/utils/request.js';

const route = useRoute();
const router = useRouter();

const documentId = ref(route.params.documentId);
const documentInfo = ref(null);

const chunksList = ref([]);
const chunksLoading = ref(false);
const chunksCurrentPage = ref(1);
const chunksPageSize = ref(6);
const chunksTotal = ref(0);

const editingChunkId = ref(null);
const editedContent = ref('');
const isSaving = ref(false);

const pageTitle = computed(() => {
  if (documentInfo.value) {
    return `Details of document "${documentInfo.value.fileName}" block division`;
  }
  return 'Document chunk details';
});

function goBack() {
  router.back();
}

// 获取分块列表
async function fetchChunksList() {
  if (!documentId.value) return;
  chunksLoading.value = true;
  try {
    const response = await request.get('/v1/chunks', {
      params: {
        knowledge_doc_id: documentId.value,
        page: chunksCurrentPage.value,
        size: chunksPageSize.value,
      },
    });
    chunksList.value = response.data.data || [];
    chunksTotal.value = response.data.total || 0;
  } catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to retrieve the list of blocks:', error);
  } finally {
    chunksLoading.value = false;
  }
}

function handleChunksPageChange(page) {
  chunksCurrentPage.value = page;
  fetchChunksList();
}

async function copyChunkContent(content) {
  if (!content) {
    ElMessage.warning('No content can be copied.');
    return;
  }
  try {
    await navigator.clipboard.writeText(content);
    ElMessage.success('Content has been copied to clipboard');
  } catch (error) {
    ElMessage.error('Copying failed, please copy manually.');
  }
}

function handleEdit(chunk) {
  editingChunkId.value = chunk.id;
  editedContent.value = chunk.content;
}

function handleCancelEdit() {
  editingChunkId.value = null;
  editedContent.value = '';
}

async function handleSaveEdit(chunk) {
  isSaving.value = true;
  try {
    await request.put('/v1/chunks_content', {
      id: chunk.id,
      content: editedContent.value,
    });
    // 更新前端数据
    const chunkToUpdate = chunksList.value.find((c) => c.id === chunk.id);
    if (chunkToUpdate) {
      chunkToUpdate.content = editedContent.value;
    }
    ElMessage.success('Content updated successfully!');
    handleCancelEdit();
  } catch (error) {
    ElMessage.error('Update failed, please try again.');
    // eslint-disable-next-line no-console
    console.error('Failed to update chunk content:', error);
  } finally {
    isSaving.value = false;
  }
}

async function handleDeleteChunk(chunk) {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to delete the content with chunk ID ${chunk.chunkId}? This operation is irreversible.`,
      'Confirm deletion',
      {
        confirmButtonText: 'Confirm deletion',
        cancelButtonText: 'Cancel',
        type: 'warning',
      },
    );
    await request.delete('/v1/chunks', { params: { id: chunk.id } });
    ElMessage.success('Block deletion successful!');
    fetchChunksList();
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Block deletion failed, please try again.');
      // eslint-disable-next-line no-console
      console.error('Block deletion failed:', error);
    }
  }
}

onMounted(() => {
  const docFromStorage = localStorage.getItem(`document-${documentId.value}`);
  // eslint-disable-next-line no-console
  console.log('docFromStorage', docFromStorage);
  // eslint-disable-next-line no-console
  console.log('route.params', route.params);
  if (docFromStorage) {
    try {
      const docData = JSON.parse(docFromStorage);
      if (docData && docData.id == documentId.value) {
        documentInfo.value = docData;
      } else {
        ElMessage.warning('The document information does not match. Please go to the document list page.');
        router.push('/knowledge-documents');
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to parse document data:', error);
      ElMessage.warning('The document information is formatted incorrectly. Please access it from the document list page.');
      router.push('/knowledge-documents');
    }
  } else {
    ElMessage.warning('The document information is incomplete. Please access it from the document list page.');
    router.push('/knowledge-documents');
  }
  fetchChunksList();
});
</script>

<template>
  <div class="chunk-details-container">
    <el-page-header @back="goBack" class="page-header">
      <template #content>
        <span class="text-large font-600 mr-3"> {{ pageTitle }} </span>
      </template>
    </el-page-header>
    <div v-if="chunksLoading" class="loading-container">
      <el-skeleton :rows="5" animated />
    </div>
    <div v-else-if="chunksList.length > 0">
      <el-card v-for="chunk in chunksList" :key="chunk.id" class="chunk-item-card">
        <template #header>
          <div class="chunk-card-header">
            <span>Chunk ID: {{ chunk.chunkId }}</span>
            <el-space>
              <el-button
                text
                size="small"
                :icon="CopyDocument"
                @click="copyChunkContent(chunk.content)">
                Copy
              </el-button>
              <el-button 
                text
                size="small"
                :icon="Edit"
                @click="handleEdit(chunk)">
                Edit
              </el-button>
              <el-button
                text
                size="small"
                type="danger"
                :icon="Delete"
                @click="handleDeleteChunk(chunk)"
              >
                Delete
              </el-button>
            </el-space>
          </div>
        </template>
        <el-input
          v-if="editingChunkId === chunk.id"
          v-model="editedContent"
          type="textarea"
          :rows="8"
          class="chunk-content-textarea"
        />
        <el-scrollbar v-else class="chunk-content-scrollbar">
          <div class="markdown-content chunk-content-pre" v-html="renderMarkdown(chunk.content)"></div>
        </el-scrollbar>

        <div class="chunk-card-footer">
          <div v-if="editingChunkId === chunk.id" class="edit-actions">
            <el-button @click="handleCancelEdit">Cancel</el-button>
            <el-button type="primary" @click="handleSaveEdit(chunk)" :loading="isSaving">Save</el-button>
          </div>
          <span v-else>Created in: {{ formatDate(chunk.createdAt) }}</span>
        </div>
      </el-card>
      <div class="pagination-container" v-if="chunksTotal > chunksPageSize">
        <el-pagination
          :current-page="chunksCurrentPage"
          :page-size="chunksPageSize"
          :total="chunksTotal"
          layout="total, prev, pager, next"
          @current-change="handleChunksPageChange" />
      </div>
    </div>
    <el-empty v-else description="This document does not currently contain block data."></el-empty>

  </div>
</template>

<style scoped>
.chunk-details-container {
   margin: 10px;
}

.chunk-item-card {
  box-shadow: var(--el-box-shadow-light);
  margin-bottom: 20px;
}

.chunk-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 14px;
  color: #606266;
}

.chunk-content-scrollbar {
  height: 200px;
  text-align: left;
}

.chunk-content-pre {
  white-space: normal;
  word-wrap: break-word;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  background-color: #f8f9fa;
  padding: 10px;
  border-radius: 4px;
  color: #495057;
  margin: 0;
}

.chunk-content-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.chunk-card-footer {
  margin-top: 15px;
  text-align: right;
  font-size: 12px;
  color: #909399;
}
</style>