<template>
  <div class="reference-image-manager-pro">
    <div class="header">
      <h3>参考图</h3>
      <el-button type="primary" size="small" @click="showUploadDialog = true">
        <el-icon><Plus /></el-icon>
        添加参考图
      </el-button>
    </div>

    <div class="image-grid" v-if="referenceImages.length > 0">
      <div 
        v-for="(image, index) in referenceImages" 
        :key="image.key"
        class="image-item"
        :class="{ 'disabled': image.disabled, 'custom': image.isCustom }"
        @dblclick="image.disabled ? toggleDisable(image.key, false) : null"
      >
        <div class="image-container">
          <img 
            :src="image.url" 
            :alt="image.name"
            class="reference-image"
          />
          
          <!-- Disabled overlay -->
          <div v-if="image.disabled" class="disabled-overlay">
            <span class="disabled-text">已禁用</span>
          </div>
          
          <!-- Close button for active images (hover to show) -->
          <div v-else class="close-button" @click="toggleDisable(image.key, true)">
            <el-icon><Close /></el-icon>
          </div>
        </div>
        
        <div class="image-info">
          <span class="image-name">{{ image.name }}</span>
          <span v-if="image.disabled" class="deleted-status">
            双击恢复
          </span>
        </div>

        <!-- Delete button for custom images -->
        <el-button 
          v-if="image.isCustom" 
          size="small" 
          text 
          type="danger"
          class="delete-btn"
          @click="removeCustomImage(image.key)"
        >
          删除
        </el-button>
      </div>
    </div>

    <div v-else class="empty-state">
      <el-empty description="暂无参考图">
        <el-button type="primary" @click="showUploadDialog = true">
          添加第一张参考图
        </el-button>
      </el-empty>
    </div>

    <!-- Upload dialog -->
    <el-dialog
      v-model="showUploadDialog"
      title="添加参考图"
      width="500px"
      :close-on-click-modal="false"
      @close="resetUploadState"
    >
      <el-upload
        ref="uploadRef"
        class="upload-demo"
        drag
        :auto-upload="false"
        :on-change="handleFileChange"
        :before-upload="beforeUpload"
        accept="image/*"
        :limit="1"
      >
        <el-icon class="el-icon--upload"><upload-filled /></el-icon>
        <div class="el-upload__text">
          将文件拖到此处，或点击上传
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持 jpg/png 格式，文件大小不超过 10MB
          </div>
        </template>
      </el-upload>
      
      <template #footer>
        <el-button @click="showUploadDialog = false">取消</el-button>
        <el-button type="primary" @click="handleUploadConfirm" :disabled="!selectedFile">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, type UploadInstance, type UploadRawFile } from 'element-plus'
import { Plus, Close, UploadFilled } from '@element-plus/icons-vue'

interface ReferenceImage {
  key: string
  name: string
  url: string
  disabled: boolean
  isCustom: boolean
}

interface Props {
  modelValue: ReferenceImage[]
  systemReferenceImages?: Array<{key: string, name: string, url: string}>
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: ReferenceImage[]]
}>()

const showUploadDialog = ref(false)
const uploadRef = ref<UploadInstance>()
const selectedFile = ref<File | null>(null)

const referenceImages = computed<ReferenceImage[]>({
  get: () => {
    const images: ReferenceImage[] = []
    
    // Add system reference images (scene + character)
    if (props.systemReferenceImages) {
      props.systemReferenceImages.forEach((sysImg, index) => {
        const existing = props.modelValue.find(img => img.key === sysImg.key)
        images.push({
          key: sysImg.key,
          name: sysImg.name,
          url: sysImg.url,
          disabled: existing?.disabled || false,
          isCustom: false
        })
      })
    }
    
    // Add custom reference images
    props.modelValue.forEach(img => {
      if (!img.isCustom) return
      
      const existing = images.find(i => i.key === img.key)
      if (existing) {
        existing.disabled = img.disabled
      } else {
        images.push({
          key: img.key,
          name: img.name,
          url: img.url,
          disabled: img.disabled,
          isCustom: true
        })
      }
    })
    
    return images
  },
  set: (value: ReferenceImage[]) => {
    // Filter out system images from the value, keep only custom images
    const customImages = value.filter(img => img.isCustom)
    emit('update:modelValue', customImages)
  }
})

