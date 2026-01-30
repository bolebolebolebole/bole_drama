<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="720px"
    :close-on-click-modal="false"
  >
    <div class="image-manager">
      <div class="upload-row">
        <el-upload
          :action="uploadAction"
          :headers="uploadHeaders"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :before-upload="beforeUpload"
          :show-file-list="false"
          accept="image/jpeg,image/png,image/jpg,image/webp"
        >
          <el-button type="primary" :icon="Upload" :loading="uploading">
            {{ uploading ? '上传中...' : '上传图片' }}
          </el-button>
        </el-upload>
        <span class="image-count">{{ images.length }} 张图片</span>
      </div>

      <div v-if="images.length > 0" class="image-list">
        <div v-for="(img, index) in images" :key="img.id" class="image-item">
          <el-image :src="img.image_url" fit="cover" />
          <div class="image-overlay">
            <div class="image-actions">
              <el-button
                :icon="ArrowUp"
                circle
                size="small"
                :disabled="index === 0"
                @click="moveImage(index, index - 1)"
              />
              <el-button
                :icon="ArrowDown"
                circle
                size="small"
                :disabled="index === images.length - 1"
                @click="moveImage(index, index + 1)"
              />
              <el-button
                :icon="Delete"
                circle
                size="small"
                type="danger"
                @click="deleteImage(img)"
              />
            </div>
          </div>
          <el-tag v-if="index === 0" size="small" type="success" class="main-tag">主图</el-tag>
        </div>
      </div>
      <el-empty v-else description="暂无图片" />
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, ArrowUp, ArrowDown, Delete } from '@element-plus/icons-vue'
import { characterLibraryAPI } from '@/api/character-library'
import { sceneLibraryAPI } from '@/api/scene-library'

interface ImageItem {
  id: number
  image_url: string
  sort_order: number
}

const props = defineProps<{
  modelValue: boolean
  type: 'character' | 'scene'
  entityId: string | number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  refresh: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const dialogTitle = computed(() => (props.type === 'character' ? '角色图片管理' : '场景图片管理'))
const images = ref<ImageItem[]>([])
const uploading = ref(false)

const uploadAction = computed(() => '/api/v1/upload/image')
const uploadHeaders = computed(() => ({}))

const loadImages = async () => {
  try {
    if (props.type === 'character') {
      const result = await characterLibraryAPI.listImages(String(props.entityId))
      images.value = result.items || []
    } else {
      const result = await sceneLibraryAPI.listImages(String(props.entityId))
      images.value = result.items || []
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载图片失败')
  }
}

watch(
  () => [props.modelValue, props.entityId, props.type],
  async ([isOpen]) => {
    if (isOpen) {
      await loadImages()
    }
  }
)

const beforeUpload = (file: File) => {
  const isImage = file.type.startsWith('image/')
  const isLt10M = file.size / 1024 / 1024 < 10
  if (!isImage) {
    ElMessage.error('只能上传图片文件')
    return false
  }
  if (!isLt10M) {
    ElMessage.error('图片大小不能超过10MB')
    return false
  }
  uploading.value = true
  return true
}

const handleUploadSuccess = async (response: any) => {
  uploading.value = false
  const imageUrl = response?.url || response?.data?.url || response?.data?.data?.url
  if (!imageUrl) {
    ElMessage.error('上传失败：未获取到图片地址')
    return
  }
  try {
    if (props.type === 'character') {
      await characterLibraryAPI.addImage(String(props.entityId), imageUrl)
    } else {
      await sceneLibraryAPI.addImage(String(props.entityId), imageUrl)
    }
    ElMessage.success('上传成功')
    await loadImages()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(error.message || '保存失败')
  }
}

const handleUploadError = () => {
  uploading.value = false
  ElMessage.error('上传失败，请重试')
}

const moveImage = async (fromIndex: number, toIndex: number) => {
  if (toIndex < 0 || toIndex >= images.value.length) return
  const imageIds = images.value.map((img) => img.id)
  const [moved] = imageIds.splice(fromIndex, 1)
  imageIds.splice(toIndex, 0, moved)
  try {
    if (props.type === 'character') {
      await characterLibraryAPI.reorderImages(String(props.entityId), imageIds)
    } else {
      await sceneLibraryAPI.reorderImages(String(props.entityId), imageIds)
    }
    await loadImages()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(error.message || '排序失败')
  }
}

const deleteImage = async (img: ImageItem) => {
  try {
    await ElMessageBox.confirm('确定要删除这张图片吗？', '删除确认', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })
    if (props.type === 'character') {
      await characterLibraryAPI.deleteImage(String(props.entityId), img.id)
    } else {
      await sceneLibraryAPI.deleteImage(String(props.entityId), img.id)
    }
    ElMessage.success('图片已删除')
    await loadImages()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}
</script>

<style scoped>
.image-manager {
  min-height: 280px;
}

.upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-primary);
}

.image-count {
  font-size: 12px;
  color: var(--text-secondary);
}

.image-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.image-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--border-primary);
}

.image-item .el-image {
  width: 100%;
  height: 100%;
}

.image-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.image-item:hover .image-overlay {
  opacity: 1;
}

.image-actions {
  display: flex;
  gap: 8px;
}

.main-tag {
  position: absolute;
  top: 8px;
  left: 8px;
}
</style>
