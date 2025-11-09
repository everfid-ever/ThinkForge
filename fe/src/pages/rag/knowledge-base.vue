<template>
  <div class="kb-container">
    <el-card class="kb-card">
      <template #header>
        <div class="card-header">
          <el-icon class="header-icon"><Folder /></el-icon>
          <span>Knowledge base management</span>
          <el-button 
            type="primary" 
            size="small" 
            plain 
            class="add-kb-btn"
            @click="showAddDialog">
            <el-icon><Plus /></el-icon> Create a new knowledge base
          </el-button>
        </div>
      </template>
      
      <!-- 知识库列表 -->
      <div class="kb-list">
        <el-table 
          v-loading="loading" 
          :data="knowledgeBaseList" 
          style="width: 100%"
          border>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="Knowledge base name" width="180" />
          <el-table-column prop="description" label="description" />
          <el-table-column prop="category" label="category" width="120" />
          <el-table-column prop="status" label="status" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.status === 2 ? 'danger': 'success'">
                {{ scope.row.status === 2 ? 'Disable': 'Enable' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="operate" width="200">
            <template #default="scope">
              <el-button 
                size="small" 
                type="primary" 
                @click="showEditDialog(scope.row)"
                plain>
                edit
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                @click="confirmDelete(scope.row)"
                plain>
                delete
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      
      <!-- 空状态 -->
      <div v-if="!loading && knowledgeBaseList.length === 0" class="empty-kb">
        <el-empty description="There is no knowledge base yet. Please click Create New in the upper right corner.">
          <template #image>
            <el-icon class="empty-icon"><Folder /></el-icon>
          </template>
        </el-empty>
      </div>
    </el-card>
    
    <!-- 新建/编辑知识库对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="isEdit ? 'Edit knowledge base' : 'Create new knowledge base'"
      width="500px">
      <el-form 
        :model="kbForm" 
        :rules="rules" 
        ref="kbFormRef" 
        label-width="100px">
        <el-form-item label="Knowledge base name" prop="name">
          <el-input v-model="kbForm.name" placeholder="Please enter the knowledge base name." />
        </el-form-item>
        <el-form-item label="description" prop="description">
          <el-input 
            v-model="kbForm.description" 
            type="textarea" 
            :rows="3" 
            placeholder="Please enter a description of the knowledge base." />
        </el-form-item>
        <el-form-item label="category" prop="category">
          <el-input v-model="kbForm.category" placeholder="Please enter the knowledge base category." />
        </el-form-item>
        <el-form-item label="status" prop="status" v-if="isEdit">
          <el-radio-group v-model="kbForm.status">
            <el-radio :label="1">Enable</el-radio>
            <el-radio :label="2">Disable</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm" :loading="submitting">
            confirm
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Folder, Plus } from '@element-plus/icons-vue'
import request from '../../utils/request.js'

// 知识库列表
const knowledgeBaseList = ref([])
// 加载状态
const loading = ref(false)
// 提交状态
const submitting = ref(false)
// 对话框可见性
const dialogVisible = ref(false)
// 是否为编辑模式
const isEdit = ref(false)
// 表单引用
const kbFormRef = ref(null)

// 表单数据
const kbForm = reactive({
  id: null,
  name: '',
  description: '',
  category: '',
  status: 1
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: 'Please enter the knowledge base name.', trigger: 'blur' },
    { min: 3, max: 20, message: 'The length is between 3 and 20 characters.', trigger: 'blur' }
  ],
  description: [
    { required: true, message: 'Please enter a description of the knowledge base.', trigger: 'blur' },
    { min: 3, max: 200, message: 'The length is between 3 and 20 characters.', trigger: 'blur' }
  ],
  category: [
    { min: 3, max: 10, message: 'The length is between 3 and 20 characters.', trigger: 'blur' }
  ]
}

// 页面加载时获取知识库列表
onMounted(() => {
  fetchKnowledgeBaseList()
})

// 获取知识库列表
const fetchKnowledgeBaseList = async () => {
  loading.value = true
  try {
    const response = await request.get('/v1/kb')
    knowledgeBaseList.value = response.data.list || []
  } catch (error) {
    console.error('Failed to retrieve knowledge base list:', error)
    ElMessage.error('Failed to retrieve knowledge base list: ' + (error.response?.data?.message || 'Unknown error'))
  } finally {
    loading.value = false
  }
}

// 显示新建对话框
const showAddDialog = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

// 显示编辑对话框
const showEditDialog = (row) => {
  isEdit.value = true
  resetForm()
  // 复制数据到表单
  Object.assign(kbForm, row)
  dialogVisible.value = true
}

// 重置表单
const resetForm = () => {
  kbForm.id = null
  kbForm.name = ''
  kbForm.description = ''
  kbForm.category = ''
  kbForm.status = 1
  
  // 重置表单验证
  if (kbFormRef.value) {
    kbFormRef.value.resetFields()
  }
}

// 提交表单
const submitForm = async () => {
  if (!kbFormRef.value) return
  
  await kbFormRef.value.validate(async (valid) => {
    if (!valid) return
    
    submitting.value = true
    try {
      if (isEdit.value) {
        // 编辑知识库
        await request.put(`/v1/kb/${kbForm.id}`, {
          name: kbForm.name,
          description: kbForm.description,
          category: kbForm.category,
          status: kbForm.status
        })
        ElMessage.success('Knowledge base updated successfully')
      } else {
        // 创建知识库
        await request.post('/v1/kb', {
          name: kbForm.name,
          description: kbForm.description,
          category: kbForm.category,
        })
        ElMessage.success('Knowledge base created successfully')
      }
      
      // 关闭对话框并刷新列表
      dialogVisible.value = false
      fetchKnowledgeBaseList()
    } catch (error) {
      console.error('Operation failed:', error)
      ElMessage.error('Operation failed: ' + (error.response?.data?.message || 'Unknown error'))
    } finally {
      submitting.value = false
    }
  })
}

// 确认删除
const confirmDelete = (row) => {
  ElMessageBox.confirm(
    `Are you sure you want to delete the knowledge base "${row.name}" ? This operation is irreversible.`,
    'Warm',
    {
      confirmButtonText: 'Confirm',
      cancelButtonText: 'Cancel',
      type: 'warning',
    }
  ).then(async () => {
    try {
      await request.delete(`/v1/kb/${row.id}`)
      ElMessage.success('Knowledge base deleted successfully')
      fetchKnowledgeBaseList()
    } catch (error) {
      console.error('Deletion failed:', error)
      ElMessage.error('Deletion failed: ' + (error.response?.data?.message || 'Unknown error'))
    }
  }).catch(() => {
    // 用户取消删除
  })
}
</script>

<style scoped>
.kb-container {
  margin: 10px;
}

.kb-card {
  margin-bottom: 20px;
}

.add-kb-btn {
  margin-left: auto;
}

.kb-list {
  margin-top: 20px;
}

.empty-kb {
  height: 300px;
}
</style>