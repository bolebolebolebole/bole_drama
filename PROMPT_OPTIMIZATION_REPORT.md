# 镜头图片提取提示词优化报告

## 📋 问题描述

**问题**：当前镜头图片提取功能提取出了英文提示词，不符合预期。

**示例问题输出**：
```
Anime style, close-up, slight low angle. Male character Fang Che, sudden movement standing up, reaching hands forward desperately towards viewer. Hand suspended in mid-air, motion blur, dynamic tension. Face twisted in agony, eyes begging and tearful, mouth open pleading. Office background, daylight, warm light rays piercing through heavy grey shadows, desolate atmosphere, high contrast, emotional lighting, 4k.
```

**期望输出**：纯中文提示词

**使用模型**：豆包模型

---

## 🔍 问题根源分析

经过代码审查，发现问题出在以下几个地方：

### 1. **系统提示词约束不够强**
- 虽然提示词中提到"必须使用中文"，但没有给出具体的中文格式示例
- 缺少对比性的错误示例
- 没有明确禁止英文术语的使用

### 2. **存在英文分支代码**
- `image_generation_service.go` 中存在 `IsEnglish()` 判断分支
- 虽然当前配置返回中文，但代码中保留了完整的英文提示词模板
- AI 模型可能受到英文示例的影响

### 3. **豆包模型特性**
- 豆包等国产模型在处理中英混合提示词时，可能倾向于生成英文输出
- 需要更强的约束和明确的中文示例

---

## ✅ 优化方案

采用**方案 A**：增强中文提示词的强制性，移除所有英文支持。

### 优化内容

#### 1. **优化 `GetFirstFramePrompt()` - 首帧提示词**
**文件**：`src/application/services/prompt_i18n.go` (第 130-163 行)

**优化点**：
- ✅ 添加【重要约束】章节，明确"必须100%使用中文"
- ✅ 添加【正确示例】，展示完整的中文提示词格式
- ✅ 添加【错误示例】，明确禁止的英文和中英混杂格式
- ✅ 提供英文术语到中文的对照（如 "close-up" → "特写"）

**示例对比**：

**优化前**：
```
输出格式：
返回一个JSON对象，包含：
- prompt：完整的中文图片生成提示词（详细描述，适合AI图像生成）
- description：简化的中文描述（供参考）
```

**优化后**：
```
【重要约束】
- prompt字段必须100%使用中文，严禁出现任何英文单词、字母或标点符号
- 禁止使用英文术语，如"close-up"应写成"特写"，"anime style"应写成"动漫风格"
- 所有描述必须用中文表达，包括风格、动作、情绪、光线等

【正确示例】
{
  "prompt": "动漫风格，特写镜头，微微仰视角度。男性角色方澈，静止站立姿态，双手自然垂放身侧，表情平静。办公室背景，白天，温暖的阳光透过窗户洒入，明亮氛围，高清画质。",
  "description": "方澈站立的初始状态"
}

【错误示例 - 严禁模仿】
❌ "Anime style, close-up, male character standing..." - 包含英文
❌ "动漫风格，close-up，男性角色..." - 中英混杂
❌ "动漫风格，特写镜头，male character..." - 中英混杂
```

---

#### 2. **优化 `GetKeyFramePrompt()` - 关键帧提示词**
**文件**：`src/application/services/prompt_i18n.go` (第 165-201 行)

**优化点**：
- ✅ 添加【重要约束】章节
- ✅ 添加【正确示例】，使用您提供的实际场景（方澈拼命伸手）
- ✅ 添加【错误示例】，展示禁止的格式
- ✅ 提供英文术语到中文的对照（如 "motion blur" → "动作模糊"）

**正确示例**：
```json
{
  "prompt": "动漫风格，特写镜头，微微仰视角度。男性角色方澈，突然起身动作，双手拼命向前伸向观众方向。手悬停在半空中，动作模糊效果，动态张力十足。面部表情痛苦扭曲，眼神哀求含泪，嘴巴张开呼喊。办公室背景，白天，温暖的光线穿透厚重的灰色阴影，荒凉氛围，高对比度，情绪化光照，超高清画质。",
  "description": "方澈拼命伸手的关键瞬间"
}
```

---

#### 3. **优化 `GetLastFramePrompt()` - 尾帧提示词**
**文件**：`src/application/services/prompt_i18n.go` (第 203-239 行)

**优化点**：
- ✅ 添加【重要约束】章节
- ✅ 添加【正确示例】和【错误示例】
- ✅ 提供英文术语到中文的对照

---

#### 4. **优化 `GetSceneExtractionPrompt()` - 场景提取提示词**
**文件**：`src/application/services/prompt_i18n.go` (第 86-128 行)

**优化点**：
- ✅ 增强"必须100%使用中文"的约束
- ✅ 添加完整的【正确示例】（豪华办公室、城市街道）
- ✅ 添加【错误示例】，包括英文、中英混杂、包含人物等错误
- ✅ 明确标注所有字段必须使用"纯中文"

---

#### 5. **移除英文分支 - `extractBackgroundsFromScript()`**
**文件**：`src/application/services/image_generation_service.go` (第 835-924 行)

**优化点**：
- ✅ 移除 `if s.promptI18n.IsEnglish()` 英文分支（第 841-878 行）
- ✅ 只保留中文格式说明
- ✅ 增强中文约束，添加【重要约束】章节
- ✅ 优化示例，确保所有示例都是纯中文

**优化前**：
```go
var formatInstructions string
if s.promptI18n.IsEnglish() {
    formatInstructions = `[Output JSON Format]...` // 英文模板
} else {
    formatInstructions = `【输出JSON格式】...` // 中文模板
}
```

