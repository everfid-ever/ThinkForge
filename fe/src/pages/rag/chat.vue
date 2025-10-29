<script setup lang="ts">
import {
  ChatDotRound,
  ChatRound,
  CopyDocument,
  Document,
  Plus,
  Position,
  Service,
  Setting,
  User,
} from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import { v4 as uuidv4 } from 'uuid'
import { nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import KnowledgeSelector from '~/components/KnowledgeSelector.vue'
import { renderMarkdown } from '~/utils/markdown.js'
import '~/styles/markdown.css'

const _router = useRouter()

interface Message {
  role: string
  content: string
  timestamp: Date
}

interface ChatSettings {
  top_k: number
  score: number
}

const sessionId = ref('')

const messages = ref<Message[]>([])

const currentMessage = ref('')

const isStreaming = ref(false)

const currentStreamingMessage = ref('')

const references = ref<any[]>([])

const knowledgeSelectorRef = ref<any>(null)

const chatSettings = ref<ChatSettings>({
  top_k: 5,
  score: 0.2,
})

const messagesContainer = ref<HTMLElement | null>(null)

const loading = ref(false)
const _inputMessage = ref('')
const showSettings = ref(false)

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

function generateSessionId() {
  sessionId.value = uuidv4()
}

async function sendMessage() {
  if (!currentMessage.value.trim() || isStreaming.value) {
    return
  }

  const message = currentMessage.value.trim()
  currentMessage.value = ''
  references.value = []

  // 添加用户消息
  messages.value.push({
    role: 'user',
    content: message,
    timestamp: new Date(),
  })

  // 添加空的AI回复消息
  messages.value.push({
    role: 'assistant',
    content: '',
    timestamp: new Date(),
  })

  isStreaming.value = true
  currentStreamingMessage.value = ''

  await nextTick()
  scrollToBottom()

  try {
    const response = await fetch('/api/v1/chat/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        question: message,
        top_k: chatSettings.value.top_k,
        score: chatSettings.value.score,
        conv_id: sessionId.value,
        knowledge_name: knowledgeSelectorRef.value?.getSelectedKnowledgeId() || '',
      }),
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = '' // 用于累积不完整的数据

    // eslint-disable-next-line no-constant-condition
    while (true) {
      const { value, done } = await reader.read()

      if (done) {
        // 处理最后剩余的数据
        if (buffer.trim()) {
          processLines([buffer])
        }
        break
      }

      // 解码数据并添加到缓冲区
      const chunk = decoder.decode(value, { stream: true })
      buffer += chunk

      // 按行分割，保留最后一个可能不完整的行
      const lines = buffer.split('\n')
      buffer = lines.pop() || '' // 保留最后一个可能不完整的行

      // 处理完整的行
      processLines(lines)
    }

    // 处理行数据的函数
    function processLines(lines: string[]) {
      for (const line of lines) {
        if (line.startsWith('data:')) {
          const data = line.slice(5).trim()
          if (data === '[DONE]') {
            // 流结束
            isStreaming.value = false
            // 确保最后一次完整渲染
            messages.value[messages.value.length - 1].content = currentStreamingMessage.value
            nextTick().then(() => scrollToBottom())
            return
          }

          try {
            const parsedData = JSON.parse(data)
            if (parsedData.content) {
              currentStreamingMessage.value += parsedData.content
              // 更新最后一条消息的内容
              messages.value[messages.value.length - 1].content = currentStreamingMessage.value
              nextTick().then(() => scrollToBottom())
            }
          }
          catch (e) {
            // eslint-disable-next-line no-console
            console.error('Failed to parse stream data: ', line, '<===>', line.slice(5).trim(), '<===>', e)
          }
        }

        if (line.startsWith('documents:')) {
          const data = line.slice(10).trim()

          try {
            const parsedData = JSON.parse(data)
            if (parsedData.document) {
              references.value.push(...parsedData.document)
              // eslint-disable-next-line no-console
              // console.log('references', references.value)
            }
          }
          catch (e) {
            // eslint-disable-next-line no-console
            console.error('Failed to parse stream data: ', line.slice(10).trim(), e)
          }
        }
      }
    }
  }
  catch (error) {
    // eslint-disable-next-line no-console
    console.error('Failed to send message: ', error)
    ElNotification({
      title: 'Error',
      message: 'Failed to send message, please try again later',
      type: 'error',
    })

    // 移除最后一条消息（AI回复）
    if (messages.value.length > 0 && messages.value[messages.value.length - 1].role === 'assistant') {
      messages.value.pop()
    }
  }
  finally {
    isStreaming.value = false
  }
}

