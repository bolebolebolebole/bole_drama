-- 删除旧的图片生成配置
DELETE FROM ai_service_configs WHERE service_type = 'image';

-- 删除旧的视频生成配置
DELETE FROM ai_service_configs WHERE service_type = 'video';

-- 插入火山引擎图片生成配置
INSERT INTO ai_service_configs (service_type, provider, name, base_url, api_key, model, endpoint, is_active, priority, created_at, updated_at) 
VALUES ('image', 'volcengine', '火山引擎-图片生成', 'https://ark.cn-beijing.volces.com/api/v3', '6443bc5a-3b57-459a-b15b-8835875f94d0', '["doubao-seedream-4-5-251128"]', '/images/generations', 1, 100, datetime('now'), datetime('now'));

-- 插入火山引擎视频生成配置
INSERT INTO ai_service_configs (service_type, provider, name, base_url, api_key, model, endpoint, query_endpoint, is_active, priority, created_at, updated_at) 
VALUES ('video', 'volcengine', '火山引擎-视频生成', 'https://ark.cn-beijing.volces.com/api/v3', '6443bc5a-3b57-459a-b15b-8835875f94d0', '["doubao-seedance-1-5-pro-251215"]', '/contents/generations/tasks', '/generations/tasks/{taskId}', 1, 100, datetime('now'), datetime('now'));
