# 水印问题分析与解决方案

## 问题分析

根据调查，发现以下情况：

### 当前配置
- **使用的服务**: ChatFire代理服务
- **API端点**: `https://ai.t8star.cn/v1`
- **Provider**: chatfire
- **当前代码**: 已设置 `Watermark: false`

### 水印来源

根据代码分析和API文档调查，水印可能来自以下两个层面：

1. **VolcEngine API层的水印**
   - API文档明确支持 `watermark` 参数
   - 设置 `watermark: false` 可以禁用
   - 当前代码已正确设置此参数

2. **ChatFire代理层的水印**（主要问题）
   - ChatFire是一个第三方代理服务
   - 代理可能在返回图片时添加自己的水印
   - 这层水印无法通过API参数控制

---

## 解决方案

### 方案1：使用VolcEngine官方API（推荐）⭐

**优势**：
- ✅ 完全控制水印参数
- ✅ 无额外代理层水印
- ✅ 直接官方API，性能更稳定
- ✅ 获取最新功能更新

**步骤**：

1. **获取VolcEngine官方API Key**
   - 访问：https://console.volcengine.com/
   - 登录后进入：API Key管理
   - 创建新的API Key（或使用现有的）

2. **修改configs.json，添加VolcEngine官方API配置**

```json
{
  "success": true,
  "data": [
    {
      "id": 4,
      "service_type": "image",
      "provider": "volcengine_official",
      "name": "火山引擎-图片-官方API",
      "base_url": "https://ark.cn-beijing.volces.com/api/v3",
      "api_key": "你的官方API Key",
      "model": [
        "doubao-seedream-4-5-251128",
        "doubao-seedream-4-0-250828"
      ],
      "endpoint": "/images/generations",
      "query_endpoint": "",
      "priority": 0,
      "is_default": true,
      "is_active": true,
      "settings": ""
    }
  ]
}
```

3. **在AI Service中添加volcengine_official provider支持**

需要在 `src/application/services/ai_service.go` 中添加：

```go
case "volcengine_official":
    return image.NewVolcEngineImageClient(
        config.BaseURL,
        config.APIKey,
        model,
        endpoint,
        queryEndpoint,
    ), nil
```

### 方案2：代码中验证水印参数是否正确传递

**当前代码检查**：

文件：`src/pkg/image/volcengine_image_client.go`

```go
reqBody := VolcEngineImageRequest{
    Model:                     model,
    Prompt:                    promptText,
    Image:                     images,
    SequentialImageGeneration: "disabled",
    Size:                      size,
    Width:                     width,
    Height:                    height,
    Watermark:                 false,  // ✅ 已设置为false
}
```

**验证请求是否包含水印参数**：

```go
// 添加更详细的日志
fmt.Printf("[VolcEngine Image] Request with Watermark=%v\n", reqBody.Watermark)
```

### 方案3：后处理去除水印（临时方案）

如果必须使用ChatFire代理，可以使用AI工具后处理去除水印：

**工具1：WatermarkRemover.io**
- URL: https://www.watermarkremover.io/zh
- 免费使用（个人）
- 支持图片、视频
- 最大分辨率：5000 × 5000 px

**实现方式**：

在图片生成后，自动调用去水印API：

```go
// 在 image_generation_service.go 的 ProcessImageGeneration 中
// 添加水印去除逻辑
func (s *ImageGenerationService) removeWatermark(imageURL string) (string, error) {
    // 调用 WatermarkRemover.io API
    // 返回无水印的图片URL
}
```

---

## 推荐实施计划

### 立即执行（今天）

**步骤1：验证当前请求**
1. 查看后端日志，确认 `watermark: false` 是否被发送
2. 如果是，说明水印来自ChatFire代理层

**步骤2：联系ChatFire**
- 发送邮件到：support@chatfire.site
- 询问如何禁用ChatFire代理添加的水印
- 是否有特殊参数或配置选项

### 短期执行（本周）

**步骤3：切换到VolcEngine官方API**
1. 获取官方API Key
2. 创建新的配置项
3. 添加 volcengine_official provider支持
4. 在前端添加服务切换选项

**步骤4：测试无水印生成**
1. 使用官方API生成测试图片
2. 验证确实无水印
3. 对比ChatFire代理的图片质量

### 长期优化（未来）

1. **添加水印去除工具集成**
   - 集成去水印API作为备选方案
   - 提供用户选项：是否使用去水印服务

2. **添加水印状态检测**
   - 自动检测生成的图片是否包含水印
   - 如果检测到水印，自动触发去除流程

---

## API文档参考

### VolcEngine官方文档

**图片生成API**：
- 文档：https://www.volcengine.com/docs/82379/1541523
- Base URL：`https://ark.cn-beijing.volces.com/api/v3`
- Endpoint：`/images/generations`
- 水印参数：`watermark` (boolean)

**请求示例**：

```bash
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-5-251128",
    "prompt": "你的提示词",
    "size": "2560x1440",
    "watermark": false
  }'
```

### ChatFire代理信息

- GitHub：https://github.com/chatfire-AI
- API：https://api.chatfire.site/v1, https://ai.t8star.cn/v1
- 问题：可能添加代理层水印

---

## 快速测试命令

### 测试当前请求参数

查看后端日志：

```bash
# 查看最新的图片生成请求
grep "VolcEngine Image" server_verification.log | tail -10
```

应该看到：

```
[VolcEngine Image] Request Body: {
  "model": "doubao-seedream-4-5-251128",
  "prompt": "...",
  "size": "2560x1440",
  "width": 2560,
  "height": 1440,
  "watermark": false
}
```

### 测试VolcEngine官方API

```bash
# 替换YOUR_API_KEY和PROMPT
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-5-251128",
    "prompt": "测试提示词",
    "size": "2K",
    "watermark": false,
    "response_format": "url"
  }'
```

---

## 代码修改建议

### 修改1：添加详细的日志输出

文件：`src/pkg/image/volcengine_image_client.go`

在发送请求前添加：

```go
fmt.Printf("[VolcEngine Image] Full Request Body: %s\n", string(jsonData))
fmt.Printf("[VolcEngine Image] Watermark Parameter: %v\n", reqBody.Watermark)
```

### 修改2：添加官方API支持

需要修改的文件：

1. `configs.json` - 添加volcengine_official配置
2. `src/application/services/ai_service.go` - 添加volcengine_official case
3. `src/api/handlers/image_generation.go` - 支持新的provider

---

## 总结

| 方案 | 难度 | 效果 | 推荐度 | 说明 |
|------|-------|------|--------|------|
| 方案1：官方API | 中 | 完美无水印 | ⭐⭐⭐⭐⭐ | 最推荐 |
| 方案2：验证参数 | 低 | 可能无效 | ⭐⭐ | 如果参数未正确传递可解决 |
| 方案3：后处理 | 中 | 有一定损失 | ⭐⭐⭐ | 临时方案 |
| 联系ChatFire | 低 | 不确定 | ⭐⭐ | 需等待支持 |

---

## 下一步行动

1. ✅ **立即**：查看后端日志，验证watermark参数
2. 📧 **今天**：联系ChatFire支持
3. 🔧 **本周**：实现官方API切换功能
4. 🧪 **本周**：测试并验证无水印生成
5. 📝 **未来**：添加水印检测和自动去除功能

---

**生成时间**: 2026-01-23
**文档版本**: 1.0
