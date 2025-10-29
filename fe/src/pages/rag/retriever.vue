<template>
  <div class="retriever-container">
    <el-card class="retriever-card">
      <template #header>
        <div class="card-header">
          <el-icon class="header-icon"><Search /></el-icon>
          <span>Document Retrieval</span>
          <div class="header-actions">
            <KnowledgeSelector ref="knowledgeSelectorRef" />
          </div>
        </div>
      </template>

      <div class="search-area">
        <el-form :model="searchForm" label-position="top">
          <el-form-item label="">
            <el-input
                v-model="searchForm.question"
                placeholder="Please enter the question you want to search"
                clearable
                @keyup.enter="handleSearch">
              <template #append>
                <el-button :icon="Search" @click="handleSearch">Retrieve</el-button>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="Number of returned results">
                  <el-input-number
                      v-model="searchForm.top_k"
                      :min="1"
                      :max="10"
                      controls-position="right"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Similarity threshold">
                  <el-slider
                      v-model="searchForm.score"
                      :min="0"
                      :max="1"
                      :step="0.05"
                      :format-tooltip="(val) => val.toFixed(2)"
                  />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form-item>
        </el-form>
      </div>

      <div class="loading-area" v-if="loading">
        <el-skeleton :rows="5" animated />
      </div>

      <div class="result-area" v-if="!loading && searchResults.length > 0">
        <div class="result-header">
          <el-divider content-position="left">
            <el-icon><Document /></el-icon>
            Search Results
          </el-divider>
        </div>

        <el-collapse v-model="activeNames">
          <el-collapse-item
              v-for="(result, index) in searchResults"
              :key="index"
              :title="`Document fragments #${index + 1} (Similarity: ${result.meta_data._score.toFixed(2)})`"
              :name="index">
            <div class="result-content">
              <el-card shadow="never" class="content-card">
                <div class="source-info">
                  <el-tag size="small">{{ result.meta_data.ext._file_name || 'Unknown Source' }}</el-tag>
                </div>
                <div class="content-text markdown-content" v-html="renderMarkdown(result.content)"></div>
              </el-card>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>

      <div class="empty-result" v-if="!loading && searchResults.length === 0 && searched">
        <el-empty description="No relevant documents found" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { Search, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import hljs from 'highlight.js'
import 'highlight.js/styles/github.css'
import KnowledgeSelector from '../../components/KnowledgeSelector.vue'
import request from "../../utils/request.js";

// 配置Marked和代码高亮
marked.setOptions({
  highlight: function(code, lang) {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value;
    }
    return hljs.highlightAuto(code).value;
  },
  breaks: true
});

// Markdown渲染函数
const renderMarkdown = (content) => {
  if (!content) return '';
  try {
    const html = marked(content);
    return DOMPurify.sanitize(html);
  } catch (error) {
    console.error('Markdown rendering errors:', error);
    return content;
  }
};

const searchForm = reactive({
  question: '',
  top_k: 5,
  score: 0.2
})

const loading = ref(false)
const searchResults = ref([])
const activeNames = ref([0]) // 默认展开第一个结果
const searched = ref(false)
const knowledgeSelectorRef = ref(null)

const handleSearch = async () => {
  if (!searchForm.question) {
    ElMessage.warning('Please enter your search question')
    return
  }

  loading.value = true
  searched.value = true

  try {
    const response = await request.post('/v1/retriever', {
      question: searchForm.question,
      top_k: searchForm.top_k,
      score: searchForm.score,
      knowledge_name: knowledgeSelectorRef.value?.getSelectedKnowledgeId() || ''
    })
    searchResults.value = response.data.document || []

    if (searchResults.value.length === 0) {
      ElMessage.info('No relevant documents found')
    }
  } catch (error) {
    console.error('Retrieval failed:', error)
    ElMessage.error('Retrieval failed: ' + (error.response?.data?.message || 'Unknow Error'))
    searchResults.value = []
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.retriever-container {
  margin: 10px;
}

.retriever-card {
  margin-bottom: 20px;
}

.content-text {
  line-height: 1.6;
}

/* 页面特定样式 - Markdown样式已移至公共样式文件 */
</style>