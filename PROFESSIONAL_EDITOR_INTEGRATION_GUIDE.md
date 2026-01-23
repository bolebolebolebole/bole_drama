# 专业编辑器参考图功能 - 完整实现指南

## 当前状态

由于 ProfessionalEditor.vue 文件较大（5000+ 行），直接编辑导致模板语法错误。

## 解决方案：使用新创建的 ReferenceImageManagerPro 组件

我已经创建了新组件：`src/web/src/components/ReferenceImageManagerPro.vue`

### 新组件功能

1. ✅ 显示系统参考图（场景背景 + 角色）
2. ✅ 支持添加自定义参考图
3. ✅ 禁用/恢复功能（X 图标 + 双击）
4. ✅ 删除自定义参考图功能
5. ✅ 拖拽上传支持

## 集成步骤

### 步骤 1：在 ProfessionalEditor.vue 中导入组件

在 script 部分的 import 语句中添加：

```typescript
import ReferenceImageManagerPro from '@/components/ReferenceImageManagerPro.vue'
```

### 步骤 2：添加状态变量

在 script 部分的状态声明区域添加：

```typescript
// 现有的参考图相关状态
const referenceImageOverrides = ref<Record<string, string>>({})

// 新增状态
const customReferenceImages = ref<ReferenceImageSlot[]>([])
const showCustomImageUpload = ref(false)
const selectedCustomFile = ref<File | null>(null)
const uploadAction = computed(() => '/api/v1/upload/image')
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
}))
```

### 步骤 3：更新 referenceSlots 计算属性

```typescript
const referenceSlots = computed<ReferenceImageSlot[]>(() => {
  const slots: ReferenceImageSlot[] = []
  const sb = currentStoryboard.value
  if (!sb) return slots
  
  // 场景背景
  const sceneUrl = sb.background?.image_url
  if (sceneUrl) {
    const sceneNameParts = [sb.background?.location, sb.background?.time].filter(Boolean)
    slots.push({
      key: `scene:${sb.id}`,
      kind: 'scene',
      name: sceneNameParts.length > 0 ? sceneNameParts.join(' · ') : '场景背景',
      originalUrl: sceneUrl
    })
  }
  
  // 角色
  const storyboardCharacters = currentStoryboardCharacters.value
  if (Array.isArray(storyboardCharacters) && storyboardCharacters.length > 0) {
    storyboardCharacters.forEach((char: any) => {
      const url = char?.image_url
      if (!url) return
      slots.push({
        key: `character:${char.id}`,
        kind: 'character',
        name: char?.name || `角色${char.id}`,
        originalUrl: url
      })
    })
  }
  
  // 自定义参考图
  if (Array.isArray(customReferenceImages.value)) {
    customReferenceImages.value.forEach((img, index) => {
      const url = img.url
      if (!url) return
      slots.push({
        key: img.key,
        kind: 'custom',
        name: img.name || `自定义${index + 1}`,
        originalUrl: url,
        isCustom: true,
        disabled: img.disabled
      })
    })
  }
  
  return slots
})
```

### 步骤 4：更新模板

找到现有的参考图部分（大约在第 283-311 行），替换为：

```vue
<!-- 参考图 -->
<div class="reference-images-section">
  <div class="section-label">
    参考图
  </div>
  
  <!-- 使用新的 ReferenceImageManagerPro 组件 -->
  <ReferenceImageManagerPro
    v-model="customReferenceImages"
    :system-reference-images="referenceSlots.value.filter(s => !s.isCustom)"
    @add="handleAddCustomReference"
  />
</div>
```

### 步骤 5：添加处理函数

```typescript
// 处理添加自定义参考图（从系统参考图添加）
const handleAddCustomReference = (reference: ReferenceImageSlot) => {
  const key = `custom:${Date.now()}`
  customReferenceImages.value.push({
    key: key,
    name: reference.name,
    url: reference.originalUrl,
    isCustom: true,
    disabled: false
  })
  ElMessage.success('参考图已添加到自定义列表')
}
```

### 步骤 6：更新图片生成逻辑

找到 `generateFrameImage` 函数，更新参考图收集逻辑：

```typescript
const generateFrameImage = async () => {
  if (!currentStoryboard.value || !currentFramePrompt.value) return
  
  generatingImage.value = true
  try {
    // 收集参考图片URL（过滤掉禁用的参考图）
    const referenceImages = referenceSlots.value
      .filter(slot => !slot.disabled)
      .map(slot => {
        const url = getReferenceImageUrl(slot)
        return url
      })
      .filter((url): url is string => typeof url === 'string' && url.length > 0)
    
    // ... 其余代码保持不变
    
    const result = await imageAPI.generateImage({
      drama_id: dramaId.toString(),
      prompt: currentFramePrompt.value,
      storyboard_id: currentStoryboard.value.id,
      image_type: 'storyboard',
      frame_type: selectedFrameType.value,
      reference_images: referenceImages.length > 0 ? referenceImages : undefined,
      provider: 'volcengine',
      model: 'doubao-seedream-4-5-251128',
      size: size,
      style: style.value
    })
    
    // ... 其余代码保持不变
  } catch (error: any) {
    ElMessage.error('生成失败: ' + (error.message || '未知错误'))
  } finally {
    generatingImage.value = false
  }
}
```

### 步骤 7：添加样式

在 style 部分添加：

```scss
.reference-images-section {
  margin-top: 14px;
  
  .section-label {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 10px;
  }
}
```

## 测试验证

集成后，测试以下功能：

1. ✅ 查看系统参考图（场景背景 + 角色）
2. ✅ 点击"+"按钮添加自定义参考图
3. ✅ 在自定义参考图上悬停显示 X 图标
4. ✅ 单击 X 禁用参考图
5. ✅ 双击禁用的参考图恢复
6. ✅ 删除自定义参考图
7. ✅ 生成图片时只使用启用的参考图

## 注意事项

1. 所有修改都应该在现有代码基础上进行
2. 确保所有标签正确闭合
3. 每次修改后运行 `npm run build` 验证
4. 如有错误，使用 git 回滚或恢复备份

## 快速开始

如果您希望我直接帮您完成集成，请：

1. 备份当前的 ProfessionalEditor.vue
2. 告诉我准备好进行修改
3. 我将逐步进行，每次修改后验证构建

或者，您可以根据上述步骤手动完成集成。
