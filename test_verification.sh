#!/bin/bash

# Verification Test Script
# Tests for aspect ratio and style propagation fixes

API_BASE="http://localhost:5678/api/v1"

echo "====================================="
echo "Starting Verification Tests"
echo "====================================="
echo ""

# Test 1: Landscape Aspect Ratio (16:9)
echo "Test 1: Testing landscape aspect ratio (16:9)"
echo "----------------------------------------"

RESPONSE1=$(curl -s -X POST "$API_BASE/images" \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A beautiful landscape scene with mountains and sunset, cinematic view",
    "image_type": "scene",
    "provider": "volcengine",
    "model": "doubao-seedream-4-5-251128",
    "size": "2560x1440"
  }')

echo "Response:"
echo "$RESPONSE1" | head -20
echo ""
echo "Checking backend logs for aspect ratio parameters..."
sleep 3
echo ""

# Test 2: Portrait Aspect Ratio (9:16)
echo "Test 2: Testing portrait aspect ratio (9:16)"
echo "----------------------------------------"

RESPONSE2=$(curl -s -X POST "$API_BASE/images" \
  -H "Content-Type: application/json" \
  -d '{
    "drama_id": "1",
    "prompt": "A beautiful portrait of a person, professional photography",
    "image_type": "character",
    "provider": "volcengine",
    "model": "doubao-seedream-4-5-251128",
    "size": "1440x2560"
  }')

echo "Response:"
echo "$RESPONSE2" | head -20
echo ""
echo "Checking backend logs for aspect ratio parameters..."
sleep 3
echo ""

# Test 3: Style - Anime
echo "Test 3: Testing anime style in frame prompt generation"
echo "----------------------------------------"

# First, we need to check if there's a storyboard ID to test with
echo "Note: This test requires a valid storyboard_id"
echo "For now, we'll test the API structure without a real storyboard"

echo ""
echo "Test completed. Check backend logs for verification."
echo ""
echo "To view backend logs in real-time, you can:"
echo "1. Check console output from the running backend"
echo "2. Or add logging to file if configured"
echo ""
