package services

import (
	"fmt"
	"strconv"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/ai"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/utils"
	"gorm.io/gorm"
)

type ScriptGenerationService struct {
	db         *gorm.DB
	aiService  *AIService
	log        *logger.Logger
	config     *config.Config
	promptI18n *PromptI18n
}

type SingleCharacterExtractionResult struct {
	Role        string `json:"role"`
	Appearance  string `json:"appearance"`
	Personality string `json:"personality"`
	Description string `json:"description"`
	VoiceStyle  string `json:"voice_style"`
}

// ReextractCharacterFromScript 重新从剧本中提取指定角色的提示词/设定（仅更新该角色，不影响其他角色）
func (s *ScriptGenerationService) ReextractCharacterFromScript(characterID uint, scriptContent string, model string) (*models.Character, error) {
	var character models.Character
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		return nil, fmt.Errorf("character not found")
	}
	if scriptContent == "" {
		return nil, fmt.Errorf("script content is empty")
	}

	// Prefer a dedicated system prompt to avoid extracting unrelated roles.
	systemPrompt := `你是一个专业的角色分析师，擅长从剧本中提取和分析角色信息。

你的任务是：只针对指定角色，从剧本内容中提取并整理该角色的详细设定。

【重要：appearance用于“可复用的角色形象图/角色卡”】【必须严格静态化】
appearance 只能写角色的稳定外观特征（像证件照/角色设定卡），便于后续反复复用同一张角色图。

appearance 允许包含：
- 性别、年龄段、身高体型
- 肤色肤质、脸型、五官、明显识别点（疤痕/痣/酒窝/虎牙等）
- 发型发色
- 常用穿搭风格（款式/材质/颜色/整体风格）和稳定配饰
- 可见的身体特征（如手部薄茧/纹身/疤痕），但不要解释成因

appearance 严禁包含任何：
- 动作/姿态/运动
- 情绪/表情/心理/状态判断（如：疲惫、憔悴、绝望、坚定、怯懦等）
- 关系/他人名字/互动
- 时间地点/环境/镜头语言/叙事

额外严禁：
- 任何“前后变化/对比”或剧情词（如：原本、曾经、之前、之后、变故后、经历背叛、觉醒、褪去、燃起）
- 因果/转折连接词引导的叙事（如：因为、由于、所以、导致、让她/使她、但、却、然而）
- 眼神/神情/泪痕/通红/颤抖等表情状态词

【appearance 输出格式硬性要求】
只写“名词+形容词”的静态短语，用中文逗号分隔；尽量避免句号和长句；不要写职业/身份标签。

要求：
	1. 只输出一个JSON对象，不要输出数组，不要包含任何markdown代码块、说明文字或其他内容。
	2. 所有字段必须使用中文。
	3. 字段要求：
	   - role: 角色类型（main/supporting/minor）
	   - appearance: 外貌描述（80-180字，严格静态化，只写外观与稳定穿搭）
	   - personality: 性格特点（100-200字）
	   - description: 背景故事和角色关系（100-200字，动作/关系写在这里，不要写进appearance）
	   - voice_style: 声线风格（可为空字符串）`

	userPrompt := fmt.Sprintf("【角色名】%s\n\n【剧本内容】\n%s\n\n请按以下JSON对象格式输出：\n{\n  \"role\": \"main\",\n  \"appearance\": \"...\",\n  \"personality\": \"...\",\n  \"description\": \"...\",\n  \"voice_style\": \"...\"\n}", character.Name, scriptContent)

	var text string
	var err error
	if model != "" {
		s.log.Infow("Using specified model for single character re-extraction", "model", model, "character_id", characterID)
		client, getErr := s.aiService.GetAIClientForModel("text", model)
		if getErr != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", model, "error", getErr)
			text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(0.2))
		} else {
			text, err = client.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(0.2))
		}
	} else {
		text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(0.2))
	}
	if err != nil {
		s.log.Errorw("Failed to reextract character", "error", err, "character_id", characterID)
		return nil, fmt.Errorf("重新提取角色提示词失败: %w", err)
	}

	var result SingleCharacterExtractionResult
	if err := utils.SafeParseAIJSON(text, &result); err != nil {
		s.log.Errorw("Failed to parse single character JSON", "error", err, "character_id", characterID, "raw_response", text[:minInt(500, len(text))])
		return nil, fmt.Errorf("解析AI返回结果失败: %w", err)
	}

	updates := map[string]interface{}{}
	if result.Role != "" {
		role := result.Role
		updates["role"] = &role
	}
	if result.Appearance != "" {
		appearance := sanitizeCharacterAppearance(result.Appearance)
		updates["appearance"] = &appearance
	}
	if result.Personality != "" {
		personality := result.Personality
		updates["personality"] = &personality
	}
	if result.Description != "" {
		desc := result.Description
		updates["description"] = &desc
	}
	if result.VoiceStyle != "" {
		voice := result.VoiceStyle
		updates["voice_style"] = &voice
	}

	if len(updates) == 0 {
		return &character, nil
	}
	if err := s.db.Model(&character).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to update character after re-extraction", "error", err, "character_id", characterID)
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	// Reload
	if err := s.db.Where("id = ?", characterID).First(&character).Error; err != nil {
		return nil, err
	}
	return &character, nil
}

