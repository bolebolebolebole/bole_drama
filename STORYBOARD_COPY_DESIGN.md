# 分镜复制功能需求分析

## 功能描述
将选中的分镜（镜头）复制并插入到下一个镜头位置。需要复制所有相关数据：
- 镜头属性（标题、类型、角度、运镜等）
- 镜头图片
- 视频生成记录
- 音效配乐
- 视频合成记录

---

## 数据结构分析

### 1. Storyboard模型（分镜表）

**文件**: `src/domain/models/drama.go`

```go
type Storyboard struct {
    ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
    EpisodeID       uint           `gorm:"not null;index:idx_storyboards_episode_id" json:"episode_id"`
    SceneID         *uint          `gorm:"index:idx_storyboards_scene_id;column:scene_id" json:"scene_id"`
    StoryboardNumber int            `gorm:"not null;column:storyboard_number" json:"storyboard_number"`
    Title            *string        `gorm:"size:255" json:"title"`
    Location         *string        `gorm:"size:255" json:"location"`
    Time             *string        `gorm:"size:255" json:"time"`
    ShotType         *string        `gorm:"size:100" json:"shot_type"`
    Angle            *string        `gorm:"size:100" json:"angle"`
    Movement         *string        `gorm:"size:100" json:"movement"`
    Action           *string        `gorm:"type:text" json:"action"`
    Result           *string        `gorm:"type:text" json:"result"`
    Atmosphere       *string        `gorm:"type:text" json:"atmosphere"`
    ImagePrompt      *string        `gorm:"type:text" json:"image_prompt"`
    VideoPrompt      *string        `gorm:"type:text" json:"video_prompt"`
    VideoPromptIsCustom bool        `gorm:"not null;default:false" json:"video_prompt_is_custom"`
    BgmPrompt        *string        `gorm:"type:text" json:"bgm_prompt"`
    SoundEffect      *string        `gorm:"size:255" json:"sound_effect"`
    Dialogue         *string        `gorm:"type:text" json:"dialogue"`
    Description      *string        `gorm:"type:text" json:"description"`
    Duration         int            `gorm:"default:5" json:"duration"`
    ComposedImage    *string        `gorm:"type:text" json:"composed_image"`
    VideoURL         *string        `gorm:"type:text" json:"video_url"`
    Status           string         `gorm:"type:varchar(20);default:'pending'" json:"status"`
    ReferenceOverrides datatypes.JSON `gorm:"type:json" json:"reference_overrides"`
    CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

    // 关联
    Episode    Episode     `gorm:"foreignKey:EpisodeID;constraint:OnDelete:CASCADE" json:"episode,omitempty"`
    Background *Scene      `gorm:"foreignKey:SceneID" json:"background,omitempty"`
    Characters []Character `gorm:"many2many:storyboard_characters;" json:"characters,omitempty"`
}
```

### 2. 关联数据表

#### ImageGeneration（图片生成记录）
- 与storyboard关联（1:N）
- 记录每次图片生成的详细信息

#### VideoGeneration（视频生成记录）
- 与storyboard关联（1:N）
- 记录每次视频生成的详细信息

#### VideoMerge（视频合成记录）
- 与storyboard关联（1:N）
- 记录视频合成任务

#### FramePrompt（帧提示词）
- 与storyboard关联（1:N）
- 记录不同帧类型的提示词

---

## 需要复制的数据清单

### 1. Storyboard主表字段 ✅
- [x] EpisodeID
- [x] SceneID
- [x] StoryboardNumber（需要调整顺序）
- [x] Title
- [x] Location
- [x] Time
- [x] ShotType
- [x] Angle
- [x] Movement
- [x] Action
- [x] Result
- [x] Atmosphere
- [x] ImagePrompt
- [x] VideoPrompt
- [x] VideoPromptIsCustom
- [x] BgmPrompt
- [x] SoundEffect
- [x] Dialogue
- [x] Description
- [x] Duration

### 2. 关联数据

#### Background（场景背景）
- [x] Scene数据（Location和Time）
- [ ] ImageURL（如果需要）

#### Characters（角色）
- [x] 所有Character ID的完整列表

### 3. 辅助数据

