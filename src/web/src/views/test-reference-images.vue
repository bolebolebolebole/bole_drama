<template>
  <div class="test-page">
    <h1>参考图管理测试页面</h1>

    <div class="section">
      <h2>系统参考图（场景背景 + 角色）</h2>
      <div class="info">
        这些参考图会自动从当前镜头加载。
      </div>

      <ReferenceImageManagerPro
        v-model="customReferenceImages"
        :system-reference-images="systemReferenceImages"
        @add="handleAddCustomReference"
      />
    </div>

    <div class="section">
      <h2>活跃参考图列表（将用于生成）</h2>
      <div class="info">
        共 {{ activeReferenceImages.length }} 张参考图将用于生成
      </div>

      <div class="image-grid">
        <div v-for="(img, index) in activeReferenceImages" :key="img.key" class="card">
          <img :src="img.url" :alt="img.name" class="thumbnail" />
          <div class="card-info">
            <strong>{{ img.name }}</strong>
            <span v-if="img.kind" class="tag">{{ img.kind }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="section">
      <h2>测试说明</h2>
      <ul>
        <li>系统参考图会自动从当前镜头加载（需要集成到 ProfessionalEditor 后才能工作）</li>
        <li>点击"添加参考图"按钮可以上传自定义图片</li>
        <li>悬停在图片上会显示 X 图标</li>
        <li>单击 X 图标会禁用图片（显示遮罩）</li>
        <li>双击禁用的图片可以恢复</li>
        <li>自定义参考图可以删除</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import ReferenceImageManagerPro from '@/components/ReferenceImageManagerPro.vue'

interface ReferenceImage {
  key: string
  name: string
  url: string
  disabled: boolean
  isCustom: boolean
  kind?: string
}

const systemReferenceImages = ref<ReferenceImage[]>([
  {
    key: 'scene:1',
    name: '场景背景 · 室内',
    url: 'https://via.placeholder.com/800x400/FF6B6B/ffffff?text=Scene+Background',
    disabled: false,
    isCustom: false,
    kind: 'scene'
  },
  {
    key: 'character:1',
    name: '角色1',
    url: 'https://via.placeholder.com/200x200/4ECDC4/ffffff?text=Character+1',
    disabled: false,
    isCustom: false,
    kind: 'character'
  }
])

const customReferenceImages = ref<ReferenceImage[]>([])

const activeReferenceImages = computed<ReferenceImage[]>(() => {
  return [...systemReferenceImages.value, ...customReferenceImages.value]
    .filter(img => !img.disabled)
})

const handleAddCustomReference = (reference: ReferenceImage) => {
  const key = `custom:${Date.now()}`
  customReferenceImages.value.push({
    key: key,
    name: reference.name,
    url: reference.originalUrl,
    isCustom: true,
    disabled: false
  })
  console.log('Added custom reference:', reference)
}
</script>

<style scoped>
.test-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

h1 {
  font-size: 32px;
  margin-bottom: 40px;
  color: #333;
}

.section {
  margin-bottom: 40px;
  padding: 24px;
  background: #f5f7fa;
  border-radius: 8px;
}

.section h2 {
  font-size: 24px;
  margin-bottom: 16px;
  color: #333;
}

.info {
  background: #e9ecef;
  padding: 12px;
  border-radius: 4px;
  margin-bottom: 16px;
  font-size: 14px;
  color: #495057;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.card {
  border: 1px solid #dfe6e9;
  border-radius: 8px;
  overflow: hidden;
  background: white;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.thumbnail {
  width: 100%;
  height: 150px;
  object-fit: cover;
  display: block;
}

.card-info {
  padding: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-info strong {
  font-size: 14px;
  color: #333;
}

.tag {
  padding: 4px 8px;
  background: #409eff;
  color: white;
  border-radius: 4px;
  font-size: 12px;
}

.section ul {
  margin: 0;
  padding-left: 20px;
}

.section li {
  margin-bottom: 8px;
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
}
</style>
