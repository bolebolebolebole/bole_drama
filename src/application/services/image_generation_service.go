package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/image"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type ImageGenerationService struct {
	db              *gorm.DB
	aiService       *AIService
	transferService *ResourceTransferService
	localStorage    *storage.LocalStorage
	log             *logger.Logger
	config          *config.Config
	promptI18n      *PromptI18n
}

// truncateImageURL 截断图片 URL，避免 base64 格式的 URL 占满日志
func truncateImageURL(url string) string {
	if url == "" {
		return ""
	}
	// 如果是 data URI 格式（base64），只显示前缀
	if strings.HasPrefix(url, "data:") {
		if len(url) > 50 {
			return url[:50] + "...[base64 data]"
		}
	}
	// 普通 URL 如果过长也截断
	if len(url) > 100 {
		return url[:100] + "..."
	}
	return url
}

func NewImageGenerationService(db *gorm.DB, cfg *config.Config, transferService *ResourceTransferService, localStorage *storage.LocalStorage, log *logger.Logger) *ImageGenerationService {
	return &ImageGenerationService{
		db:              db,
		aiService:       NewAIService(db, log),
		transferService: transferService,
		localStorage:    localStorage,
		config:          cfg,
		promptI18n:      NewPromptI18n(cfg),
		log:             log,
	}
}

// GetDB 获取数据库连接
func (s *ImageGenerationService) GetDB() *gorm.DB {
	return s.db
}

type singleScenePromptResult struct {
	Prompt string `json:"prompt"`
}

// ReextractScenePrompt 重新从剧本中为指定场景生成 prompt 字段（仅更新该场景，不影响其他场景）
func (s *ImageGenerationService) ReextractScenePrompt(sceneID uint, scriptContent string, model string) (*models.Scene, error) {
	var scene models.Scene
	if err := s.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		return nil, fmt.Errorf("scene not found")
	}
	if scriptContent == "" {
		return nil, fmt.Errorf("script content is empty")
	}

	// 获取AI客户端（如果指定了模型则使用指定的模型）
	var client ai.AIClient
	var err error
	if model != "" {
		s.log.Infow("Using specified model for single scene re-extraction", "model", model, "scene_id", sceneID)
		client, err = s.aiService.GetAIClientForModel("text", model)
		if err != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", err)
			client, err = s.aiService.GetAIClient("text")
		}
	} else {
		client, err = s.aiService.GetAIClient("text")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get AI client: %w", err)
	}

	systemPrompt := `你是一个专业的提示词工程师，擅长为AI图像生成模型撰写高质量的纯中文背景场景提示词。

你的任务是：只为指定的场景生成一个适合生成“可复用场景背景图/场景库”的prompt。

【核心原则：prompt必须严格静态化】
只描述稳定的背景环境，不得包含任何剧情镜头内容。

要求：
1. 只返回一个JSON对象，不要包含markdown代码块或任何解释文字。
2. JSON结构必须是：{"prompt":"..."}
3. prompt必须是纯中文，不得包含任何英文单词、字母或英文标点。
4. prompt必须描述“空场景/纯背景”，必须明确写出“无人物，无角色，空场景”。
5. prompt要包含环境细节、建筑/物品、光照、色调、氛围（仅作为环境形容词），细节丰富，高质量。

严禁：人物/角色/人群、动作/事件/剧情进展、情绪/心理/状态判断、他人名字/互动、镜头语言/摄影术语、拟人化叙事（如“仿佛在等待”）。`

	userPrompt := fmt.Sprintf("【地点】%s\n【时间】%s\n\n【剧本内容】\n%s\n\n请输出JSON：{\"prompt\":\"...\"}\n提示：prompt需以\"动漫风格纯背景场景\"开头，并明确“无人物，无角色，空场景”。不要写任何人物、动作、剧情、镜头术语或拟人化叙事。", scene.Location, scene.Time, scriptContent)

	text, err := client.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(0.4))
	if err != nil {
		s.log.Errorw("Failed to reextract scene prompt", "error", err, "scene_id", sceneID)
		return nil, fmt.Errorf("重新提取场景提示词失败: %w", err)
	}

	var result singleScenePromptResult
	if err := utils.SafeParseAIJSON(text, &result); err != nil {
		s.log.Errorw("Failed to parse scene prompt JSON", "error", err, "scene_id", sceneID, "raw_response", text[:min(len(text), 500)])
		return nil, fmt.Errorf("解析AI返回结果失败: %w", err)
	}
	if result.Prompt == "" {
		return nil, fmt.Errorf("AI返回的prompt为空")
	}

	if err := s.db.Model(&scene).Update("prompt", result.Prompt).Error; err != nil {
		s.log.Errorw("Failed to update scene prompt", "error", err, "scene_id", sceneID)
		return nil, fmt.Errorf("更新场景失败: %w", err)
	}

	if err := s.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		return nil, err
	}
	return &scene, nil
}

