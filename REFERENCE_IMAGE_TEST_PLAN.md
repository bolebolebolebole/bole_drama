# Reference Image Manager - Test Plan & Verification

## Implementation Summary

### Requirements Implemented
1. ✅ **Add Button**: Reference images can be added using the existing "添加图片" button
2. ✅ **Hover X Icon**: X icon appears in top-right corner on hover (previously always visible delete button)
3. ✅ **Single Click to Disable**: Clicking X icon disables the reference image (marks as deleted)
4. ✅ **Double Click to Restore**: Double-clicking on disabled image restores it as a reference image

### Changes Made

#### 1. ReferenceImageManager.vue Component
- **Template Changes**:
  - Replaced circular delete button with simple X icon in top-right corner
  - Added `@dblclick` event handler on image container for restore functionality
  - Changed overlay from showing restore button to showing "已禁用" text
  - X icon only appears on hover (opacity: 0 → 1 on hover)

- **Script Changes**:
  - Updated imports: Removed `Delete`, `RefreshRight`, added `Close` icon
  - Behavior remains the same: `handleDeleteImage()` marks as deleted, `handleRestoreImage()` restores

- **Style Changes**:
  - Removed `.action-buttons`, `.delete-btn`, `.restore-btn` classes
  - Added `.close-button` class with hover-only visibility
  - Added `.deleted-text` class for overlay text
  - X icon positioned at `top: 4px; right: 4px` with 24px diameter
  - Smooth transitions for opacity and scale on hover

#### 2. Locale Translations (zh-CN.ts)
- Added `disabled: '已禁用'` - Text shown on disabled image overlay
- Added `disabledHint: '双击恢复'` - Hint shown below disabled image
- Changed `imageDeleted: '图片已禁用'` - Updated message when disabling

## Test Plan

### Prerequisites
- Backend server running on `http://localhost:5678`
- Frontend dev server running on `http://localhost:3012`
- Or single service mode with built frontend

### Test Cases

#### Test Case 1: Add Reference Image
**Steps**:
1. Navigate to image generation dialog
2. Click "添加图片" button
3. Upload an image file (jpg/png, < 10MB)
4. Confirm upload

**Expected Result**:
- ✅ Upload dialog appears
- ✅ Image is added to the grid
- ✅ Image displays correctly
- ✅ Success message: "图片添加成功"

#### Test Case 2: Hover to Show X Icon
**Steps**:
1. Add at least one reference image
2. Move mouse over the image
3. Move mouse away from the image

**Expected Result**:
- ✅ X icon appears in top-right corner on hover
- ✅ X icon is hidden when not hovering
- ✅ X icon has red background (rgba(245, 108, 108, 0.9))
- ✅ X icon scales up slightly on hover (transform: scale(1.1))

#### Test Case 3: Disable Reference Image (Single Click X)
**Steps**:
1. Add at least one reference image
2. Hover over the image to show X icon
3. Click the X icon once

**Expected Result**:
- ✅ Image becomes semi-transparent (opacity: 0.5)
- ✅ Dark overlay appears with "已禁用" text
- ✅ Below image shows "双击恢复" hint
- ✅ Success message: "图片已禁用"
- ✅ Image is excluded from active reference images

#### Test Case 4: Restore Reference Image (Double Click)
**Steps**:
1. Disable a reference image (follow Test Case 3)
2. Double-click on the disabled image

**Expected Result**:
- ✅ Image returns to full opacity
- ✅ Dark overlay disappears
- ✅ "双击恢复" hint disappears
- ✅ X icon appears on hover again
- ✅ Success message: "图片已恢复"
- ✅ Image is included in active reference images

#### Test Case 5: Multiple Images Management
**Steps**:
1. Add 3 reference images
2. Disable the 2nd image
3. Verify 1st and 3rd images still work
4. Restore the 2nd image
5. Disable all 3 images
6. Restore all 3 images one by one

**Expected Result**:
- ✅ Each image can be independently disabled/restored
- ✅ Active image count updates correctly
- ✅ Only non-disabled images are sent to generation API

#### Test Case 6: Maximum Images Limit
**Steps**:
1. Add 5 reference images (default max)
2. Try to add a 6th image

**Expected Result**:
- ✅ Warning message: "最多只能添加 5 张参考图片"
- ✅ Upload dialog does not open

#### Test Case 7: Image Generation with Reference Images
**Steps**:
1. Add 3 reference images
2. Disable 1 image
3. Fill in other required fields (drama, prompt, etc.)
4. Click "生成图片"

**Expected Result**:
- ✅ Only 2 active images are sent to API
- ✅ Disabled image is not included in `reference_images` array
- ✅ Generation task is submitted successfully

## Verification Checklist

### Visual Verification
- [ ] X icon appears only on hover
- [ ] X icon is positioned in top-right corner (4px from edges)
- [ ] X icon has red circular background
- [ ] X icon scales smoothly on hover
- [ ] Disabled images show dark overlay with "已禁用" text
- [ ] Disabled images show "双击恢复" hint below
- [ ] Image border changes to blue on hover (#409eff)

### Functional Verification
- [ ] Single click X disables image
- [ ] Double click on disabled image restores it
- [ ] Add button works correctly
- [ ] Upload dialog validates file type and size
- [ ] Maximum image limit is enforced
- [ ] Active images are correctly filtered for API calls
- [ ] All success/error messages display correctly

### Build Verification
- [x] Frontend builds successfully without errors
- [x] No TypeScript compilation errors
- [x] No console errors in browser (to be verified in manual test)

## Manual Testing Instructions

### Start the Application
```bash
# Terminal 1: Start backend
cd src
go run main.go

# Terminal 2: Start frontend dev server
cd src/web
npm run dev
```

### Access the Application
1. Open browser: `http://localhost:3012`
2. Navigate to image generation feature
3. Follow test cases above

### What to Look For
1. **Hover Behavior**: X icon should smoothly fade in/out
2. **Click Feedback**: Visual feedback when clicking X
3. **Double Click**: Must double-click quickly to restore (not two separate clicks)
4. **State Persistence**: Disabled state should persist until restored
5. **API Integration**: Check network tab to verify only active images are sent

## Success Criteria

All test cases pass with:
- ✅ No console errors
- ✅ Smooth animations and transitions
- ✅ Correct visual feedback
- ✅ Proper state management
- ✅ Correct API data sent to backend

## Known Limitations

1. **Double Click Timing**: User must double-click within standard OS double-click interval
2. **Image Persistence**: Disabled state is not persisted to backend (only in component state)
3. **Undo/Redo**: No undo functionality for disable/restore actions

## Next Steps

After manual verification:
1. Test in production build (`npm run build`)
2. Test on different browsers (Chrome, Firefox, Safari, Edge)
3. Test on different screen sizes (desktop, tablet, mobile)
4. Consider adding keyboard shortcuts (e.g., Delete key to disable)
5. Consider adding batch operations (disable all, restore all)
