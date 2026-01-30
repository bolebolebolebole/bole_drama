package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type StoryboardCompositionService struct {
	db       *gorm.DB
	log      *logger.Logger
	imageGen *ImageGenerationService
}

func NewStoryboardCompositionService(db *gorm.DB, log *logger.Logger, imageGen *ImageGenerationService) *StoryboardCompositionService {
	return &StoryboardCompositionService{
		db:       db,
		log:      log,
		imageGen: imageGen,
	}
}

type SceneCharacterInfo struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	ImageURL *string `json:"image_url,omitempty"`
}

type SceneBackgroundInfo struct {
	ID       uint    `json:"id"`
	Location string  `json:"location"`
	Time     string  `json:"time"`
	ImageURL *string `json:"image_url,omitempty"`
	Status   string  `json:"status"`
}

type SceneCompositionInfo struct {
	ID                    uint                 `json:"id"`
	StoryboardNumber      int                  `json:"storyboard_number"`
	Title                 *string              `json:"title"`
	Description           *string              `json:"description"`
	ShotType              *string              `json:"shot_type"`
	Angle                 *string              `json:"angle"`
	Movement              *string              `json:"movement"`
	Location              *string              `json:"location"`
	Time                  *string              `json:"time"`
	Duration              int                  `json:"duration"`
	Dialogue              *string              `json:"dialogue"`
	Action                *string              `json:"action"`
	Result                *string              `json:"result"`
	Atmosphere            *string              `json:"atmosphere"`
	BgmPrompt             *string              `json:"bgm_prompt,omitempty"`
	SoundEffect           *string              `json:"sound_effect,omitempty"`
	ImagePrompt           *string              `json:"image_prompt,omitempty"`
	VideoPrompt           *string              `json:"video_prompt,omitempty"`
	Characters            []SceneCharacterInfo `json:"characters"`
	Background            *SceneBackgroundInfo `json:"background"`
	SceneID               *uint                `json:"scene_id"`
	ComposedImage         *string              `json:"composed_image,omitempty"`
	VideoURL              *string              `json:"video_url,omitempty"`
	ImageGenerationID     *uint                `json:"image_generation_id,omitempty"`
	ImageGenerationStatus *string              `json:"image_generation_status,omitempty"`
	VideoGenerationID     *uint                `json:"video_generation_id,omitempty"`
	VideoGenerationStatus *string              `json:"video_generation_status,omitempty"`
	ReferenceOverrides    map[string]string    `json:"reference_overrides,omitempty"`
}

