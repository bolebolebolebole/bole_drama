# 火山引擎 AI 服务配置完成报告

## 配置摘要

已成功将所有 AI 服务配置为直接使用火山引擎 API。

### 配置详情

#### 1. 文本大模型（Text）
- **Provider**: `volcengine`
- **Model**: `doubao-seed-1-8-251228`
- **Base URL**: `https://ark.cn-beijing.volces.com/api/v3`
- **Endpoint**: `/chat/completions`
- **API Key**: `6443bc5a-3b57-459a-b15b-8835875f94d0`
- **状态**: ✅ 已启用，设为默认

#### 2. 图片生成模型（Image）
- **Provider**: `volcengine`
- **Model**: `doubao-seedream-4-5-251128` (SeedDream 4.5)
- **Base URL**: `https://ark.cn-beijing.volces.com/api/v3`
- **Endpoint**: `/images/generations`
- **API Key**: `6443bc5a-3b57-459a-b15b-8835875f94d0`
- **状态**: ✅ 已启用，设为默认

#### 3. 视频生成模型（Video）
- **Provider**: `volcengine`
- **Model**: `doubao-seedance-1-5-pro-251215` (SeedDance 1.5 Pro)
- **Base URL**: `https://ark.cn-beijing.volces.com/api/v3`
- **Endpoint**: `/contents/generations/tasks`
- **Query Endpoint**: `/contents/generations/tasks/{taskId}`
- **API Key**: `6443bc5a-3b57-459a-b15b-8835875f94d0`
- **状态**: ✅ 已启用，设为默认

## 配置变更

### 之前的配置
- 使用 ChatFire 代理服务（`https://ai.t8star.cn/v1`）
- 通过代理访问火山引擎模型
- 可能存在代理服务不稳定或配置问题

### 现在的配置
- 直接连接火山引擎官方 API（`https://ark.cn-beijing.volces.com/api/v3`）
- 使用你提供的官方 API Key
- 绕过中间代理，提高稳定性和速度

## 数据库更新记录

```sql
-- 更新文本模型配置
UPDATE ai_service_configs 
SET 
  provider = 'volcengine',
  base_url = 'https://ark.cn-beijing.volces.com/api/v3',
  api_key = '6443bc5a-3b57-459a-b15b-8835875f94d0',
  model = '["doubao-seed-1-8-251228"]',
  is_default = 1,
  is_active = 1
WHERE service_type = 'text';

-- 更新图片模型配置
UPDATE ai_service_configs 
SET 
  provider = 'volcengine',
  base_url = 'https://ark.cn-beijing.volces.com/api/v3',
  api_key = '6443bc5a-3b57-459a-b15b-8835875f94d0',
  model = '["doubao-seedream-4-5-251128"]',
  is_default = 1,
  is_active = 1
WHERE service_type = 'image';

-- 更新视频模型配置
UPDATE ai_service_configs 
SET 
  provider = 'volcengine',
  base_url = 'https://ark.cn-beijing.volces.com/api/v3',
  api_key = '6443bc5a-3b57-459a-b15b-8835875f94d0',
  model = '["doubao-seedance-1-5-pro-251215"]',
  endpoint = '/contents/generations/tasks',
  query_endpoint = '/contents/generations/tasks/{taskId}',
  is_default = 1,
  is_active = 1
WHERE service_type = 'video';
```

## 后端代码支持确认

### 图片生成服务
```go
// application/services/image_generation_service.go
case "volcengine", "volces", "doubao":
    endpoint = "/images/generations"
    queryEndpoint = ""
    return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
```

### 视频生成服务
```go
// application/services/video_generation_service.go
case "doubao", "volcengine", "volces":
    if model == "" {
        model = DefaultDoubaoVideoModel
    }
    endpoint = "/contents/generations/tasks"
    queryEndpoint = "/contents/generations/tasks/{taskId}"
    return video.NewVolcesArkClient(baseURL, apiKey, model, endpoint, queryEndpoint), nil
```

### 文本生成服务
```go
// 使用标准 OpenAI 兼容格式
// 火山引擎支持 OpenAI 格式的 /chat/completions 端点
```

## 服务器状态

✅ **服务器已重启并成功加载新配置**

```
INFO    🚀 Server starting...    {"port": 5678, "mode": "debug"}
INFO    📍 Access URLs:
INFO       Frontend:  http://localhost:5678
INFO       API:       http://localhost:5678/api/v1
INFO       Health:    http://localhost:5678/health
```

## API 调用格式参考

### 文本生成（Chat Completions）
```bash
curl https://ark.cn-beijing.volces.com/api/v3/chat/completions \
  -H "Authorization: Bearer 6443bc5a-3b57-459a-b15b-8835875f94d0" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seed-1-8-251228",
    "messages": [
      {
        "role": "user",
        "content": "你好"
      }
    ]
  }'
```

### 图片生成（Image Generations）
```bash
curl https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Authorization: Bearer 6443bc5a-3b57-459a-b15b-8835875f94d0" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedream-4-5-251128",
    "prompt": "一只可爱的猫咪",
    "size": "2k"
  }'
```