#### ReferenceOverrides
- [x] JSON格式的参考图替换配置

### 4. 不复制的字段

- [ ] ID（自动生成新的）
- [ ] CreatedAt/UpdatedAt（自动设置）
- [ ] Status（新记录默认为pending）
- [ ] ComposedImage（视频合成结果，不复制）
- [ ] VideoURL（视频生成结果，不复制）

---

## 复制逻辑设计

### 1. 查询源分镜数据

```go
// 从数据库完整加载源分镜及其所有关联数据
func (s *StoryboardService) GetStoryboardWithFullData(id uint) (*models.Storyboard, error) {
    var storyboard models.Storyboard
    err := s.db.Preload("Episode", "Background", "Characters").
        First(&storyboard, id).
        Error
    if err != nil {
        return nil, err
    }
    
    return &storyboard, nil
}
```

### 2. 创建新分镜（插入到下一个位置）

```go
func (s *StoryboardService) CopyStoryboard(sourceID uint, episodeID uint) (*models.Storyboard, error) {
    // 1. 查找源分镜
    var sourceSB models.Storyboard
    err := s.db.Preload("Episode", "Background", "Characters").
        First(&sourceSB, sourceID).Error
    if err != nil {
        return nil, err
    }
    
    // 2. 计算新分镜序号（插入到当前分镜之后）
    // 获取同episode的所有分镜，按storyboard_number排序
    var allStoryboards []models.Storyboard
    err = s.db.Where("episode_id = ? AND deleted_at IS NULL", episodeID).
        Order("storyboard_number ASC").
        Find(&allStoryboards).Error
    if err != nil {
        return nil, err
    }
    
    // 找到源分镜的位置
    sourceIndex := -1
    for i, sb := range allStoryboards {
        if sb.ID == sourceID {
            sourceIndex = i
            break
        }
    }
    
    if sourceIndex == -1 {
        return nil, fmt.Errorf("source storyboard not found")
    }
    
    // 新分镜序号 = 源分镜序号 + 1
    newStoryboardNumber := sourceSB.StoryboardNumber + 1
    
    // 3. 创建新分镜（复制源分镜数据，调整必要字段）
    newStoryboard := models.Storyboard{
        EpisodeID:       sourceSB.EpisodeID,
        SceneID:         sourceSB.SceneID,
        StoryboardNumber: newStoryboardNumber,
        Title:            sourceSB.Title,
        Location:         sourceSB.Location,
        Time:             sourceSB.Time,
        ShotType:         sourceSB.ShotType,
        Angle:            sourceSB.Angle,
        Movement:         sourceSB.Movement,
        Action:           sourceSB.Action,
        Result:           sourceSB.Result,
        Atmosphere:       sourceSB.Atmosphere,
        ImagePrompt:      sourceSB.ImagePrompt,
        VideoPrompt:      sourceSB.VideoPrompt,
        VideoPromptIsCustom: false, // 重置为自定义
        BgmPrompt:        sourceSB.BgmPrompt,
        SoundEffect:      sourceSB.SoundEffect,
        Dialogue:         sourceSB.Dialogue,
        Description:      sourceSB.Description,
        Duration:         sourceSB.Duration,
        Status:           "pending",
        ReferenceOverrides: sourceSB.ReferenceOverrides,
    }
    
    // 4. 保存到数据库
    if err := s.db.Create(&newStoryboard).Error; err != nil {
        return nil, fmt.Errorf("failed to create copied storyboard: %w", err)
    }
    
    // 5. 复制角色关联（如果源分镜有角色）
    if len(sourceSB.Characters) > 0 {
        var characterIDs []uint
        for _, char := range sourceSB.Characters {
            characterIDs = append(characterIDs, char.ID)
        }
        
        // 重新加载新分镜并关联角色
        if err := s.db.Model(&newStoryboard).Association("Characters").Append(characterIDs).Error; err != nil {
            return nil, fmt.Errorf("failed to copy character associations: %w", err)
        }
    }
    
    return &newStoryboard, nil
}
```

---

## API端点设计

### 后端API

**新增端点**: `POST /api/v1/storyboards/:id/copy`