const clearChat = () => {
  messages.value = []
  references.value = []
  generateSessionId()
}

const handleKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage({
      message: 'Copied to clipboard',
      type: 'success',
    })
  }
  catch (err) {
    ElMessage({
      message: 'Replication failed',
      type: 'error',
    })
  }
}

// 处理键盘事件
function handleKeyDown(e) {
  // 只有在按下Enter键且没有同时按下Shift键时才发送消息
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault() // 阻止默认行为
    sendMessageOld()
  }
}

// 开始新会话
function startNewSession() {
  if (messages.value.length > 0) {
    ElMessage({
      message: 'New session started',
      type: 'success',
    })
  }

  messages.value = []
  references.value = []
  sessionId.value = uuidv4()
}

// 复制会话ID
function copySessionId() {
  // 检查是否支持 Clipboard API
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(sessionId.value)
        .then(() => {
          ElMessage({
            message: 'Session ID copied to clipboard',
            type: 'success',
          })
        })
        .catch(() => {
          fallbackCopyToClipboard(sessionId.value)
        })
  } else {
    // 降级方案：使用传统的复制方法
    fallbackCopyToClipboard(sessionId.value)
  }
}

// 降级复制方案
function fallbackCopyToClipboard(text) {
  try {
    // 创建一个临时的 textarea 元素
    const textArea = document.createElement('textarea')
    textArea.value = text
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    textArea.style.top = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()

    // 尝试执行复制命令
    const successful = document.execCommand('copy')
    document.body.removeChild(textArea)

    if (successful) {
      ElMessage({
        message: 'Session ID copied to clipboard',
        type: 'success',
      })
    } else {
      throw new Error('The copy command failed to execute.')
    }
  } catch (err) {
    ElMessage({
      message: 'Copy failed, please copy the session ID manually: ' + text,
      type: 'error',
    })
  }
}