**优化后**：
```go
// 强制使用中文格式说明（移除英文支持）
formatInstructions := `【输出JSON格式】
{
  "backgrounds": [
    {
      "location": "地点名称（纯中文）",
      "time": "时间描述（纯中文）",
      "atmosphere": "氛围描述（纯中文）",
      "prompt": "动漫风格纯背景场景，展现[地点描述]在[时间]的环境。画面呈现[环境细节、建筑、物品、光线等，不包含人物]。无人物，无角色，空场景。风格：细节丰富，高质量，氛围光照。情绪：[环境情绪描述]。"
    }
  ]
}

【重要约束】
- 所有字段必须100%使用中文，严禁出现任何英文单词、字母或标点符号
- prompt字段必须是纯中文描述，禁止使用英文术语
- 禁止使用英文术语，如"background"应写成"背景"，"scene"应写成"场景"，"anime style"应写成"动漫风格"
...
```

---

#### 6. **移除英文分支 - `extractBackgroundsWithAI()`**
**文件**：`src/application/services/image_generation_service.go` (第 966-1043 行)

**优化点**：
- ✅ 移除 `if s.promptI18n.IsEnglish()` 英文分支（第 971-1006 行）
- ✅ 只保留中文格式说明
- ✅ 增强中文约束
- ✅ 优化示例格式

---

## 📊 优化效果对比

### 优化前的问题
```
❌ 输出：Anime style, close-up, slight low angle. Male character Fang Che...
```

### 优化后的预期输出
```
✅ 输出：动漫风格，特写镜头，微微仰视角度。男性角色方澈，突然起身动作，双手拼命向前伸向观众方向。手悬停在半空中，动作模糊效果，动态张力十足。面部表情痛苦扭曲，眼神哀求含泪，嘴巴张开呼喊。办公室背景，白天，温暖的光线穿透厚重的灰色阴影，荒凉氛围，高对比度，情绪化光照，超高清画质。
```

---

## 🎯 关键优化策略

### 1. **多层次约束**
- 在系统提示词开头明确约束
- 在格式说明中重复约束
- 在示例中展示约束效果

### 2. **正反示例对比**
- 提供完整的正确示例（纯中文）
- 提供明确的错误示例（英文、中英混杂）
- 使用 ❌ 符号标记错误示例，增强视觉效果

### 3. **术语对照表**
- 明确列出常见英文术语的中文对应
- 帮助 AI 模型理解如何转换

### 4. **移除干扰因素**
- 完全移除英文分支代码
- 避免 AI 模型受到英文模板的影响

---

## 🔧 修改文件清单

| 文件 | 修改内容 | 行数 |
|------|---------|------|
| `src/application/services/prompt_i18n.go` | 优化 `GetFirstFramePrompt()` | 130-163 |
| `src/application/services/prompt_i18n.go` | 优化 `GetKeyFramePrompt()` | 165-201 |
| `src/application/services/prompt_i18n.go` | 优化 `GetLastFramePrompt()` | 203-239 |
| `src/application/services/prompt_i18n.go` | 优化 `GetSceneExtractionPrompt()` | 86-128 |
| `src/application/services/image_generation_service.go` | 移除英文分支 `extractBackgroundsFromScript()` | 835-924 |
| `src/application/services/image_generation_service.go` | 移除英文分支 `extractBackgroundsWithAI()` | 966-1043 |

---

## ✅ 验证结果

### 代码语法检查
- ✅ `prompt_i18n.go` - 无语法错误
- ✅ `image_generation_service.go` - 无语法错误

### 功能完整性
- ✅ 保留所有原有功能
- ✅ 只移除英文支持，不影响中文功能
- ✅ 所有函数签名保持不变

---

## 🚀 测试建议

### 1. **重新编译项目**
```bash
cd E:\AI\Bole_drama
go build -o bole-drama.exe .
```

### 2. **测试场景**
- 测试首帧提示词生成
- 测试关键帧提示词生成
- 测试尾帧提示词生成
- 测试场景背景提取

### 3. **验证要点**
- ✅ 生成的 prompt 字段是否为纯中文
- ✅ 是否包含任何英文单词或字母
- ✅ 中文描述是否详细且符合要求
- ✅ JSON 格式是否正确

---

## 📝 后续建议

### 1. **监控 AI 输出**
建议在日志中添加 prompt 语言检测，监控是否还有英文输出：

```go
// 在 completeImageGeneration 或相关函数中添加
if containsEnglish(result.Prompt) {
    s.log.Warnw("Detected English in prompt", 
        "prompt", result.Prompt,
        "image_gen_id", imageGenID)
}
```

### 2. **A/B 测试**
- 对比优化前后的提示词质量
- 收集用户反馈

### 3. **持续优化**
- 根据实际使用情况调整示例
- 针对豆包模型的特性进一步优化

---

## 🎉 总结

本次优化通过以下措施，确保镜头图片提取功能生成**纯中文提示词**：

1. ✅ **增强约束**：在多个层次明确"必须100%使用中文"
2. ✅ **提供示例**：添加完整的正确示例和错误示例对比
3. ✅ **术语对照**：明确英文术语的中文对应
4. ✅ **移除干扰**：完全移除英文分支代码
5. ✅ **针对豆包**：优化提示词结构，适配豆包模型特性

**预期效果**：AI 模型将严格按照中文格式生成提示词，不再出现英文或中英混杂的情况。

---

**优化完成时间**：2026-01-23  
**优化人员**：Sisyphus (OhMyOpenCode AI Agent)  
**使用模型**：豆包模型  
**优化方案**：方案 A - 增强中文提示词强制性
