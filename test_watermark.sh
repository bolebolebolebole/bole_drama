#!/bin/bash

# Watermark Test Script
# Test to verify watermark parameter is being sent correctly

API_BASE="http://localhost:5678/api/v1"

echo "====================================="
echo "Testing Watermark Parameter"
echo "====================================="
echo ""

# Test: Generate image with watermark=false
echo "Test: Generate image to check watermark parameter"
echo "----------------------------------------"

RESPONSE=$(curl -s -X POST "$API_BASE/images" \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A beautiful mountain landscape at sunset, no watermark, clean image, high quality",
    "image_type": "scene",
    "provider": "volcengine",
    "model": "doubao-seedream-4-5-251128",
    "size": "2560x1440"
  }')

echo "API Response:"
echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

# Extract image generation ID
IMAGE_ID=$(echo "$RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin).get('data', {}).get('id', 'N/A'))" 2>/dev/null)

if [ "$IMAGE_ID" != "N/A" ]; then
  echo "✅ Image generation initiated with ID: $IMAGE_ID"
  echo ""
  echo "📋 To check logs, run:"
  echo "   tail -50 watermark_test.log | grep 'VolcEngine Image'"
  echo ""
  echo "🔍 Look for this line in the logs:"
  echo "   [VolcEngine Image] Watermark Parameter Value: false"
  echo ""
else
  echo "❌ Failed to initiate image generation"
fi

echo ""
echo "====================================="
echo "Test completed"
echo "====================================="
echo ""
echo "Next steps:"
echo "1. Check the log file for watermark parameter"
echo "2. Wait for image generation to complete"
echo "3. Download the generated image"
echo "4. Check if it has watermark"
echo ""
echo "💡 If watermark still appears:"
echo "   - It may be added by ChatFire proxy layer"
echo "   - Consider using VolcEngine official API (see WATERMARK_SOLUTION.md)"
echo "   - Contact ChatFire support: support@chatfire.site"