func (s *StoryboardCompositionService) GetScenesForEpisode(episodeID string) ([]SceneCompositionInfo, error) {
	// 验证权限
	var episode models.Episode
	err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error
	if err != nil {
		s.log.Errorw("Episode not found", "episode_id", episodeID, "error", err)
		return nil, fmt.Errorf("episode not found")
	}

	s.log.Infow("GetScenesForEpisode auth check",
		"episode_id", episodeID,
		"drama_id", episode.DramaID)

	// 获取分镜列表
	var storyboards []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).
		Preload("Characters").
		Order("storyboard_number ASC").
		Find(&storyboards).Error; err != nil {
		return nil, fmt.Errorf("failed to load storyboards: %w", err)
	}

	// 获取所有角色（用于匹配角色信息）
	var characters []models.Character
	if err := s.db.Where("drama_id = ?", episode.DramaID).Find(&characters).Error; err != nil {
		s.log.Warnw("Failed to load characters", "error", err)
	}

	// 创建角色ID到角色信息的映射
	charIDToInfo := make(map[uint]*models.Character)
	for i := range characters {
		charIDToInfo[characters[i].ID] = &characters[i]
	}

	// 角色图片兜底：
	// 1) 优先使用 characters.image_url
	// 2) 若为空，尝试从 image_generations（completed）里取最新一张
	// 3) 若仍为空，按角色别名（例如“遥遥”）在同剧本中找有图的角色替代显示
	charIDs := make([]uint, 0, len(characters))
	for i := range characters {
		charIDs = append(charIDs, characters[i].ID)
	}

	completedCharImageURL := make(map[uint]*string, len(characters))
	if len(charIDs) > 0 {
		var gens []models.ImageGeneration
		if err := s.db.
			Where("character_id IN ? AND status = ?", charIDs, models.ImageStatusCompleted).
			Order("created_at DESC").
			Find(&gens).Error; err != nil {
			s.log.Warnw("Failed to load completed character image generations", "error", err)
		} else {
			for i := range gens {
				g := &gens[i]
				if g.CharacterID == nil || g.ImageURL == nil {
					continue
				}
				if strings.TrimSpace(*g.ImageURL) == "" {
					continue
				}
				if _, ok := completedCharImageURL[*g.CharacterID]; !ok {
					completedCharImageURL[*g.CharacterID] = g.ImageURL
				}
			}
		}
	}

	aliasToBestImage := make(map[string]*string, len(characters)*2)
	for i := range characters {
		c := &characters[i]
		var url *string
		if c.ImageURL != nil && strings.TrimSpace(*c.ImageURL) != "" {
			url = c.ImageURL
		} else if u, ok := completedCharImageURL[c.ID]; ok {
			url = u
		}
		if url == nil {
			continue
		}
		for _, alias := range buildCharacterAliases(c.Name) {
			if alias == "" {
				continue
			}
			// Keep first non-empty mapping to reduce random overrides.
			if _, exists := aliasToBestImage[alias]; !exists {
				aliasToBestImage[alias] = url
			}
		}
	}

	// 获取所有场景ID
	var sceneIDs []uint
	for _, storyboard := range storyboards {
		if storyboard.SceneID != nil {
			sceneIDs = append(sceneIDs, *storyboard.SceneID)
		}
	}

	// 批量获取场景信息
	var scenes []models.Scene
	sceneMap := make(map[uint]*models.Scene)
	if len(sceneIDs) > 0 {
		if err := s.db.Where("id IN ?", sceneIDs).Find(&scenes).Error; err == nil {
			for i := range scenes {
				sceneMap[scenes[i].ID] = &scenes[i]
			}
		}
	}

	// 额外加载：同剧本下“已有图片”的场景，用于补全背景图（避免同地点重复场景导致素材缺失）
	var dramaScenesWithImages []models.Scene
	if err := s.db.
		Where("drama_id = ? AND image_url IS NOT NULL AND image_url <> ''", episode.DramaID).
		Find(&dramaScenesWithImages).Error; err != nil {
		s.log.Warnw("Failed to load drama scenes with images", "error", err)
	}
	sceneKeyToImageURL := make(map[string]*string, len(dramaScenesWithImages))
	for i := range dramaScenesWithImages {
		sc := &dramaScenesWithImages[i]
		if sc.ImageURL == nil || strings.TrimSpace(*sc.ImageURL) == "" {
			continue
		}
		key := normalizeSceneKey(sc.Location, sc.Time)
		if _, ok := sceneKeyToImageURL[key]; !ok {
			sceneKeyToImageURL[key] = sc.ImageURL
		}
	}

	// 获取分镜的合成图片（从 image_generations 表）
	storyboardIDs := make([]uint, len(storyboards))
	for i, storyboard := range storyboards {
		storyboardIDs[i] = storyboard.ID
	}

	imageGenMap := make(map[uint]string)                      // storyboard_id -> image_url
	imageGenTaskMap := make(map[uint]*models.ImageGeneration) // storyboard_id -> processing task
	if len(storyboardIDs) > 0 {
		var imageGens []models.ImageGeneration
		// 查询已完成的图片生成记录，每个镜头只取最新的一条
		if err := s.db.Where("storyboard_id IN ? AND status = ?", storyboardIDs, models.ImageStatusCompleted).
			Order("created_at DESC").
			Find(&imageGens).Error; err == nil {
			// 为每个镜头保留最新的一条记录
			for _, ig := range imageGens {
				if ig.StoryboardID != nil {
					if _, exists := imageGenMap[*ig.StoryboardID]; !exists {
						if ig.ImageURL != nil {
							imageGenMap[*ig.StoryboardID] = *ig.ImageURL
						}
					}
				}
			}
		}

		// 查询进行中的图片生成任务
		var processingImageGens []models.ImageGeneration
		if err := s.db.Where("storyboard_id IN ? AND status = ?", storyboardIDs, models.ImageStatusProcessing).
			Order("created_at DESC").
			Find(&processingImageGens).Error; err == nil {
			for _, ig := range processingImageGens {
				if ig.StoryboardID != nil {
					if _, exists := imageGenTaskMap[*ig.StoryboardID]; !exists {
						igCopy := ig
						imageGenTaskMap[*ig.StoryboardID] = &igCopy
					}
				}
			}
		}
	}

	// 批量查询进行中的视频生成任务
	videoGenTaskMap := make(map[uint]*models.VideoGeneration) // storyboard_id -> processing task
	if len(storyboardIDs) > 0 {
		var processingVideoGens []models.VideoGeneration
		if err := s.db.Where("scene_id IN ? AND status = ?", storyboardIDs, models.VideoStatusProcessing).
			Order("created_at DESC").
			Find(&processingVideoGens).Error; err == nil {
			for _, vg := range processingVideoGens {
				if vg.StoryboardID != nil {
					if _, exists := videoGenTaskMap[*vg.StoryboardID]; !exists {
						vgCopy := vg
						videoGenTaskMap[*vg.StoryboardID] = &vgCopy
					}
				}
			}
		}
	}

	// 构建返回结果
	var result []SceneCompositionInfo
	for _, storyboard := range storyboards {
		storyboardInfo := SceneCompositionInfo{
			ID:               storyboard.ID,
			StoryboardNumber: storyboard.StoryboardNumber,
			Title:            storyboard.Title,
			Description:      storyboard.Description,
			ShotType:         storyboard.ShotType,
			Angle:            storyboard.Angle,
			Movement:         storyboard.Movement,
			Location:         storyboard.Location,
			Time:             storyboard.Time,
			Duration:         storyboard.Duration,
			Action:           storyboard.Action,
			Dialogue:         storyboard.Dialogue,
			Result:           storyboard.Result,
			Atmosphere:       storyboard.Atmosphere,
			BgmPrompt:        storyboard.BgmPrompt,
			SoundEffect:      storyboard.SoundEffect,
			ImagePrompt:      storyboard.ImagePrompt,
			VideoPrompt:      storyboard.VideoPrompt,
			SceneID:          storyboard.SceneID,
		}

		// 直接使用关联的角色信息
		if len(storyboard.Characters) > 0 {
			for _, char := range storyboard.Characters {
				img := char.ImageURL
				if (img == nil || strings.TrimSpace(*img) == "") && completedCharImageURL[char.ID] != nil {
					img = completedCharImageURL[char.ID]
				}
				if img == nil || strings.TrimSpace(*img) == "" {
					for _, alias := range buildCharacterAliases(char.Name) {
						if u, ok := aliasToBestImage[alias]; ok {
							img = u
							break
						}
					}
				}
				storyboardChar := SceneCharacterInfo{
					ID:       char.ID,
					Name:     char.Name,
					ImageURL: img,
				}
				storyboardInfo.Characters = append(storyboardInfo.Characters, storyboardChar)
			}
		}

		// 添加场景信息
		if storyboard.SceneID != nil {
			if scene, ok := sceneMap[*storyboard.SceneID]; ok {
				bg := &SceneBackgroundInfo{
					ID:       scene.ID,
					Location: scene.Location,
					Time:     scene.Time,
					ImageURL: scene.ImageURL,
					Status:   scene.Status,
				}
				// 如果该场景本身没有 image_url，则尝试使用同地点同时间的“已生成场景图”作为背景图
				if (bg.ImageURL == nil || strings.TrimSpace(*bg.ImageURL) == "") && bg.Location != "" {
					key := normalizeSceneKey(bg.Location, bg.Time)
					if url, ok := sceneKeyToImageURL[key]; ok {
						bg.ImageURL = url
					} else {
						// Fallback: loose match by location/time to any scene with images
						bestScore := 0
						var bestURL *string
						for i := range dramaScenesWithImages {
							sc := &dramaScenesWithImages[i]
							score := sceneMatchScore(bg.Location, bg.Time, sc.Location, sc.Time)
							if score > bestScore {
								bestScore = score
								bestURL = sc.ImageURL
							}
						}
						if bestURL != nil {
							bg.ImageURL = bestURL
						}
					}
				}
				storyboardInfo.Background = bg
			}
		} else {
			// SceneID 为空时，尝试根据分镜 location/time 推断一个背景（仅用于编辑器显示与选图，不改变原分镜数据）
			if storyboard.Location != nil {
				sbLoc := strings.TrimSpace(*storyboard.Location)
				sbTime := ""
				if storyboard.Time != nil {
					sbTime = strings.TrimSpace(*storyboard.Time)
				}
				bestScore := 0
				var bestScene *models.Scene
				for i := range dramaScenesWithImages {
					sc := &dramaScenesWithImages[i]
					score := sceneMatchScore(sbLoc, sbTime, sc.Location, sc.Time)
					if score > bestScore {
						bestScore = score
						bestScene = sc
					}
				}
				if bestScene != nil {
					storyboardInfo.SceneID = &bestScene.ID
					storyboardInfo.Background = &SceneBackgroundInfo{
						ID:       bestScene.ID,
						Location: bestScene.Location,
						Time:     bestScene.Time,
						ImageURL: bestScene.ImageURL,
						Status:   bestScene.Status,
					}
				}
			}
		}

		// 如果镜头没有角色关联，尝试从文本中推断（用于编辑器展示/参考图）
		if len(storyboardInfo.Characters) == 0 {
			aliasToID := make(map[string]uint, len(charIDToInfo)*2)
			for id, c := range charIDToInfo {
				for _, alias := range buildCharacterAliases(c.Name) {
					aliasToID[alias] = id
				}
			}

			var merged []uint
			if storyboard.Action != nil {
				merged = append(merged, inferCharacterIDsFromText(*storyboard.Action, aliasToID)...)
			}
			if storyboard.Dialogue != nil {
				merged = append(merged, inferCharacterIDsFromText(*storyboard.Dialogue, aliasToID)...)
			}
			if storyboard.Description != nil {
				merged = append(merged, inferCharacterIDsFromText(*storyboard.Description, aliasToID)...)
			}
			// Dedup and materialize
			seen := make(map[uint]struct{}, len(merged))
			for _, id := range merged {
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				if c, ok := charIDToInfo[id]; ok {
					storyboardInfo.Characters = append(storyboardInfo.Characters, SceneCharacterInfo{
						ID:       c.ID,
						Name:     c.Name,
						ImageURL: c.ImageURL,
					})
				}
			}
		}

		// 添加合成图片
		if imageURL, ok := imageGenMap[storyboard.ID]; ok {
			storyboardInfo.ComposedImage = &imageURL
		}

		// 添加视频URL
		if storyboard.VideoURL != nil {
			storyboardInfo.VideoURL = storyboard.VideoURL
		}

		// 添加进行中的图片生成任务信息
		if imageTask, ok := imageGenTaskMap[storyboard.ID]; ok {
			storyboardInfo.ImageGenerationID = &imageTask.ID
			statusStr := string(imageTask.Status)
			storyboardInfo.ImageGenerationStatus = &statusStr
		}

		// 添加进行中的视频生成任务信息
		if videoTask, ok := videoGenTaskMap[storyboard.ID]; ok {
			storyboardInfo.VideoGenerationID = &videoTask.ID
			statusStr := string(videoTask.Status)
			storyboardInfo.VideoGenerationStatus = &statusStr
		}

		// 添加参考图替换
		if storyboard.ReferenceOverrides != nil {
			var overrides map[string]string
			if err := json.Unmarshal(storyboard.ReferenceOverrides, &overrides); err == nil && len(overrides) > 0 {
				storyboardInfo.ReferenceOverrides = overrides
			}
		}

		result = append(result, storyboardInfo)
	}

	return result, nil
}

