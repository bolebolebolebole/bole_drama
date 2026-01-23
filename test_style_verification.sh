#!/bin/bash

# Style Verification Test Script
# Tests for style propagation in frame prompt generation

API_BASE="http://localhost:5678/api/v1"

echo "====================================="
echo "Testing Style Propagation"
echo "====================================="
echo ""

# First, let's check if we have storyboards to test with
echo "Step 1: Checking for existing storyboards..."
echo "----------------------------------------"

# Get the list of storyboards (this may fail if none exist, but we'll see the structure)
echo "Trying to query storyboards..."

# We need to get drama IDs first
echo ""
echo "Step 2: Testing frame prompt generation with style"
echo "----------------------------------------"

# Test 1: Generate frame prompt with anime style
echo "Test 1: Anime style"
echo "----------------------------------------"

# We need a real storyboard ID. Let's check the database first
if [ -f "data/drama_generator.db" ]; then
    echo "Checking database for storyboards..."
    STORYBOARD_ID=$(sqlite3 data/drama_generator.db "SELECT id FROM storyboards LIMIT 1;" 2>/dev/null)

    if [ -n "$STORYBOARD_ID" ] && [ "$STORYBOARD_ID" != "NULL" ]; then
        echo "Found storyboard_id: $STORYBOARD_ID"
        echo ""

        # Test with anime style
        echo "Calling frame prompt API with style=anime..."
        RESPONSE1=$(curl -s -X POST "$API_BASE/storyboards/$STORYBOARD_ID/frame-prompt" \
          -H "Content-Type: application/json" \
          -d "{
            \"frame_type\": \"first\",
            \"style\": \"anime\"
          }")

        echo "Response:"
        echo "$RESPONSE1" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE1"
        echo ""

        sleep 2

        # Test with realistic style
        echo "Test 2: Realistic style"
        echo "----------------------------------------"
        echo "Calling frame prompt API with style=realistic..."
        RESPONSE2=$(curl -s -X POST "$API_BASE/storyboards/$STORYBOARD_ID/frame-prompt" \
          -H "Content-Type: application/json" \
          -d "{
            \"frame_type\": \"first\",
            \"style\": \"realistic\"
          }")

        echo "Response:"
        echo "$RESPONSE2" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE2"
        echo ""

        sleep 2

        # Check database for frame prompts
        echo "Step 3: Checking frame prompts in database"
        echo "----------------------------------------"
        echo "Recent frame prompts:"
        sqlite3 data/drama_generator.db "SELECT id, storyboard_id, frame_type, prompt FROM frame_prompts ORDER BY id DESC LIMIT 3;" 2>/dev/null || echo "No frame prompts found"

    else
        echo "No storyboards found in database."
        echo ""
        echo "To test style propagation, you need to:"
        echo "1. Create a drama"
        echo "2. Create storyboards for the drama"
        echo "3. Then run this test again"
    fi
else
    echo "Database not found"
fi

echo ""
echo "====================================="
echo "Test completed"
echo "====================================="
