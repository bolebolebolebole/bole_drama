-- 添加参考图替换字段到 storyboards 表
-- 创建时间: 2026-01-21

ALTER TABLE storyboards ADD COLUMN reference_overrides TEXT;