type UpdateSceneRequest struct {
	SceneID     *uint   `json:"scene_id"`
	Characters  []uint  `json:"characters"` // 改为存储角色ID数组
	Location    *string `json:"location"`
	Time        *string `json:"time"`
	Action      *string `json:"action"`
	Dialogue    *string `json:"dialogue"`
	Description *string `json:"description"`
	Duration    *int    `json:"duration"`
	ImagePrompt *string `json:"image_prompt"`
	VideoPrompt *string `json:"video_prompt"`
}

func (s *StoryboardCompositionService) UpdateScene(sceneID string, req *UpdateSceneRequest) error {
	// 获取分镜并验证权限
	var storyboard models.Storyboard
	err := s.db.Preload("Episode.Drama").Where("id = ?", sceneID).First(&storyboard).Error
	if err != nil {
		return fmt.Errorf("scene not found")
	}

	// 构建更新数据
	updates := make(map[string]interface{})

	// 更新背景ID
	if req.SceneID != nil {
		updates["scene_id"] = req.SceneID
	}

	// 更新角色列表（直接存储ID数组）
	if req.Characters != nil {
		charactersJSON, err := json.Marshal(req.Characters)
		if err != nil {
			return fmt.Errorf("failed to serialize characters: %w", err)
		}
		updates["characters"] = charactersJSON
	}

	// 更新场景信息字段
	if req.Location != nil {
		updates["location"] = req.Location
	}
	if req.Time != nil {
		updates["time"] = req.Time
	}
	if req.Action != nil {
		updates["action"] = req.Action
	}
	if req.Dialogue != nil {
		updates["dialogue"] = req.Dialogue
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.Duration != nil {
		updates["duration"] = *req.Duration
	}
	if req.ImagePrompt != nil {
		updates["image_prompt"] = req.ImagePrompt
	}
	if req.VideoPrompt != nil {
		updates["video_prompt"] = req.VideoPrompt
	}

	// 执行更新
	if len(updates) > 0 {
		if err := s.db.Model(&models.Storyboard{}).Where("id = ?", sceneID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update scene: %w", err)
		}
	}

	s.log.Infow("Scene updated", "scene_id", sceneID, "updates", updates)
	return nil
}

type GenerateSceneImageRequest struct {
	SceneID uint   `json:"scene_id"`
	Prompt  string `json:"prompt"`
	Model   string `json:"model"`
}

func (s *StoryboardCompositionService) GenerateSceneImage(req *GenerateSceneImageRequest) (*models.ImageGeneration, error) {
	// 获取场景并验证权限
	var scene models.Scene
	err := s.db.Where("id = ?", req.SceneID).First(&scene).Error
	if err != nil {
		return nil, fmt.Errorf("scene not found")
	}

	// 验证权限：通过DramaID查询Drama
	var drama models.Drama
	if err := s.db.Where("id = ? ", scene.DramaID).First(&drama).Error; err != nil {
		return nil, fmt.Errorf("unauthorized")
	}

	// 构建场景图片生成提示词
	prompt := req.Prompt
	if prompt == "" {
		// 使用场景的Prompt字段
		prompt = scene.Prompt
		if prompt == "" {
			// 如果Prompt为空，使用Location和Time构建
			prompt = fmt.Sprintf("%s场景，%s", scene.Location, scene.Time)
		}
		s.log.Infow("Using scene prompt", "scene_id", req.SceneID, "prompt", prompt)
	}

	// 对齐角色图片逻辑：生成场景图也要参考项目风格
	styleDesc := "写实风格"
	if strings.TrimSpace(drama.Style) == "anime" {
		styleDesc = "动漫风格"
	}
	trimmed := strings.TrimSpace(prompt)
	if styleDesc != "" {
		// 若提示词已包含另一种风格前缀，优先纠正为项目风格
		if strings.HasPrefix(trimmed, "动漫风格") && styleDesc == "写实风格" {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "动漫风格"))
			trimmed = strings.TrimLeft(trimmed, "，,")
			trimmed = styleDesc + "，" + strings.TrimSpace(trimmed)
		} else if strings.HasPrefix(trimmed, "写实风格") && styleDesc == "动漫风格" {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "写实风格"))
			trimmed = strings.TrimLeft(trimmed, "，,")
			trimmed = styleDesc + "，" + strings.TrimSpace(trimmed)
		} else if !strings.HasPrefix(trimmed, styleDesc) {
			trimmed = styleDesc + "，" + trimmed
		}
		prompt = trimmed
	}

	// 使用imageGen服务直接生成
	if s.imageGen != nil {
		// 使用火山引擎API（豆包）
		provider := "volcengine"
		modelName := req.Model
		if modelName == "" {
			modelName = "doubao-seedream-4-5-251128"
		}

		genReq := &GenerateImageRequest{
			SceneID:   &req.SceneID,
			DramaID:   fmt.Sprintf("%d", scene.DramaID),
			ImageType: string(models.ImageTypeScene),
			Prompt:    prompt,
			Provider:  provider,
			Model:     modelName,
			Size:      "2560x1440",
			Quality:   "standard",
		}
		imageGen, err := s.imageGen.GenerateImage(genReq)
		if err != nil {
			return nil, fmt.Errorf("failed to generate image: %w", err)
		}

		// 异步等待生成完成后，回写 scenes.image_url，确保刷新后仍能看到新图
		go s.waitAndUpdateSceneImage(req.SceneID, imageGen.ID)

		s.log.Infow("Scene image generation created", "scene_id", req.SceneID, "image_gen_id", imageGen.ID)
		return imageGen, nil
	}

	return nil, fmt.Errorf("image generation service not available")
}

