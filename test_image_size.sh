#!/bin/bash

# 测试图片尺寸功能
echo "=== 图片尺寸修复验证测试 ==="
echo ""
echo "测试1: 生成 2048x1152 尺寸的图片（横向）"
echo "----------------------------------------"
curl -X POST http://localhost:5678/api/v1/images \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A beautiful sunset over mountains",
    "provider": "volcengine",
    "size": "2048x1152"
  }' | head -20

echo ""
echo ""
echo "测试2: 生成 1152x2048 尺寸的图片（纵向）"
echo "----------------------------------------"
curl -X POST http://localhost:5678/api/v1/images \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A tall skyscraper in the city",
    "provider": "volcengine",
    "size": "1152x2048"
  }' | head -20

echo ""
echo ""
echo "测试3: 使用明确的 width 和 height 参数"
echo "----------------------------------------"
curl -X POST http://localhost:5678/api/v1/images \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A square image of a flower",
    "provider": "volcengine",
    "width": 1024,
    "height": 1024
  }' | head -20

echo ""
echo ""
echo "测试完成！请查看服务器日志确认是否正确传递了 width 和 height 参数。"
echo "运行以下命令查看日志："
echo "tail -f server.log"