const toggleDisable = (key: string, disabled: boolean) => {
  const images = [...referenceImages.value]
  const image = images.find(img => img.key === key)
  if (image) {
    image.disabled = disabled
    if (disabled) {
      ElMessage.success('参考图已禁用')
    } else {
      ElMessage.success('参考图已恢复')
    }
  }
}

const removeCustomImage = (key: string) => {
  const images = referenceImages.value.filter(img => img.key !== key)
  ElMessage.success('参考图已删除')
}

const handleFileChange = (file: any) => {
  selectedFile.value = file.raw
}

const beforeUpload = (file: File): boolean => {
  const allowedTypes = new Set(['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'])
  if (!allowedTypes.has(file.type)) {
    ElMessage.error('只支持图片格式 (jpg, png, gif, webp)')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    ElMessage.error('文件大小不能超过 10MB')
    return false
  }
  return true
}

const handleUploadConfirm = async () => {
  if (!selectedFile.value) {
    ElMessage.warning('请选择文件')
    return
  }

  if (!beforeUpload(selectedFile.value)) {
    return
  }

  const formData = new FormData()
  formData.append('file', selectedFile.value)

  try {
    const response = await fetch('/api/v1/upload/image', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: formData
    })

    if (!response.ok) {
      throw new Error('上传失败')
    }

    const data = await response.json()
    const url = data.url || data.data?.url

    if (!url) {
      throw new Error('未获取到图片地址')
    }

    // Add new custom image
    const key = `custom:${Date.now()}`
    const newImages = [...props.modelValue, {
      key: key,
      name: `自定义${props.modelValue.filter(img => img.isCustom).length + 1}`,
      url: url,
      disabled: false,
      isCustom: true
    }]

    emit('update:modelValue', newImages)
    ElMessage.success('参考图已添加')
    showUploadDialog.value = false
    resetUploadState()
  } catch (error: any) {
    ElMessage.error(error.message || '上传失败')
  }
}

const resetUploadState = () => {
  selectedFile.value = null
  uploadRef.value?.clearFiles()
}
</script>

<style scoped>
.reference-image-manager-pro {
  width: 100%;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.image-item {
  border: 1px solid var(--border-primary);
  border-radius: 8px;
  padding: 10px;
  background: var(--bg-secondary);
  position: relative;
  transition: all 0.2s;
}

.image-item:hover {
  border-color: #409eff;
}

.image-item.disabled {
  opacity: 0.5;
  background: var(--bg-secondary);
}

.image-item.disabled .reference-image {
  filter: grayscale(100%);
}

.image-item.custom {
  border-color: #e6a23c;
}

.image-container {
  position: relative;
  width: 100%;
  margin-bottom: 8px;
}

.reference-image {
  width: 100%;
  height: 90px;
  object-fit: cover;
  border-radius: 6px;
}

.close-button {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 24px;
  height: 24px;
  background: rgba(245, 108, 108, 0.9);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.3s ease, transform 0.2s ease;
  color: white;
  font-size: 14px;
  z-index: 10;
}

.close-button:hover {
  transform: scale(1.1);
  background: rgba(245, 108, 108, 1);
}

.image-item:hover .close-button {
  opacity: 1;
}

.disabled-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  z-index: 5;
}

.disabled-text {
  color: white;
  font-size: 14px;
  font-weight: 500;
}

.image-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.image-name {
  font-size: 12px;
  color: var(--text-secondary);
  word-break: break-word;
}

.deleted-status {
  font-size: 11px;
  color: #f56c6c;
}

.delete-btn {
  position: absolute;
  top: 4px;
  left: 4px;
  padding: 4px 8px;
  z-index: 15;
}

.empty-state {
  padding: 40px;
  text-align: center;
}

.upload-demo {
  width: 100%;
}

.el-upload__text {
  margin-top: 12px;
  color: var(--text-secondary);
}

.el-upload__tip {
  margin-top: 8px;
  font-size: 12px;
  color: #999;
}
</style>