func (s *StoryboardCompositionService) waitAndUpdateSceneImage(sceneID uint, imageGenID uint) {
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		var imageGen models.ImageGeneration
		if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
			s.log.Errorw("Failed to query image generation status", "error", err, "image_gen_id", imageGenID)
			continue
		}

		if imageGen.Status == models.ImageStatusCompleted && imageGen.ImageURL != nil && *imageGen.ImageURL != "" {
			updates := map[string]interface{}{
				"image_url": *imageGen.ImageURL,
				"status":    "generated",
			}
			if err := s.db.Model(&models.Scene{}).Where("id = ?", sceneID).Updates(updates).Error; err != nil {
				s.log.Errorw("Failed to update scene image_url", "error", err, "scene_id", sceneID)
				return
			}
			s.log.Infow("Scene image updated successfully", "scene_id", sceneID, "image_url", *imageGen.ImageURL)
			return
		}

		if imageGen.Status == models.ImageStatusFailed {
			s.log.Errorw("Scene image generation failed", "scene_id", sceneID, "image_gen_id", imageGenID, "error", imageGen.ErrorMsg)
			_ = s.db.Model(&models.Scene{}).Where("id = ?", sceneID).Update("status", "failed").Error
			return
		}
	}

	s.log.Warnw("Scene image generation timeout", "scene_id", sceneID, "image_gen_id", imageGenID)
}

func (s *StoryboardCompositionService) DeleteScene(sceneID string) error {
	var scene models.Scene
	if err := s.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("scene not found")
		}
		return fmt.Errorf("failed to find scene: %w", err)
	}

	// 删除场景
	if err := s.db.Delete(&scene).Error; err != nil {
		return fmt.Errorf("failed to delete scene: %w", err)
	}

	s.log.Infow("Scene deleted successfully", "scene_id", sceneID)
	return nil
}

func getStringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