func NewScriptGenerationService(db *gorm.DB, cfg *config.Config, log *logger.Logger) *ScriptGenerationService {
	return &ScriptGenerationService{
		db:         db,
		aiService:  NewAIService(db, log),
		log:        log,
		config:     cfg,
		promptI18n: NewPromptI18n(cfg),
	}
}

type GenerateCharactersRequest struct {
	DramaID     string  `json:"drama_id" binding:"required"`
	EpisodeID   uint    `json:"episode_id"`
	Outline     string  `json:"outline"`
	Count       int     `json:"count"`
	Temperature float64 `json:"temperature"`
	Model       string  `json:"model"` // 指定使用的文本模型
	// AllowCreate controls whether missing characters can be created.
	// For re-extraction flows, set false to avoid creating duplicate characters due to naming variations.
	AllowCreate *bool `json:"allow_create"`
}

func (s *ScriptGenerationService) GenerateCharacters(req *GenerateCharactersRequest) ([]models.Character, error) {
	var drama models.Drama
	if err := s.db.Where("id = ? ", req.DramaID).First(&drama).Error; err != nil {
		return nil, fmt.Errorf("drama not found")
	}

	count := req.Count
	if count == 0 {
		count = 5
	}

	systemPrompt := s.promptI18n.GetCharacterExtractionPrompt()

	outlineText := req.Outline
	if outlineText == "" {
		outlineText = s.promptI18n.FormatUserPrompt("drama_info_template", drama.Title, drama.Description, drama.Genre)
	}

	userPrompt := s.promptI18n.FormatUserPrompt("character_request", outlineText, count)

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// 如果指定了模型，使用指定的模型；否则使用默认配置
	var text string
	var err error
	if req.Model != "" {
		s.log.Infow("Using specified model for character generation", "model", req.Model)
		client, getErr := s.aiService.GetAIClientForModel("text", req.Model)
		if getErr != nil {
			s.log.Warnw("Failed to get client for specified model, using default", "model", req.Model, "error", getErr)
			text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
		} else {
			text, err = client.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
		}
	} else {
		text, err = s.aiService.GenerateText(userPrompt, systemPrompt, ai.WithTemperature(temperature))
	}

	if err != nil {
		s.log.Errorw("Failed to generate characters", "error", err)
		return nil, fmt.Errorf("生成失败: %w", err)
	}

	s.log.Infow("AI response received", "length", len(text), "preview", text[:minInt(200, len(text))])

	// AI直接返回数组格式
	var result []struct {
		Name        string `json:"name"`
		Role        string `json:"role"`
		Description string `json:"description"`
		Personality string `json:"personality"`
		Appearance  string `json:"appearance"`
		VoiceStyle  string `json:"voice_style"`
	}

	if err := utils.SafeParseAIJSON(text, &result); err != nil {
		s.log.Errorw("Failed to parse characters JSON", "error", err, "raw_response", text[:minInt(500, len(text))])
		return nil, fmt.Errorf("解析 AI 返回结果失败: %w", err)
	}

	// Load existing characters once to avoid creating duplicates.
	var existingCharacters []models.Character
	if err := s.db.Where("drama_id = ?", drama.ID).Find(&existingCharacters).Error; err != nil {
		s.log.Errorw("Failed to load existing characters", "error", err, "drama_id", req.DramaID)
		return nil, fmt.Errorf("查询已有角色失败: %w", err)
	}

	allowCreate := len(existingCharacters) == 0
	if req.AllowCreate != nil {
		allowCreate = *req.AllowCreate
	}
	if !allowCreate {
		s.log.Infow("Character generation in reuse-only mode", "drama_id", req.DramaID, "episode_id", req.EpisodeID)
	}

	existingByName := make(map[string]models.Character, len(existingCharacters))
	aliasToExisting := make(map[string]models.Character, len(existingCharacters)*2)
	for _, c := range existingCharacters {
		existingByName[c.Name] = c
		for _, alias := range buildCharacterAliases(c.Name) {
			if alias == "" {
				continue
			}
			if _, ok := aliasToExisting[alias]; !ok {
				aliasToExisting[alias] = c
			}
		}
	}

	var characters []models.Character
	newCount := 0
	for _, char := range result {
		// 1) Exact name match
		if existing, ok := existingByName[char.Name]; ok {
			s.log.Infow("Character already exists, reusing", "drama_id", req.DramaID, "name", char.Name, "character_id", existing.ID)
			characters = append(characters, existing)
			continue
		}

		// 2) Alias match (e.g. "苏遥遥（遥遥）" vs "苏遥（遥遥)")
		matched := false
		for _, alias := range buildCharacterAliases(char.Name) {
			if existing, ok := aliasToExisting[alias]; ok {
				s.log.Infow("Character matched by alias, reusing", "drama_id", req.DramaID, "name", char.Name, "alias", alias, "character_id", existing.ID)
				characters = append(characters, existing)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// 3) Missing character
		if !allowCreate {
			s.log.Warnw("Character not found; creation disabled, skipping",
				"drama_id", req.DramaID,
				"episode_id", req.EpisodeID,
				"name", char.Name)
			continue
		}

		dramaID, _ := strconv.ParseUint(req.DramaID, 10, 32)
		appearance := sanitizeCharacterAppearance(char.Appearance)
		character := models.Character{
			DramaID:     uint(dramaID),
			Name:        char.Name,
			Role:        &char.Role,
			Description: &char.Description,
			Personality: &char.Personality,
			Appearance:  &appearance,
			VoiceStyle:  &char.VoiceStyle,
		}

		if err := s.db.Create(&character).Error; err != nil {
			s.log.Errorw("Failed to create character", "error", err, "name", char.Name, "drama_id", req.DramaID)
			continue
		}
		newCount++
		characters = append(characters, character)
	}

	// 如果提供了 EpisodeID，建立 episode_characters 关联关系
	if req.EpisodeID > 0 {
		var episode models.Episode
		if err := s.db.First(&episode, req.EpisodeID).Error; err == nil {
			// 使用 GORM 的 Association 建立多对多关联
			if err := s.db.Model(&episode).Association("Characters").Append(characters); err != nil {
				s.log.Errorw("Failed to associate characters with episode", "error", err, "episode_id", req.EpisodeID)
			} else {
				s.log.Infow("Characters associated with episode", "episode_id", req.EpisodeID, "character_count", len(characters))
			}
		} else {
			s.log.Errorw("Episode not found for association", "episode_id", req.EpisodeID, "error", err)
		}
	}

	s.log.Infow("Characters generated", "drama_id", req.DramaID, "total_count", len(characters), "new_count", newCount, "allow_create", allowCreate)
	return characters, nil
}

// GenerateScenesForEpisode 已废弃，使用 StoryboardService.GenerateStoryboard 替代
// ParseScript 已废弃，使用 GenerateCharacters 替代

// minInt 返回两个整数中较小的一个
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
