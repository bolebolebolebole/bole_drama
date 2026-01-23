<template>
  <div class="reference-image-manager">
    <div class="header">
      <h3>{{ $t('referenceImages.title') }}</h3>
      <el-button type="primary" size="small" @click="handleAddImage">
        <el-icon><Plus /></el-icon>
        {{ $t('referenceImages.addImage') }}
      </el-button>
    </div>

    <div class="image-grid" v-if="referenceImages.length > 0">
      <div 
        v-for="(image, index) in referenceImages" 
        :key="index"
        class="image-item"
        :class="{ 'deleted': image.deleted }"
      >
        <div 
          class="image-container"
          @dblclick="image.deleted ? handleRestoreImage(index) : null"
        >
          <img 
            :src="image.url" 
            :alt="`Reference ${index + 1}`"
            class="reference-image"
          />
          
          <!-- Overlay for deleted images -->
          <div v-if="image.deleted" class="deleted-overlay">
            <span class="deleted-text">{{ $t('referenceImages.disabled') }}</span>
          </div>
          
          <!-- X button for active images (hover to show) -->
          <div v-else class="close-button" @click="handleDeleteImage(index)">
            <el-icon><Close /></el-icon>
          </div>
        </div>
        
        <div class="image-info">
          <span class="image-name">{{ $t('referenceImages.reference') }} {{ index + 1 }}</span>
          <span v-if="image.deleted" class="deleted-status">
            {{ $t('referenceImages.disabledHint') }}
          </span>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <el-empty :description="$t('referenceImages.noImages')">
        <el-button type="primary" @click="handleAddImage">
          {{ $t('referenceImages.addFirstImage') }}
        </el-button>
      </el-empty>
    </div>

    <!-- File upload dialog -->
    <el-dialog
      v-model="uploadDialogVisible"
      :title="$t('referenceImages.uploadImage')"
      width="500px"
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
          {{ $t('referenceImages.dragOrClick') }}
        </div>
        <template #tip>
          <div class="el-upload__tip">
            {{ $t('referenceImages.uploadTip') }}
          </div>
        </template>
      </el-upload>
      
      <template #footer>
        <el-button @click="uploadDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleUploadConfirm" :disabled="!selectedFile">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, type UploadInstance, type UploadRawFile } from 'element-plus'
import { Plus, Close, UploadFilled } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'

interface ReferenceImage {
  url: string
  file?: File
  deleted: boolean
}

interface Props {
  modelValue: string[]
  maxImages?: number
}

const props = withDefaults(defineProps<Props>(), {
  maxImages: 5
})

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const { t } = useI18n()

const uploadDialogVisible = ref(false)
const uploadRef = ref<UploadInstance>()
const selectedFile = ref<File | null>(null)
const referenceImages = ref<ReferenceImage[]>([])

// Initialize reference images from props
const initializeImages = () => {
  referenceImages.value = props.modelValue.map(url => ({
    url,
    deleted: false
  }))
}

// Watch for prop changes
watch(() => props.modelValue, () => {
  initializeImages()
}, { immediate: true })

// Computed property for active (non-deleted) image URLs
const activeImageUrls = computed(() => {
  return referenceImages.value
    .filter(img => !img.deleted)
    .map(img => img.url)
})

// Emit changes to parent
const emitChanges = () => {
  emit('update:modelValue', activeImageUrls.value)
}

const handleAddImage = () => {
  if (activeImageUrls.value.length >= props.maxImages) {
    ElMessage.warning(t('referenceImages.maxImagesReached', { max: props.maxImages }))
    return
  }
  uploadDialogVisible.value = true
}

const handleFileChange = (file: UploadRawFile) => {
  selectedFile.value = file
}

const beforeUpload = (file: UploadRawFile) => {
  const isImage = file.type.startsWith('image/')
  const isLt10M = file.size / 1024 / 1024 < 10

  if (!isImage) {
    ElMessage.error(t('referenceImages.onlyImages'))
    return false
  }
  if (!isLt10M) {
    ElMessage.error(t('referenceImages.fileSizeLimit'))
    return false
  }
  return false // Prevent auto upload
}

const handleUploadConfirm = () => {
  if (!selectedFile.value) return

  // Create object URL for preview
  const imageUrl = URL.createObjectURL(selectedFile.value)
  
  referenceImages.value.push({
    url: imageUrl,
    file: selectedFile.value,
    deleted: false
  })

  emitChanges()
  uploadDialogVisible.value = false
  selectedFile.value = null
  uploadRef.value?.clearFiles()
  
  ElMessage.success(t('referenceImages.imageAdded'))
}

const handleDeleteImage = (index: number) => {
  referenceImages.value[index].deleted = true
  emitChanges()
  ElMessage.success(t('referenceImages.imageDeleted'))
}

const handleRestoreImage = (index: number) => {
  referenceImages.value[index].deleted = false
  emitChanges()
  ElMessage.success(t('referenceImages.imageRestored'))
}

// Expose methods for parent component
defineExpose({
  getActiveImages: () => activeImageUrls.value,
  getAllImages: () => referenceImages.value,
  clearAll: () => {
    referenceImages.value = []
    emitChanges()
  }
})
</script>

<style scoped>
.reference-image-manager {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px;
  background: #fafafa;
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
  color: #303133;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
}

.image-item {
  position: relative;
  transition: all 0.3s ease;
}

.image-item.deleted {
  opacity: 0.5;
}

.image-container {
  position: relative;
  width: 100%;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid #e4e7ed;
  transition: border-color 0.3s ease;
}

.image-container:hover {
  border-color: #409eff;
}

.reference-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.deleted-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.deleted-text {
  color: white;
  font-size: 14px;
  font-weight: 500;
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
}

.close-button:hover {
  transform: scale(1.1);
  background: rgba(245, 108, 108, 1);
}

.image-container:hover .close-button {
  opacity: 1;
}

.image-info {
  margin-top: 8px;
  text-align: center;
}

.image-name {
  font-size: 12px;
  color: #606266;
  display: block;
}

.deleted-status {
  font-size: 11px;
  color: #f56c6c;
  font-style: italic;
}

.empty-state {
  padding: 40px 0;
}

.upload-demo {
  width: 100%;
}

.el-upload__tip {
  color: #999;
  font-size: 12px;
}
</style>