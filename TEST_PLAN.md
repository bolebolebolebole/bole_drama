# 测试计划 / Test Plan

## 问题修复总结 / Summary of Fixes

### 问题 1: 图片画幅不生效 / Issue 1: Aspect Ratio Not Applied

**根本原因 / Root Cause:**
- VolcEngine API 需要明确的 `width` 和 `height` 参数，而不仅仅是 `size` 字符串

**修复内容 / Fix:**
1. 更新 `src/pkg/image/volcengine_image_client.go`
   - 优先使用明确的 width/height 参数
   - 如果没有明确的 width/height，从 size 字符串解析
   - 同时发送 size 字符串和 width/height 到 API

**测试用例 / Test Cases:**

#### Test 1.1: 横屏图片生成 / Landscape Image Generation
- **步骤 / Steps:**
  1. 启动后端服务
  2. 打开前端，进入专业编辑器
  3. 选择一个分镜
  4. 在"画幅"下拉框中选择"16:9"
  5. 生成图片

- **预期结果 / Expected Results:**
  - 后端日志显示: `final size = 2560x1440, parsed width=2560, height=1440`
  - API 请求包含: `"size":"2560x1440","width":2560,"height":1440`
  - 生成的图片实际尺寸为 2560x1440 或接近该比例

- **验证方法 / How to Verify:**
  ```bash
  # 查看后端日志
  grep "VolcEngine Image" server_correct.log

  # 检查图片实际尺寸（需要图片URL）
  curl -I <generated_image_url> | grep Content-Length
  ```

#### Test 1.2: 竖屏图片生成 / Portrait Image Generation
- **步骤 / Steps:**
  1. 在"画幅"下拉框中选择"9:16"
  2. 生成图片

- **预期结果 / Expected Results:**
  - 后端日志显示: `final size = 1440x2560, parsed width=1440, height=2560`
  - API 请求包含: `"size":"1440x2560","width":1440,"height":2560`
  - 生成的图片实际尺寸为 1440x2560 或接近该比例

- **验证方法 / How to Verify:**
  ```bash
  # 查看后端日志
  grep "VolcEngine Image" server_correct.log | grep "1440x2560"
  ```

---

### 问题 2: 风格设置不传递到提示词生成 / Issue 2: Style Settings Not Passed to Prompt Generation

**根本原因 / Root Cause:**
- 前端在调用帧提示词生成 API 时没有传递 `style` 参数
- 后端 fallback 提示词使用硬编码的"动漫风格"

**修复内容 / Fix:**
1. 更新 `src/web/src/views/drama/ProfessionalEditor.vue`
   - 在 `extractFramePrompt` 函数中传递 `style: style.value`

2. 更新 `src/web/src/api/frame.ts`
   - 所有帧提示词生成函数添加可选的 `style` 参数

3. 更新 `src/application/services/frame_prompt_service.go`
   - 修改 `buildFallbackPrompt` 函数，根据 style 参数映射到具体的风格描述

**测试用例 / Test Cases:**

#### Test 2.1: 动漫风格提示词生成 / Anime Style Prompt Generation
- **步骤 / Steps:**
  1. 在"风格"下拉框中选择"动漫"
  2. 点击"提取提示词"按钮
  3. 查看生成的提示词

- **预期结果 / Expected Results:**
  - 提示词中包含 "动漫风格" 或 "anime style" 相关描述
  - 如果 AI 生成失败，fallback 提示词包含: "动漫风格, anime style"
  - 提示词体现动漫风格的视觉特征（如：夸张的表情、鲜艳的色彩等）

- **验证方法 / How to Verify:**
  ```bash
  # 查看后端日志中的提示词
  grep "AI Prompt for Background" server_correct.log | grep "anime"

  # 或者查看帧提示词记录
  # 在前端 UI 中检查提示词内容
  ```

#### Test 2.2: 写实风格提示词生成 / Realistic Style Prompt Generation
- **步骤 / Steps:**
  1. 在"风格"下拉框中选择"写实"
  2. 点击"提取提示词"按钮
  3. 查看生成的提示词

- **预期结果 / Expected Results:**
  - 提示词中包含 "写实风格" 或 "realistic style" 相关描述
  - 如果 AI 生成失败，fallback 提示词包含: "写实风格, realistic style"
  - 提示词体现写实风格的视觉特征（如：真实的光照、细腻的纹理等）

- **验证方法 / How to Verify:**
  ```bash
  # 查看后端日志中的提示词
  grep "AI Prompt for Background" server_correct.log | grep "realistic"
  ```

#### Test 2.3: 图片生成时应用风格 / Style Applied in Image Generation
- **步骤 / Steps:**
  1. 选择"动漫"风格
  2. 生成图片
  3. 切换到"写实"风格
  4. 生成另一张图片

- **预期结果 / Expected Results:**
  - 动漫风格生成的图片具有动漫特征
  - 写实风格生成的图片具有写实特征
  - 两种风格的图片有明显的视觉差异