### 视频生成（Video Generations）
```bash
# 创建任务
curl -X POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks \
  -H "Authorization: Bearer 6443bc5a-3b57-459a-b15b-8835875f94d0" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-1-5-pro-251215",
    "content": [
      {
        "type": "text",
        "text": "无人机飞行 --duration 5"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/image.jpg"
        }
      }
    ]
  }'

# 查询任务状态
curl https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/{taskId} \
  -H "Authorization: Bearer 6443bc5a-3b57-459a-b15b-8835875f94d0"
```

## 测试建议

### 1. 测试文本生成
1. 进入系统设置 → AI 配置
2. 查看文本服务配置是否正确
3. 尝试生成角色描述或剧本

### 2. 测试图片生成
1. 进入专业编辑器
2. 选择一个镜头
3. 填写背景描述
4. 点击"生成"按钮
5. 查看是否成功生成图片

### 3. 测试视频生成
1. 确保已有生成的图片
2. 点击"生成视频"按钮
3. 等待 1-3 分钟
4. 查看视频生成状态

## 故障排查

### 如果图片/视频仍然无法生成

#### 检查 1：API Key 是否有效
```bash
# 测试 API Key
curl https://ark.cn-beijing.volces.com/api/v3/chat/completions \
  -H "Authorization: Bearer 6443bc5a-3b57-459a-b15b-8835875f94d0" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seed-1-8-251228",
    "messages": [{"role": "user", "content": "测试"}]
  }'
```

#### 检查 2：模型名称是否正确
- 文本：`doubao-seed-1-8-251228`
- 图片：`doubao-seedream-4-5-251128`
- 视频：`doubao-seedance-1-5-pro-251215`

#### 检查 3：查看服务器日志
```bash
cd src
tail -f server_volcengine.log
# 或
tail -f server.log
```

查找错误信息：
- `API error`
- `unauthorized`
- `invalid model`
- `rate limit`

#### 检查 4：验证数据库配置
```bash
cd src/data
sqlite3 drama_generator.db "
SELECT 
  service_type,
  provider,
  model,
  base_url,
  is_default,
  is_active
FROM ai_service_configs
ORDER BY service_type;
"
```

应该看到：
```
image|volcengine|["doubao-seedream-4-5-251128"]|https://ark.cn-beijing.volces.com/api/v3|1|1
text|volcengine|["doubao-seed-1-8-251228"]|https://ark.cn-beijing.volces.com/api/v3|1|1
video|volcengine|["doubao-seedance-1-5-pro-251215"]|https://ark.cn-beijing.volces.com/api/v3|1|1
```

### 常见错误及解决方案

#### 错误 1：`unauthorized` 或 `invalid api key`
**原因**：API Key 无效或过期
**解决**：
1. 检查 API Key 是否正确
2. 在火山引擎控制台验证 API Key 状态
3. 如需更新，运行：
```bash
cd src/data
sqlite3 drama_generator.db "
UPDATE ai_service_configs 
SET api_key = '新的API_KEY'
WHERE provider = 'volcengine';
"
```

#### 错误 2：`model not found` 或 `invalid model`
**原因**：模型名称错误或模型未开通
**解决**：
1. 在火山引擎控制台确认已开通对应模型
2. 验证模型名称拼写是否正确
3. 检查模型是否在你的账户区域可用

#### 错误 3：`rate limit exceeded`
**原因**：请求频率超过限制
**解决**：
1. 等待一段时间后重试
2. 检查账户配额
3. 考虑升级账户套餐

#### 错误 4：`timeout` 或 `connection refused`
**原因**：网络连接问题
**解决**：
1. 检查网络连接
2. 验证防火墙设置
3. 确认可以访问 `ark.cn-beijing.volces.com`

## 配置文件位置

- **数据库**: `src/data/drama_generator.db`
- **表名**: `ai_service_configs`
- **服务器配置**: `src/configs/config.yaml`

## 下一步操作

1. ✅ 配置已完成
2. ✅ 服务器已重启
3. ⏳ 等待测试验证
4. ⏳ 如有问题，查看上述故障排查部分

## 验证清单

- [x] 文本模型配置已更新
- [x] 图片模型配置已更新
- [x] 视频模型配置已更新
- [x] API Key 已设置
- [x] Base URL 已更新为火山引擎官方地址
- [x] 所有配置已设为默认和启用
- [x] 服务器已重启并加载新配置
- [ ] 用户测试文本生成功能
- [ ] 用户测试图片生成功能
- [ ] 用户测试视频生成功能

## 技术支持

如果遇到问题：
1. 查看服务器日志：`tail -f src/server.log`
2. 检查浏览器控制台（F12）
3. 参考本文档的故障排查部分
4. 验证火山引擎控制台中的 API Key 和模型状态

---

**配置完成时间**: 2026-01-23 02:50
**状态**: ✅ 配置完成，等待用户测试验证
