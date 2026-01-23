# 专业编辑器参考图功能 - 简化实现方案

## 问题说明

由于 ProfessionalEditor.vue 文件较大（5000+ 行），直接编辑模板容易导致语法错误。

## 简化实现方案

### 方案 1：创建独立的参考图管理组件

创建新文件：`src/web/src/components/ReferenceImageManagerPro.vue`

该组件将包含：
1. 自定义参考图列表
2. 禁用/恢复功能
3. 上传功能
4. 继承现有的 ReferenceImageManager.vue 功能

然后在 ProfessionalEditor.vue 中：
```vue
<ReferenceImageManagerPro
  v-model="customReferenceImages"
  :disabled-set="disabledReferenceImages"
  @add="handleAddCustomReference"
/>
```

### 方案 2：使用更简单的模板修改

只修改最小必要部分：

1. 添加导入
2. 添加状态
3. 添加函数
4. 修改参考图显示部分
5. 添加上传对话框

### 方案 3：修改现有 ReferenceImageManager.vue

增强现有的 ReferenceImageManager.vue，使其支持：
1. 预加载参考图（场景+角色）
2. 保留禁用/恢复功能
3. 添加删除功能

## 建议

**推荐使用方案 1**：创建新的 ReferenceImageManagerPro.vue

优点：
- 不需要修改大型文件
- 组件独立，易于测试
- 可以在其他地方复用
- 减少模板语法错误风险

实现步骤：
1. 创建 ReferenceImageManagerPro.vue
2. 在 ProfessionalEditor.vue 中引入和使用
3. 传递初始参考图数据
4. 处理参考图变化事件

## 下一步

请选择一个方案，我将按照该方案实现功能。

或者，如果您希望我继续修复当前的 ProfessionalEditor.vue 文件，我可以：
1. 恢复到原始状态
2. 使用更谨慎的方式逐步实现
3. 每步验证构建是否成功