// 格式化时间
function formatTime(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

watch(
    () => messages.value.length,
    () => {
      nextTick(() => {
        scrollToBottom()
      })
    },
)

// 组件挂载后滚动到底部
onMounted(() => {
  generateSessionId()
  scrollToBottom()
})
</script>

<template>
  <div class="chat-container">
    <el-row>
      <el-col :span="16">
        <el-card class="chat-card">
          <template #header>
            <div class="card-header">
              <el-icon class="header-icon"><ChatDotRound /></el-icon>
              <span>Intelligent Q&A</span>
              <div class="header-actions">
                <KnowledgeSelector ref="knowledgeSelectorRef" class="knowledge-selector" />
                <el-button
                    type="primary"
                    size="small"
                    plain
                    class="new-session-btn"
                    @click="startNewSession"
                >
                  <el-icon><Plus /></el-icon> New Session
                </el-button>
              </div>
            </div>
          </template>

          <div ref="messagesContainer" class="chat-messages">
            <div v-if="messages.length === 0" class="empty-chat">
              <el-empty description="Start a new conversation">
                <template #image>
                  <el-icon class="empty-icon"><ChatRound /></el-icon>
                </template>
              </el-empty>
            </div>

            <div v-else class="message-list">
              <div
                  v-for="(message, index) in messages"
                  :key="index"
                  class="message-item"
                  :class="[message.role === 'user' ? 'user-message' : 'ai-message']"
              >
                <div class="message-avatar">
                  <el-avatar :icon="message.role === 'user' ? User : Service" :size="36" />
                </div>
                <div class="message-content">
                  <div v-if="message.role === 'user'" class="message-text">{{ message.content }}</div>
                  <div v-else class="message-text markdown-content" v-html="renderMarkdown(message.content)" />
                  <div class="message-time">{{ formatTime(message.timestamp) }}</div>
                </div>
              </div>
            </div>

            <div v-if="loading" class="loading-message">
              <el-skeleton :rows="1" animated />
            </div>
          </div>

          <div class="chat-input">
            <el-form @submit.prevent="sendMessage">
              <el-input
                  v-model="currentMessage"
                  type="textarea"
                  :rows="3"
                  placeholder="Please enter your question..."
                  :disabled="isStreaming"
                  @keydown="handleKeydown"
              />
              <div class="input-actions">
                <el-tooltip content="Advanced Settings" placement="top">
                  <el-button
                      type="info"
                      plain
                      circle
                      @click="showSettings = !showSettings"
                  >
                    <el-icon><Setting /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-button
                    type="primary"
                    :loading="isStreaming"
                    :disabled="!currentMessage.trim()"
                    @click="sendMessage"
                >
                  Send <el-icon class="el-icon--right"><Position /></el-icon>
                </el-button>
              </div>
            </el-form>

            <el-collapse-transition>
              <div v-show="showSettings" class="settings-panel">
                <el-form :model="chatSettings" label-position="left" label-width="180px">
                  <el-form-item label="Number of reference document results returned">
                    <el-input-number
                        v-model="chatSettings.top_k"
                        :min="1"
                        :max="10"
                        controls-position="right"
                        size="small"
                    />
                  </el-form-item>
                  <el-form-item label="Similarity threshold">
                    <el-slider
                        v-model="chatSettings.score"
                        :min="0"
                        :max="1"
                        :step="0.05"
                        :format-tooltip="(val) => val.toFixed(2)"
                        size="small"
                    />
                  </el-form-item>
                </el-form>
              </div>
            </el-collapse-transition>
          </div>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card class="references-card">
          <template #header>
            <div class="card-header">
              <el-icon class="header-icon"><Document /></el-icon>
              <span>Session Informaton</span>
            </div>
          </template>
          <div class="session-info">
            <div class="session-id">
              <span class="label">Session ID:</span>
              <el-tag size="small" type="info">{{ sessionId }}</el-tag>
              <el-tooltip content="Copy Session ID" placement="top">
                <el-button
                    type="primary"
                    link
                    size="small"
                    @click="copySessionId"
                >
                  <el-icon><CopyDocument /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <div class="message-count">
              <span class="label">Number of messages:</span>
              <span>{{ messages.length }}</span>
            </div>
          </div>

          <div class="references-content">
            <el-divider content-position="left">Reference Documentation</el-divider>

            <div v-if="references.length === 0" class="empty-references">
              <el-empty description="No reference document yet" />
            </div>

            <div v-else class="reference-list">
              <el-collapse accordion>
                <el-collapse-item
                    v-for="(ref, index) in references"
                    :key="index"
                    :title="`Document fragments #${index + 1} (Similarity: ${ref.meta_data._score.toFixed(2)})`"
                    :name="index"
                >
                  <div class="reference-content">
                    <div class="source-info">
                      <el-tag size="small">{{ ref.meta_data.ext._file_name || 'Unknown Source' }}</el-tag>
                    </div>
                    <div class="content-text markdown-content" v-html="renderMarkdown(ref.content)" />
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.chat-container {
  height: calc(100vh - 140px);
  max-height: 800px;
  min-height: 500px;
}

.chat-card, .references-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin: 10px;
}

.new-session-btn {
  margin-left: 5px;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  background-color: #f9f9f9;
  border-radius: 4px;
  margin-bottom: 15px;
  min-height: 300px;
  max-height: calc(100vh - 350px);
}

.empty-chat {
  height: 100%;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.message-item {
  display: flex;
  margin-bottom: 15px;
}

.user-message {
  flex-direction: row-reverse;
}

.message-avatar {
  margin: 0 10px;
}

.message-content {
  max-width: 70%;
  padding: 10px 15px;
  border-radius: 8px;
  padding: 12px;
  position: relative;
}

.user-message .message-content {
  background-color: #ecf5ff;
  border: 1px solid #d9ecff;
  text-align: right;
}

.ai-message .message-content {
  background-color: #fff;
  border: 1px solid #ebeef5;
  text-align: left;
}





.chat-input {
  margin-top: auto;
}

.references-content {
  flex: 1;
  overflow-y: auto;
}


/* 页面特定的Markdown样式扩展 */
.markdown-content blockquote {
  border-left: 4px solid #d0d7de;
  padding-left: 1em;
  color: #57606a;
  margin: 1em 0;
}

/* 打字效果的光标动画 */
@keyframes cursor-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* 为最后一条AI消息添加光标效果，但仅在流式传输时显示 */
.ai-message:last-child .message-text:after {
  content: '|';
  display: v-bind(isStreaming ? 'inline-block' : 'none');
  color: var(--el-color-primary);
  animation: cursor-blink 0.8s infinite;
  font-weight: bold;
  margin-left: 2px;
}
</style>