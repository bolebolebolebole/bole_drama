# 专业编辑器参考图功能实现方案

## 问题分析

当前实现存在模板语法错误，需要重新实现。

## 实现需求

1. ✅ **添加自定义参考图按钮**：在参考图部分添加"+"按钮
2. ✅ **禁用/恢复功能**：每个参考图可以禁用（X图标）和恢复（双击）
3. ✅ **数据结构更新**：支持禁用状态和自定义参考图
4. ✅ **生成逻辑更新**：过滤掉禁用的参考图

## 实现方案

### 步骤 1：更新数据类型

```typescript
type ReferenceImageSlot = {
  key: string
  kind: 'scene' | 'character' | 'custom'
  name: string
  originalUrl: string
  disabled?: boolean
  custom?: boolean
}
```

### 步骤 2：添加状态变量

```typescript
const customReferenceImages = ref<ReferenceImageSlot[]>([])
const disabledReferenceImages = ref<Set<string>>(new Set())
const showCustomImageUpload = ref(false)
const selectedCustomFile = ref<File | null>(null)
```

### 步骤 3：更新 referenceSlots 计算属性

在原有的场景和角色参考图基础上，添加自定义参考图。

### 步骤 4：添加处理函数

1. `toggleDisableReference` - 禁用/启用参考图
2. `handleAddCustomReferenceImage` - 添加自定义参考图
3. `handleCustomFileChange` - 处理文件选择
4. `handleCustomUploadConfirm` - 确认上传
5. `removeCustomReference` - 删除自定义参考图

### 步骤 5：更新模板

1. 为每个参考图添加：
   - `.image-container` 包装器（用于禁用遮罩和 X 按钮）
   - `.close-button`（X图标，悬停显示）
   - `.disabled-overlay`（禁用遮罩）
   - 双击事件处理

2. 添加"添加参考图"按钮

3. 添加自定义参考图上传对话框

### 步骤 6：更新样式

```scss
.image-container {
  position: relative;
  width: 100%;
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

  &:hover {
    transform: scale(1.1);
    background: rgba(245, 108, 108, 1);
  }
}

.reference-image-item:hover .close-button {
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

  .disabled-text {
    color: white;
    font-size: 14px;
    font-weight: 500;
  }
}

.reference-image-item.disabled {
  opacity: 0.5;

  .reference-image-thumbnail {
    filter: grayscale(100%);
  }
}
```

### 步骤 7：更新 generateFrameImage 函数

```typescript
const referenceImages = referenceSlots.value
  .filter(slot => !slot.disabled)  // 过滤掉禁用的参考图
  .map(slot => getReferenceImageUrl(slot))
  .filter((url): url is string => typeof url === 'string' && url.length > 0)
```

## 注意事项

1. 确保模板标签正确闭合
2. 确保所有新增的 div 都有对应的闭合标签
3. 确保双击事件只在禁用的图片上触发
4. 确保 X 按钮只在启用的图片上显示（悬停）

## 测试计划

1. 测试添加自定义参考图
2. 测试禁用参考图（单击 X）
3. 测试恢复参考图（双击）
4. 测试生成图片时只使用启用的参考图
5. 测试删除自定义参考图

## 预期结果

- 用户可以看到场景背景、角色和自定义参考图
- 用户可以禁用任意参考图（单击 X）
- 用户可以恢复已禁用的参考图（双击）
- 用户可以添加自定义参考图（点击 "+" 按钮）
- 用户可以删除自定义参考图
- 生成图片时，只有启用的参考图会被使用