type GenerateImageRequest struct {
	StoryboardID    *uint    `json:"storyboard_id"`
	DramaID         string   `json:"drama_id" binding:"required"`
	SceneID         *uint    `json:"scene_id"`
	CharacterID     *uint    `json:"character_id"`
	ImageType       string   `json:"image_type"` // character, scene, storyboard
	FrameType       *string  `json:"frame_type"` // first, key, last, panel, action
	Prompt          string   `json:"prompt" binding:"required,min=5,max=2000"`
	NegativePrompt  *string  `json:"negative_prompt"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	Size            string   `json:"size"`
	Quality         string   `json:"quality"`
	Style           *string  `json:"style"`
	Steps           *int     `json:"steps"`
	CfgScale        *float64 `json:"cfg_scale"`
	Seed            *int64   `json:"seed"`
	Width           *int     `json:"width"`
	Height          *int     `json:"height"`
	ReferenceImages []string `json:"reference_images"` // 参考图片URL列表
}

func (s *ImageGenerationService) GenerateImage(request *GenerateImageRequest) (*models.ImageGeneration, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", request.DramaID).First(&drama).Error; err != nil {
		return nil, fmt.Errorf("drama not found")
	}

	// 注意：SceneID可能指向Scene或Storyboard表，调用方已经做过权限验证，这里不再重复验证

	provider := request.Provider
	if provider == "" {
		provider = "volcengine"
	}

	// 序列化参考图片
	var referenceImagesJSON []byte
	if len(request.ReferenceImages) > 0 {
		referenceImagesJSON, _ = json.Marshal(request.ReferenceImages)
	}

	// 转换DramaID
	dramaIDParsed, err := strconv.ParseUint(request.DramaID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid drama ID")
	}

	// 设置默认图片类型
	imageType := request.ImageType
	if imageType == "" {
		imageType = string(models.ImageTypeStoryboard)
	}

	// 根据图片类型强制设置固定长宽比（超高清尺寸）
	var width *int
	var height *int
	var size string

	switch imageType {
	case string(models.ImageTypeCharacter):
		// 角色图片：强制 3:4 长宽比，且满足下游模型最小像素要求（>= 3,686,400）
		// 1664x2240 = 3,727,360 pixels
		characterWidth := 1664
		characterHeight := 2240
		width = &characterWidth
		height = &characterHeight
		size = "1664x2240"
		s.log.Infow("Enforced character image dimensions", "width", *width, "height", *height, "ratio", "3:4")
	case string(models.ImageTypeScene):
		// 场景图片：强制 1:1 长宽比 (2048x2048)
		sceneWidth := 2048
		sceneHeight := 2048
		width = &sceneWidth
		height = &sceneHeight
		size = "2048x2048"
		s.log.Infow("Enforced scene image dimensions", "width", *width, "height", *height, "ratio", "1:1")
	case string(models.ImageTypeStoryboard):
		// 分镜图片：按项目全局画幅配置强制 16:9 / 9:16
		aspectRatio := strings.TrimSpace(drama.AspectRatio)
		if aspectRatio == "" {
			aspectRatio = "9:16"
		}

		switch aspectRatio {
		case "16:9":
			storyboardWidth := 2560
			storyboardHeight := 1440
			width = &storyboardWidth
			height = &storyboardHeight
			size = "2560x1440"
		case "9:16":
			storyboardWidth := 1440
			storyboardHeight := 2560
			width = &storyboardWidth
			height = &storyboardHeight
			size = "1440x2560"
		default:
			// 容错：不识别则回退 9:16
			storyboardWidth := 1440
			storyboardHeight := 2560
			width = &storyboardWidth
			height = &storyboardHeight
			size = "1440x2560"
			aspectRatio = "9:16"
		}
		s.log.Infow("Enforced storyboard image dimensions", "width", *width, "height", *height, "ratio", aspectRatio)
	default:
		// 分镜图片保持原有逻辑，允许用户指定尺寸
		width = request.Width
		height = request.Height
		size = request.Size
		if size == "" && (width == nil || height == nil) {
			size = request.Size
		}
	}

	imageGen := &models.ImageGeneration{
		StoryboardID:    request.StoryboardID,
		DramaID:         uint(dramaIDParsed),
		SceneID:         request.SceneID,
		CharacterID:     request.CharacterID,
		ImageType:       imageType,
		FrameType:       request.FrameType,
		Provider:        provider,
		Prompt:          request.Prompt,
		NegPrompt:       request.NegativePrompt,
		Model:           request.Model,
		Size:            size,
		ReferenceImages: referenceImagesJSON,
		Quality:         request.Quality,
		Style:           request.Style,
		Steps:           request.Steps,
		CfgScale:        request.CfgScale,
		Seed:            request.Seed,
		Width:           width,
		Height:          height,
		Status:          models.ImageStatusPending,
	}

	if err := s.db.Create(imageGen).Error; err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	go s.ProcessImageGeneration(imageGen.ID)

	return imageGen, nil
}

func (s *ImageGenerationService) ProcessImageGeneration(imageGenID uint) {
	var imageGen models.ImageGeneration
	if err := s.db.First(&imageGen, imageGenID).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	s.db.Model(&imageGen).Update("status", models.ImageStatusProcessing)

	// 如果关联了background，同步更新background为generating状态
	if imageGen.StoryboardID != nil {
		if err := s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.StoryboardID).Update("status", "generating").Error; err != nil {
			s.log.Warnw("Failed to update background status to generating", "scene_id", *imageGen.StoryboardID, "error", err)
		} else {
			s.log.Infow("Background status updated to generating", "scene_id", *imageGen.StoryboardID)
		}
	}

	client, err := s.getImageClientWithModel(imageGen.Provider, imageGen.Model)
	if err != nil {
		s.log.Errorw("Failed to get image client", "error", err, "provider", imageGen.Provider, "model", imageGen.Model)
		s.updateImageGenError(imageGenID, err.Error())
		return
	}

	// 解析参考图片
	var referenceImages []string
	if len(imageGen.ReferenceImages) > 0 {
		if err := json.Unmarshal(imageGen.ReferenceImages, &referenceImages); err == nil {
			s.log.Infow("Using reference images for generation",
				"id", imageGenID,
				"reference_count", len(referenceImages),
				"references", referenceImages)
		}
	}

	s.log.Infow("Starting image generation", "id", imageGenID, "prompt", imageGen.Prompt, "provider", imageGen.Provider)

	var opts []image.ImageOption
	if imageGen.NegPrompt != nil && *imageGen.NegPrompt != "" {
		opts = append(opts, image.WithNegativePrompt(*imageGen.NegPrompt))
	}

	// 处理图片尺寸参数：优先使用width/height，如果不存在则从size字符串解析
	s.log.Infow("Processing image size parameters",
		"width", imageGen.Width,
		"height", imageGen.Height,
		"size", imageGen.Size)

	if imageGen.Width != nil && imageGen.Height != nil {
		// 如果有明确的width和height，使用它们
		opts = append(opts, image.WithDimensions(*imageGen.Width, *imageGen.Height))
		opts = append(opts, image.WithSize(fmt.Sprintf("%dx%d", *imageGen.Width, *imageGen.Height)))
	} else if imageGen.Size != "" && strings.Contains(imageGen.Size, "x") {
		// 如果没有明确的width和height，但size字符串包含"x"，则解析size
		parts := strings.Split(imageGen.Size, "x")
		if len(parts) == 2 {
			w, errW := strconv.Atoi(parts[0])
			h, errH := strconv.Atoi(parts[1])
			if errW == nil && errH == nil {
				// 成功解析，添加width和height选项
				opts = append(opts, image.WithDimensions(w, h))
				opts = append(opts, image.WithSize(imageGen.Size))
			} else {
				// 解析失败，仍然传递size字符串
				opts = append(opts, image.WithSize(imageGen.Size))
			}
		} else {
			// size格式不正确，仍然传递size字符串
			opts = append(opts, image.WithSize(imageGen.Size))
		}
	} else if imageGen.Size != "" {
		// size字符串不包含"x"，直接传递
		opts = append(opts, image.WithSize(imageGen.Size))
	}

	if imageGen.Quality != "" {
		opts = append(opts, image.WithQuality(imageGen.Quality))
	}
	if imageGen.Style != nil && *imageGen.Style != "" {
		opts = append(opts, image.WithStyle(*imageGen.Style))
	}
	if imageGen.Steps != nil {
		opts = append(opts, image.WithSteps(*imageGen.Steps))
	}
	if imageGen.CfgScale != nil {
		opts = append(opts, image.WithCfgScale(*imageGen.CfgScale))
	}
	if imageGen.Seed != nil {
		opts = append(opts, image.WithSeed(*imageGen.Seed))
	}
	if imageGen.Model != "" {
		opts = append(opts, image.WithModel(imageGen.Model))
	}
	// 添加参考图片
	if len(referenceImages) > 0 {
		opts = append(opts, image.WithReferenceImages(referenceImages))
	}

	result, err := client.GenerateImage(imageGen.Prompt, opts...)
	if err != nil {
		s.log.Errorw("Image generation API call failed", "error", err, "id", imageGenID, "prompt", imageGen.Prompt)
		s.updateImageGenError(imageGenID, err.Error())
		return
	}

	s.log.Infow("Image generation API call completed", "id", imageGenID, "completed", result.Completed, "has_url", result.ImageURL != "")

	if !result.Completed {
		s.db.Model(&imageGen).Updates(map[string]interface{}{
			"status":  models.ImageStatusProcessing,
			"task_id": result.TaskID,
		})
		go s.pollTaskStatus(imageGenID, client, result.TaskID)
		return
	}

	s.completeImageGeneration(imageGenID, result)
}

func (s *ImageGenerationService) pollTaskStatus(imageGenID uint, client image.ImageClient, taskID string) {
	maxAttempts := 60
	pollInterval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(pollInterval)

		result, err := client.GetTaskStatus(taskID)
		if err != nil {
			s.log.Errorw("Failed to get task status", "error", err, "task_id", taskID)
			continue
		}

		if result.Completed {
			s.completeImageGeneration(imageGenID, result)
			return
		}

		if result.Error != "" {
			s.updateImageGenError(imageGenID, result.Error)
			return
		}
	}

	s.updateImageGenError(imageGenID, "timeout: image generation took too long")
}

func (s *ImageGenerationService) completeImageGeneration(imageGenID uint, result *image.ImageResult) {
	now := time.Now()

	finalImageURL := result.ImageURL
	originalImageURL := result.ImageURL

	// 将远程临时URL（如TOS签名URL）持久化到本地存储，保证后续可用。
	// 仅下载 HTTP/HTTPS URL，跳过 data URI。
	if s.localStorage != nil && result.ImageURL != "" &&
		(strings.HasPrefix(result.ImageURL, "http://") || strings.HasPrefix(result.ImageURL, "https://")) &&
		!s.localStorage.IsLocalURL(result.ImageURL) {
		localURL, err := s.localStorage.DownloadFromURL(result.ImageURL, "images")
		if err != nil {
			errStr := err.Error()
			if len(errStr) > 200 {
				errStr = errStr[:200] + "..."
			}
			s.log.Warnw("Failed to persist image to local storage",
				"error", errStr,
				"id", imageGenID,
				"original_url", truncateImageURL(result.ImageURL))
		} else {
			finalImageURL = localURL
			s.log.Infow("Image persisted to local storage",
				"id", imageGenID,
				"original_url", truncateImageURL(result.ImageURL),
				"local_url", truncateImageURL(localURL))
		}
	}

	updates := map[string]interface{}{
		"status":       models.ImageStatusCompleted,
		"image_url":    finalImageURL,
		"completed_at": now,
	}
	if originalImageURL != "" && finalImageURL != "" && originalImageURL != finalImageURL {
		updates["minio_url"] = originalImageURL
	}

	if result.Width > 0 {
		updates["width"] = result.Width
	}
	if result.Height > 0 {
		updates["height"] = result.Height
	}

	// 更新image_generation记录
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ?", imageGenID).First(&imageGen).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	s.db.Model(&models.ImageGeneration{}).Where("id = ?", imageGenID).Updates(updates)
	s.log.Infow("Image generation completed", "id", imageGenID)

	// 如果关联了storyboard，同步更新storyboard的composed_image
	if imageGen.StoryboardID != nil {
		if err := s.db.Model(&models.Storyboard{}).Where("id = ?", *imageGen.StoryboardID).Update("composed_image", finalImageURL).Error; err != nil {
			s.log.Errorw("Failed to update storyboard composed_image", "error", err, "storyboard_id", *imageGen.StoryboardID)
		} else {
			s.log.Infow("Storyboard updated with composed image",
				"storyboard_id", *imageGen.StoryboardID,
				"composed_image", truncateImageURL(finalImageURL))
		}
	}

	// 如果关联了scene，同步更新scene的image_url和status（仅当ImageType是scene时）
	if imageGen.SceneID != nil && imageGen.ImageType == string(models.ImageTypeScene) {
		sceneUpdates := map[string]interface{}{
			"status":    "generated",
			"image_url": finalImageURL,
		}
		if err := s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.SceneID).Updates(sceneUpdates).Error; err != nil {
			s.log.Errorw("Failed to update scene", "error", err, "scene_id", *imageGen.SceneID)
		} else {
			s.log.Infow("Scene updated with generated image",
				"scene_id", *imageGen.SceneID,
				"image_url", truncateImageURL(finalImageURL))
		}
	}

	// 如果关联了角色，同步更新角色的image_url
	if imageGen.CharacterID != nil {
		if err := s.db.Model(&models.Character{}).Where("id = ?", *imageGen.CharacterID).Update("image_url", finalImageURL).Error; err != nil {
			s.log.Errorw("Failed to update character image_url", "error", err, "character_id", *imageGen.CharacterID)
		} else {
			s.log.Infow("Character updated with generated image",
				"character_id", *imageGen.CharacterID,
				"image_url", truncateImageURL(finalImageURL))
		}
	}
}

func (s *ImageGenerationService) updateImageGenError(imageGenID uint, errorMsg string) {
	// 先获取image_generation记录
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ?", imageGenID).First(&imageGen).Error; err != nil {
		s.log.Errorw("Failed to load image generation", "error", err, "id", imageGenID)
		return
	}

	// 更新image_generation状态
	s.db.Model(&models.ImageGeneration{}).Where("id = ?", imageGenID).Updates(map[string]interface{}{
		"status":    models.ImageStatusFailed,
		"error_msg": errorMsg,
	})
	s.log.Errorw("Image generation failed", "id", imageGenID, "error", errorMsg)

	// 如果关联了scene，同步更新scene为失败状态
	if imageGen.SceneID != nil {
		s.db.Model(&models.Scene{}).Where("id = ?", *imageGen.SceneID).Update("status", "failed")
		s.log.Warnw("Scene marked as failed", "scene_id", *imageGen.SceneID)
	}
}

func (s *ImageGenerationService) getImageClient(provider string) (image.ImageClient, error) {
	config, err := s.aiService.GetDefaultConfig("image")
	if err != nil {
		return nil, fmt.Errorf("no image AI config found: %w", err)
	}

	// 使用第一个模型
	model := ""
	if len(config.Model) > 0 {
		model = config.Model[0]
	}

	// 使用配置中的 provider，如果没有则使用传入的 provider
	actualProvider := config.Provider
	if actualProvider == "" {
		actualProvider = provider
	}

	// 根据 provider 自动设置默认端点
	var endpoint string
	var queryEndpoint string

	switch actualProvider {
	case "openai", "dalle":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "chatfire":
		if strings.Contains(strings.ToLower(model), "doubao") || strings.Contains(strings.ToLower(model), "seedream") {
			endpoint = "/images/generations"
			queryEndpoint = ""
			return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
		}
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "volcengine", "volces", "doubao":
		endpoint = "/images/generations"
		queryEndpoint = ""
		return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	case "gemini", "google":
		endpoint = "/v1beta/models/{model}:generateContent"
		return image.NewGeminiImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	default:
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	}
}

// getImageClientWithModel 根据模型名称获取图片客户端
func (s *ImageGenerationService) getImageClientWithModel(provider string, modelName string) (image.ImageClient, error) {
	var config *models.AIServiceConfig
	var err error

	// 如果指定了模型，尝试获取对应的配置
	if modelName != "" {
		config, err = s.aiService.GetConfigForModel("image", modelName)
		if err != nil {
			s.log.Warnw("Failed to get config for model, using default", "model", modelName, "error", err)
			config, err = s.aiService.GetDefaultConfig("image")
			if err != nil {
				return nil, fmt.Errorf("no image AI config found: %w", err)
			}
		}
	} else {
		config, err = s.aiService.GetDefaultConfig("image")
		if err != nil {
			return nil, fmt.Errorf("no image AI config found: %w", err)
		}
	}

	// 使用指定的模型或配置中的第一个模型
	model := modelName
	if model == "" && len(config.Model) > 0 {
		model = config.Model[0]
	}

	// 使用配置中的 provider，如果没有则使用传入的 provider
	actualProvider := config.Provider
	if actualProvider == "" {
		actualProvider = provider
	}

	// 根据 provider 自动设置默认端点
	var endpoint string
	var queryEndpoint string

	switch actualProvider {
	case "openai", "dalle":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "chatfire":
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	case "volcengine", "volces", "doubao":
		// 使用火山引擎原生端点，支持 size 参数
		// 注意：BaseURL 已包含 /api/v3，所以 endpoint 应该是 images/generations
		endpoint = "images/generations"
		queryEndpoint = ""
		return image.NewVolcEngineImageClient(config.BaseURL, config.APIKey, model, endpoint, queryEndpoint), nil
	case "gemini", "google":
		endpoint = "/v1beta/models/{model}:generateContent"
		return image.NewGeminiImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	default:
		endpoint = "/images/generations"
		return image.NewOpenAIImageClient(config.BaseURL, config.APIKey, model, endpoint), nil
	}
}

func (s *ImageGenerationService) GetImageGeneration(imageGenID uint) (*models.ImageGeneration, error) {
	var imageGen models.ImageGeneration
	if err := s.db.Where("id = ? ", imageGenID).First(&imageGen).Error; err != nil {
		return nil, err
	}
	return &imageGen, nil
}

func (s *ImageGenerationService) ListImageGenerations(dramaID *uint, sceneID *uint, storyboardID *uint, frameType string, status string, page, pageSize int) ([]models.ImageGeneration, int64, error) {
	query := s.db.Model(&models.ImageGeneration{})

	if dramaID != nil {
		query = query.Where("drama_id = ?", *dramaID)
	}

	if sceneID != nil {
		query = query.Where("scene_id = ?", *sceneID)
	}

	if storyboardID != nil {
		query = query.Where("storyboard_id = ?", *storyboardID)
	}

	if frameType != "" {
		query = query.Where("frame_type = ?", frameType)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var images []models.ImageGeneration
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&images).Error; err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

func (s *ImageGenerationService) DeleteImageGeneration(imageGenID uint) error {
	result := s.db.Where("id = ? ", imageGenID).Delete(&models.ImageGeneration{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("image generation not found")
	}
	return nil
}

func (s *ImageGenerationService) CreateImageGeneration(storyboardID *uint, frameType, imageURL, prompt string) (*models.ImageGeneration, error) {
	imageGen := &models.ImageGeneration{
		StoryboardID: storyboardID,
		FrameType:    &frameType,
		ImageURL:     &imageURL,
		Prompt:       prompt,
		ImageType:    string(models.ImageTypeStoryboard),
		Status:       models.ImageStatusCompleted,
		Provider:     "upload",
		Model:        "manual_upload",
	}

	if err := s.db.Create(imageGen).Error; err != nil {
		return nil, fmt.Errorf("failed to create image generation record: %w", err)
	}

	return imageGen, nil
}

func (s *ImageGenerationService) GenerateImagesForScene(sceneID string) ([]*models.ImageGeneration, error) {
	// 转换sceneID
	sid, err := strconv.ParseUint(sceneID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid scene ID")
	}
	sceneIDUint := uint(sid)

	var scene models.Scene
	if err := s.db.Where("id = ?", sceneIDUint).First(&scene).Error; err != nil {
		return nil, fmt.Errorf("scene not found")
	}

	// 构建场景图片生成提示词
	prompt := scene.Prompt
	if prompt == "" {
		// 如果Prompt为空，使用Location和Time构建
		prompt = fmt.Sprintf("%s场景，%s", scene.Location, scene.Time)
	}

	req := &GenerateImageRequest{
		SceneID:   &sceneIDUint,
		DramaID:   fmt.Sprintf("%d", scene.DramaID),
		ImageType: string(models.ImageTypeScene),
		Prompt:    prompt,
	}

	imageGen, err := s.GenerateImage(req)
	if err != nil {
		return nil, err
	}

	return []*models.ImageGeneration{imageGen}, nil
}

// BackgroundInfo 背景信息结构
type BackgroundInfo struct {
	Location          string `json:"location"`
	Time              string `json:"time"`
	Atmosphere        string `json:"atmosphere"`
	Prompt            string `json:"prompt"`
	StoryboardNumbers []int  `json:"storyboard_numbers"`
	SceneIDs          []uint `json:"scene_ids"`
	StoryboardCount   int    `json:"scene_count"`
}

func (s *ImageGenerationService) BatchGenerateImagesForEpisode(episodeID string) ([]*models.ImageGeneration, error) {
	var ep models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&ep).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}
	// 从数据库读取已保存的场景
	var scenes []models.Storyboard
	if err := s.db.Where("episode_id = ?", episodeID).Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to get scenes: %w", err)
	}

	backgrounds := s.extractUniqueBackgrounds(scenes)
	s.log.Infow("Extracted unique backgrounds",
		"episode_id", episodeID,
		"background_count", len(backgrounds))

	// 为每个背景生成图片
	var results []*models.ImageGeneration
	for _, bg := range scenes {
		if bg.ImagePrompt == nil || *bg.ImagePrompt == "" {
			s.log.Warnw("Background has no prompt, skipping", "scene_id", bg.ID)
			continue
		}

		// 更新背景状态为处理中
		s.db.Model(bg).Update("status", "generating")

		// 使用火山引擎API（豆包）
		req := &GenerateImageRequest{
			StoryboardID: &bg.ID,
			DramaID:      fmt.Sprintf("%d", ep.DramaID),
			Prompt:       *bg.ImagePrompt,
			Provider:     "volcengine",
			Model:        "doubao-seedream-4-5-251128",
			Size:         "2560x1440",
			Quality:      "standard",
		}

		imageGen, err := s.GenerateImage(req)
		if err != nil {
			s.log.Errorw("Failed to generate image for background",
				"scene_id", bg.ID,
				"location", bg.Location,
				"error", err)
			s.db.Model(bg).Update("status", "failed")
			continue
		}

		s.log.Infow("Background image generation started",
			"scene_id", bg.ID,
			"image_gen_id", imageGen.ID,
			"location", bg.Location,
			"time", bg.Time)

		results = append(results, imageGen)
	}

	return results, nil
}

// GetScencesForEpisode 获取项目的场景列表（项目级）
func (s *ImageGenerationService) GetScencesForEpisode(episodeID string) ([]*models.Scene, error) {
	var episode models.Episode
	if err := s.db.Preload("Drama").Where("id = ?", episodeID).First(&episode).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	// 场景是项目级的，通过drama_id查询
	var scenes []*models.Scene
	if err := s.db.Where("drama_id = ?", episode.DramaID).Order("location ASC, time ASC").Find(&scenes).Error; err != nil {
		return nil, fmt.Errorf("failed to load scenes: %w", err)
	}

	return scenes, nil
}

// ExtractBackgroundsForEpisode 从剧本内容中提取场景并保存到项目级别数据库
func (s *ImageGenerationService) ExtractBackgroundsForEpisode(episodeID string, model string) ([]*models.Scene, error) {
	var episode models.Episode
	if err := s.db.Preload("Storyboards").First(&episode, episodeID).Error; err != nil {
		return nil, fmt.Errorf("episode not found")
	}

	// 如果没有剧本内容，无法提取场景
	if episode.ScriptContent == nil || *episode.ScriptContent == "" {
		return nil, fmt.Errorf("episode has no script content")
	}

	s.log.Infow("Extracting backgrounds from script", "episode_id", episodeID, "model", model)
	dramaID := episode.DramaID

	// 使用AI从剧本内容中提取场景
	backgroundsInfo, err := s.extractBackgroundsFromScript(*episode.ScriptContent, dramaID, model)
	if err != nil {
		s.log.Errorw("Failed to extract backgrounds from script", "error", err)
		return nil, err
	}

	// 保存到数据库（不涉及Storyboard关联，因为此时还没有生成分镜）
	var scenes []*models.Scene
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 先删除该章节的所有场景（实现重新提取覆盖功能）
		if err := tx.Where("episode_id = ?", episode.ID).Delete(&models.Scene{}).Error; err != nil {
			s.log.Errorw("Failed to delete old scenes", "error", err)
			return err
		}
		s.log.Infow("Deleted old scenes for re-extraction", "episode_id", episode.ID)

		// 创建新提取的场景
		for _, bgInfo := range backgroundsInfo {
			// 保存新场景到数据库（章节级）
			episodeIDVal := episode.ID
			scene := &models.Scene{
				DramaID:         dramaID,
				EpisodeID:       &episodeIDVal,
				Location:        bgInfo.Location,
				Time:            bgInfo.Time,
				Prompt:          bgInfo.Prompt,
				StoryboardCount: 1, // 默认为1
				Status:          "pending",
			}
			if err := tx.Create(scene).Error; err != nil {
				return err
			}
			scenes = append(scenes, scene)

			s.log.Infow("Created new scene from script",
				"scene_id", scene.ID,
				"location", scene.Location,
				"time", scene.Time)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.log.Infow("Saved scenes to database",
		"episode_id", episodeID,
		"total_storyboards", len(episode.Storyboards),
		"unique_scenes", len(scenes))

	return scenes, nil
}

// extractBackgroundsFromScript 从剧本内容中使用AI提取场景信息
func (s *ImageGenerationService) extractBackgroundsFromScript(scriptContent string, dramaID uint, model string) ([]BackgroundInfo, error) {
	if scriptContent == "" {
		return []BackgroundInfo{}, nil
	}

	// 获取AI客户端（如果指定了模型则使用指定的模型）
	var client ai.AIClient
	var err error
	if model != "" {
		s.log.Infow("Using specified model for background extraction", "model", model)
		client, err = s.aiService.GetAIClientForModel("text", model)
		if err != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", err)
			client, err = s.aiService.GetAIClient("text")
		}
	} else {
		client, err = s.aiService.GetAIClient("text")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get AI client: %w", err)
	}

	// 使用国际化提示词
	systemPrompt := s.promptI18n.GetSceneExtractionPrompt()
	contentLabel := s.promptI18n.FormatUserPrompt("script_content_label")

	// 强制使用中文格式说明（移除英文支持）
	formatInstructions := `【输出JSON格式】
 {
   "backgrounds": [
     {
       "location": "地点名称（纯中文）",
       "time": "时间描述（纯中文）",
       "atmosphere": "氛围描述（纯中文）",
	      "prompt": "动漫风格纯背景场景，展现[地点描述]在[时间]的环境。画面呈现[环境细节、建筑、物品、光线等，不包含人物]。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：[环境氛围形容词]。"
     }
   ]
 }

	【重要约束】
	- 所有字段必须100%使用中文，严禁出现任何英文单词、字母或标点符号
	- prompt字段必须是纯中文描述，禁止使用英文术语
	- 禁止使用英文术语，如"background"应写成"背景"，"scene"应写成"场景"，"anime style"应写成"动漫风格"
	- prompt必须严格静态化：只写背景环境，不得包含人物、动作、事件、剧情、镜头术语、拟人化叙事

	【正确示例】
	{
	  "backgrounds": [
	    {
	      "location": "维修店内部",
	      "time": "深夜",
	      "atmosphere": "昏暗、孤独、工业感",
	      "prompt": "动漫风格纯背景场景，展现凌乱的维修店内部在深夜的环境。昏暗的日光灯照射下，工作台上散落着各种扳手、螺丝刀和机械零件，墙上挂着油污斑斑的工具挂板和褪色海报，地面有油渍痕迹，角落堆放着废旧轮胎。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：昏暗、冷清、工业感。"
	    },
	    {
	      "location": "城市街道",
	      "time": "黄昏",
	      "atmosphere": "温暖、繁忙、生活气息",
	      "prompt": "动漫风格纯背景场景，展现繁华的城市街道在黄昏时分的环境。夕阳的余晖洒在街道的沥青路面上，两旁的商铺霓虹灯开始点亮，街边有自行车停靠架和公交站牌，远处高楼林立，天空呈现橙红色渐变。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：温暖、繁忙、生活气息。"
	    }
	  ]
	}

【错误示例 - 严禁模仿】
❌ "Anime style background scene..." - 包含英文
❌ "动漫风格background scene..." - 中英混杂
❌ "动漫风格纯背景，depicting a messy..." - 中英混杂
	❌ "展现主角站在街道上的场景" - 包含人物
	❌ "人们匆匆而过" - 包含人物
	❌ "角色在房间里活动" - 包含人物
	❌ "爆炸后的街道" - 包含事件/剧情
	❌ "仿佛在等待主人归来" - 拟人化叙事
	❌ "特写镜头，广角" - 镜头语言
	
	请严格按照JSON格式输出，确保所有字段都使用纯中文，不得包含任何英文。`

	prompt := fmt.Sprintf(`%s

%s
%s

%s`, systemPrompt, contentLabel, scriptContent, formatInstructions)

	// 打印完整提示词用于调试
	s.log.Infow("=== AI Prompt for Background Extraction (extractBackgroundsFromScript) ===",
		"language", s.promptI18n.GetLanguage(),
		"prompt_length", len(prompt),
		"full_prompt", prompt)

	response, err := client.GenerateText(prompt, "", ai.WithTemperature(0.7))
	if err != nil {
		s.log.Errorw("Failed to extract backgrounds with AI", "error", err)
		return nil, fmt.Errorf("AI提取场景失败: %w", err)
	}

	// 打印AI返回的原始响应
	s.log.Infow("=== AI Response for Background Extraction (extractBackgroundsFromScript) ===",
		"response_length", len(response),
		"raw_response", response)

	// 解析AI返回的JSON
	var backgrounds []BackgroundInfo

	// 先尝试解析为数组格式
	if err := utils.SafeParseAIJSON(response, &backgrounds); err == nil {
		s.log.Infow("Parsed backgrounds as array format", "count", len(backgrounds))
	} else {
		// 尝试解析为对象格式
		var result struct {
			Backgrounds []BackgroundInfo `json:"backgrounds"`
		}
		if err := utils.SafeParseAIJSON(response, &result); err != nil {
			s.log.Errorw("Failed to parse AI response in both formats", "error", err, "response", response[:min(len(response), 500)])
			return nil, fmt.Errorf("解析AI响应失败: %w", err)
		}
		backgrounds = result.Backgrounds
		s.log.Infow("Parsed backgrounds as object format", "count", len(backgrounds))
	}

	s.log.Infow("Extracted backgrounds from script",
		"drama_id", dramaID,
		"backgrounds_count", len(backgrounds))

	return backgrounds, nil
}

// extractBackgroundsWithAI 使用AI智能分析场景并提取唯一背景
func (s *ImageGenerationService) extractBackgroundsWithAI(storyboards []models.Storyboard) ([]BackgroundInfo, error) {
	if len(storyboards) == 0 {
		return []BackgroundInfo{}, nil
	}

	// 构建场景列表文本，使用SceneNumber而不是索引
	var scenesText string
	for _, storyboard := range storyboards {
		location := ""
		if storyboard.Location != nil {
			location = *storyboard.Location
		}
		time := ""
		if storyboard.Time != nil {
			time = *storyboard.Time
		}
		action := ""
		if storyboard.Action != nil {
			action = *storyboard.Action
		}
		description := ""
		if storyboard.Description != nil {
			description = *storyboard.Description
		}

		scenesText += fmt.Sprintf("镜头%d:\n地点: %s\n时间: %s\n动作: %s\n描述: %s\n\n",
			storyboard.StoryboardNumber, location, time, action, description)
	}

	// 使用国际化提示词
	systemPrompt := s.promptI18n.GetSceneExtractionPrompt()
	storyboardLabel := s.promptI18n.FormatUserPrompt("storyboard_list_label")

	// 强制使用中文格式说明（移除英文支持）
	formatInstructions := `【输出JSON格式】
 {
   "backgrounds": [
     {
       "location": "地点名称（纯中文）",
       "time": "时间描述（纯中文）",
	      "prompt": "动漫风格纯背景场景，展现[地点描述]在[时间]的环境。画面呈现[细节描述，不包含人物]。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：[环境氛围形容词]。",
       "scene_numbers": [1, 2, 3]
     }
   ]
 }

	【重要约束】
	- 所有字段必须100%使用中文，严禁出现任何英文单词、字母或标点符号
	- prompt字段必须是纯中文描述，禁止使用英文术语
	- 禁止使用英文术语，如"background"应写成"背景"，"anime style"应写成"动漫风格"
	- prompt必须严格静态化：只写背景环境，不得包含人物、动作、事件、剧情、镜头术语、拟人化叙事

	【正确示例】
	{
	  "backgrounds": [
	    {
	      "location": "维修店",
	      "time": "深夜",
	      "prompt": "动漫风格纯背景场景，展现凌乱的维修店内部在深夜的环境。昏暗的灯光下，工作台上散落着各种工具和零件，墙上挂着油污的海报。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：昏暗、冷清、工业感。",
	      "scene_numbers": [1, 5, 6, 10, 15]
	    },
	    {
	      "location": "城市全景",
	      "time": "深夜·酸雨",
	      "prompt": "动漫风格纯背景场景，展现沿海城市全景在深夜酸雨中的环境。霓虹灯在雨中模糊，高楼大厦笼罩在灰绿色的雨幕中，街道反射着五颜六色的光。无人物，无角色，空场景。风格：细节丰富，高质量。氛围：压抑、科幻、末世感。",
	      "scene_numbers": [2, 7]
	    }
	  ]
	}

【错误示例 - 严禁模仿】
	❌ "Anime style background depicting..." - 包含英文
	❌ "动漫风格background depicting..." - 中英混杂
	❌ "动漫风格纯背景，depicting a messy..." - 中英混杂
	❌ "有人在街上奔跑" - 包含人物/动作
	❌ "爆炸后的城市" - 包含事件/剧情
	❌ "仿佛在等待黎明" - 拟人化叙事
	❌ "广角镜头，俯拍" - 镜头语言

请严格按照JSON格式输出，确保：
1. prompt字段必须100%使用纯中文，不得包含任何英文
2. scene_numbers包含所有使用该背景的场景编号
3. 所有场景都被分配到某个背景`

	prompt := fmt.Sprintf(`%s

%s
%s

%s`, systemPrompt, storyboardLabel, scenesText, formatInstructions)

	// 打印完整提示词用于调试
	s.log.Infow("=== AI Prompt for Background Extraction (extractBackgroundsWithAI) ===",
		"language", s.promptI18n.GetLanguage(),
		"prompt_length", len(prompt),
		"full_prompt", prompt)

	// 调用AI服务
	text, err := s.aiService.GenerateText(prompt, "")
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// 打印AI返回的原始响应
	s.log.Infow("=== AI Response for Background Extraction ===",
		"response_length", len(text),
		"raw_response", text)

	// 解析AI返回的JSON
	var result struct {
		Scenes []struct {
			Location         string `json:"location"`
			Time             string `json:"time"`
			Prompt           string `json:"prompt"`
			StoryboardNumber []int  `json:"storyboard_number"`
		} `json:"backgrounds"`
	}

	if err := utils.SafeParseAIJSON(text, &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 构建场景编号到场景ID的映射
	storyboardNumberToID := make(map[int]uint)
	for _, scene := range storyboards {
		storyboardNumberToID[scene.StoryboardNumber] = scene.ID
	}

	// 转换为BackgroundInfo
	var backgrounds []BackgroundInfo
	for _, bg := range result.Scenes {
		// 将场景编号转换为场景ID
		var sceneIDs []uint
		for _, storyboardNum := range bg.StoryboardNumber {
			if storyboardID, ok := storyboardNumberToID[storyboardNum]; ok {
				sceneIDs = append(sceneIDs, storyboardID)
			}
		}

		backgrounds = append(backgrounds, BackgroundInfo{
			Location:          bg.Location,
			Time:              bg.Time,
			Prompt:            bg.Prompt,
			StoryboardNumbers: bg.StoryboardNumber,
			SceneIDs:          sceneIDs,
			StoryboardCount:   len(sceneIDs),
		})
	}

	s.log.Infow("AI extracted backgrounds",
		"total_scenes", len(storyboards),
		"extracted_backgrounds", len(backgrounds))

	return backgrounds, nil
}

// extractUniqueBackgrounds 从分镜头中提取唯一背景（代码逻辑，作为AI提取的备份）
func (s *ImageGenerationService) extractUniqueBackgrounds(scenes []models.Storyboard) []BackgroundInfo {
	backgroundMap := make(map[string]*BackgroundInfo)

	for _, scene := range scenes {
		if scene.Location == nil || scene.Time == nil {
			continue
		}

		// 使用 location + time 作为唯一标识
		key := *scene.Location + "|" + *scene.Time

		if bg, exists := backgroundMap[key]; exists {
			// 背景已存在，添加scene ID
			bg.SceneIDs = append(bg.SceneIDs, scene.ID)
			bg.StoryboardCount++
		} else {
			// 新背景 - 使用ImagePrompt构建背景提示词
			prompt := ""
			if scene.ImagePrompt != nil {
				prompt = *scene.ImagePrompt
			}
			backgroundMap[key] = &BackgroundInfo{
				Location:        *scene.Location,
				Time:            *scene.Time,
				Prompt:          prompt,
				SceneIDs:        []uint{scene.ID},
				StoryboardCount: 1,
			}
		}
	}

	// 转换为切片
	var backgrounds []BackgroundInfo
	for _, bg := range backgroundMap {
		backgrounds = append(backgrounds, *bg)
	}

	return backgrounds
}