**请求参数**:
```json
{
  "target_episode_id": 123  // 目标episode ID（可选，默认使用同episode）
  "insert_position": "after"  // 插入位置: "after" 或 "before"
}
```

**响应**:
```json
{
  "success": true,
  "data": {
    "id": 456,
    "storyboard_number": 5
  }
}
```

---

## 前端UI设计

### 分镜列表项UI增强

在 `ProfessionalEditor.vue` 的分镜列表项中添加：

1. **复制按钮**（悬停显示）
```vue
<div class="shot-actions" @mouseenter="showActions = true" @mouseleave="showActions = false">
  <el-dropdown trigger="click">
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item @click="handleCopyStoryboard(shot)">
          <el-icon><DocumentCopy /></el-icon>
          <span>复制分镜</span>
        </el-dropdown-item>
        <el-dropdown-item @click="handleDeleteStoryboard(shot.id)" v-if="canDelete">
          <el-icon><Delete /></el-icon>
          <span>删除</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </el-dropdown>
  </div>
</div>
```

2. **状态标记**
   - 显示复制的分镜有特殊标识（如"已复制"图标）
   - 高亮最新复制的分镜

### 复制逻辑

```javascript
const handleCopyStoryboard = async (shot) => {
  try {
    // 1. 调用后端复制API
    const result = await storyboardsAPI.copyStoryboard(shot.id)
    
    // 2. 刷新分镜列表
    await loadStoryboards()
    
    // 3. 滚动到新创建的分镜
    await nextTick(() => {
        const newShotElement = document.querySelector(`[data-id="${result.data.id}"]`)
        if (newShotElement) {
            newShotElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
            // 添加高亮动画
            newShotElement.classList.add('just-copied')
            setTimeout(() => {
                newShotElement.classList.remove('just-copped')
            }, 2000)
        }
    })
    
    ElMessage.success('分镜复制成功！')
  } catch (error) {
    ElMessage.error('复制失败：' + error.message)
  }
}
```

---

## 实现步骤

### Phase 1: 后端实现
1. ✅ 添加CopyStoryboard方法到storyboard_service.go
2. ✅ 添加GetStoryboardWithFullData方法
3. ✅ 添加CopyStoryboards路由和handler
4. ✅ 测试复制功能

### Phase 2: 前端实现
1. ✅ 添加复制按钮UI到分镜列表项
2. ✅ 添加handleCopyStoryboard函数
3. ✅ 添加API调用
4. ✅ 添加刷新和高亮动画
5. ✅ 测试完整流程

### Phase 3: 测试验证
1. ✅ 测试所有字段复制
2. ✅ 测试角色关联复制
3. ✅ 测试序号计算
4. ✅ 测试数据库约束

---

## 注意事项

1. **序号冲突**: 如果episode已有100个分镜，复制后会是101个，需要考虑上限
2. **性能优化**: 大量分镜时，角色关联查询可能较慢，考虑批量查询优化
3. **错误处理**: 如果源分镜不存在，返回404错误
4. **权限验证**: 检查用户是否有权限修改该episode的分镜
5. **事务处理**: 使用数据库事务确保数据一致性

---

## 测试用例

### 用例1: 复制简单分镜
- 创建episode和3个分镜
- 复制第2个分镜
- 验证：所有字段复制，序号变为4，角色关联正确

### 用例2: 复制带角色的分镜
- 添加3个角色
- 分镜2关联角色A和B
- 复制后分镜4应该也关联角色A和B

### 用例3: 复制分镜到episode末尾
- 分镜序号为3
- 复制后应该生成序号为4的新分镜

### 用例4: 复制包含复杂字段的分镜
- 包含自定义的ReferenceOverrides
- 包含所有提示词字段
- 验证所有内容正确复制

---

## 完成标准

- [ ] 后端CopyStoryboard方法实现
- [ ] 后端CopyStoryboards路由实现
- [ ] 前端copyStoryboardsAPI添加
- [ ] 前端复制按钮UI
- [ ] 前端handleCopyStoryboard函数
- [ ] 复制成功后自动滚动
- [ ] 所有字段复制测试通过
- [ ] 角色关联复制测试通过
- [ ] 序号计算正确测试通过