- **验证方法 / How to Verify:**
  - 在前端 UI 中对比两张生成的图片
  - 检查图片的视觉风格是否符合预期

---

## 自动化验证 / Automated Verification

### 脚本 1: 测试画幅设置 / Script 1: Test Aspect Ratio Settings
```bash
#!/bin/bash
# test_aspect_ratio.sh

echo "测试画幅设置..."

# 启动后端（如果未运行）
# cd src && go run main.go &
# SERVER_PID=$!
# sleep 5

# 测试 1: 横屏
echo "测试横屏 16:9..."
# 调用生成图片 API，设置 size=2560x1440
# 检查日志中是否包含 "width=2560" 和 "height=1440"

# 测试 2: 竖屏
echo "测试竖屏 9:16..."
# 调用生成图片 API，设置 size=1440x2560
# 检查日志中是否包含 "width=1440" 和 "height=2560"

# 清理
# kill $SERVER_PID
```

### 脚本 2: 测试风格传递 / Script 2: Test Style Propagation
```bash
#!/bin/bash
# test_style_propagation.sh

echo "测试风格传递..."

# 测试 1: 动漫风格
echo "测试动漫风格..."
# 调用帧提示词生成 API，设置 style=anime
# 检查返回的提示词中是否包含 "anime" 或 "动漫"

# 测试 2: 写实风格
echo "测试写实风格..."
# 调用帧提示词生成 API，设置 style=realistic
# 检查返回的提示词中是否包含 "realistic" 或 "写实"
```

---

## 回归测试 / Regression Testing

### 确保原有功能正常 / Ensure Existing Features Work

1. **分镜编辑 / Storyboard Editing**
   - [ ] 可以正常编辑分镜信息
   - [ ] 可以保存和加载分镜

2. **图片生成 / Image Generation**
   - [ ] 可以生成角色图片
   - [ ] 可以生成场景图片
   - [ ] 可以生成分镜图片

3. **视频生成 / Video Generation**
   - [ ] 可以从单张图片生成视频
   - [ ] 可以从多张图片生成视频
   - [ ] 可以使用首尾帧模式

4. **提示词生成 / Prompt Generation**
   - [ ] 可以生成首帧提示词
   - [ ] 可以生成关键帧提示词
   - [ ] 可以生成尾帧提示词
   - [ ] 可以生成分镜板提示词

---

## 手动测试检查清单 / Manual Testing Checklist

### 画幅测试 / Aspect Ratio Testing
- [ ] 横屏 (16:9) 图片生成正确
- [ ] 竖屏 (9:16) 图片生成正确
- [ ] 画幅设置在不同分镜间保持
- [ ] 刷新页面后画幅设置保持

### 风格测试 / Style Testing
- [ ] 动漫风格提示词生成正确
- [ ] 写实风格提示词生成正确
- [ ] 风格设置在不同分镜间保持
- [ ] 刷新页面后风格设置保持
- [ ] 生成的图片符合选择的风格
- [ ] 生成的视频符合选择的风格

### 集成测试 / Integration Testing
- [ ] 同时使用画幅和风格设置生成图片
- [ ] 风格设置传递到视频生成
- [ ] 批量生成时使用正确的画幅和风格

---

## 成功标准 / Success Criteria

所有测试用例通过，并且：
1. ✅ 后端日志显示正确的 width/height 值
2. ✅ API 请求包含正确的 size、width 和 height 参数
3. ✅ 生成的图片实际尺寸符合预期的画幅比例
4. ✅ 提示词中包含正确的风格描述（动漫/写实）
5. ✅ 生成的图片和视频视觉风格符合选择
6. ✅ 所有原有功能正常工作，无回归问题

---

## 问题排查 / Troubleshooting

### 如果画幅仍然不生效 / If Aspect Ratio Still Not Working

1. 检查后端日志中是否有正确的 width/height
2. 确认 VolcEngine API 确实接收到了参数
3. 检查生成的图片实际尺寸
4. 确认前端正确传递了 size 参数

### 如果风格仍然不生效 / If Style Still Not Working

1. 检查后端日志中是否收到 style 参数
2. 确认提示词生成时使用了 style 参数
3. 检查 fallback 提示词中是否包含风格描述
4. 确认图片生成请求中包含了 style 参数

---

## 测试结果 / Test Results

### 画幅测试结果 / Aspect Ratio Test Results
- Test 1.1 (横屏): [ ] 通过 / [ ] 失败
- Test 1.2 (竖屏): [ ] 通过 / [ ] 失败

### 风格测试结果 / Style Test Results
- Test 2.1 (动漫): [ ] 通过 / [ ] 失败
- Test 2.2 (写实): [ ] 通过 / [ ] 失败
- Test 2.3 (图片生成): [ ] 通过 / [ ] 失败

### 回归测试结果 / Regression Test Results
- [ ] 所有测试通过
