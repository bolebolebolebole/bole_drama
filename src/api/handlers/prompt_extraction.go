package handlers

import (
	"errors"
	"strconv"

	"github.com/drama-generator/backend/application/services"
	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PromptExtractionHandler struct {
	db            *gorm.DB
	scriptService *services.ScriptGenerationService
	imageService  *services.ImageGenerationService
	log           *logger.Logger
}

func NewPromptExtractionHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, imageService *services.ImageGenerationService) *PromptExtractionHandler {
	return &PromptExtractionHandler{
		db:            db,
		scriptService: services.NewScriptGenerationService(db, cfg, log),
		imageService:  imageService,
		log:           log,
	}
}

type reextractPromptRequest struct {
	EpisodeID     *uint   `json:"episode_id"`
	ScriptContent *string `json:"script_content"`
	Model         string  `json:"model"`
}

func (h *PromptExtractionHandler) ReextractCharacterPrompt(c *gin.Context) {
	characterIDStr := c.Param("id")
	characterID64, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}

	var req reextractPromptRequest
	_ = c.ShouldBindJSON(&req)

	// Load character
	var character models.Character
	if err := h.db.Where("id = ?", uint(characterID64)).First(&character).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "角色不存在")
			return
		}
		h.log.Errorw("Failed to find character", "error", err, "character_id", characterIDStr)
		response.InternalError(c, "查询失败")
		return
	}

	// Resolve script content
	scriptContent := ""
	if req.ScriptContent != nil {
		scriptContent = *req.ScriptContent
	}
	if scriptContent == "" {
		if req.EpisodeID == nil {
			response.BadRequest(c, "缺少 episode_id 或 script_content")
			return
		}
		var ep models.Episode
		if err := h.db.Where("id = ?", *req.EpisodeID).First(&ep).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "章节不存在")
				return
			}
			h.log.Errorw("Failed to find episode", "error", err, "episode_id", *req.EpisodeID)
			response.InternalError(c, "查询章节失败")
			return
		}
		if ep.ScriptContent == nil || *ep.ScriptContent == "" {
			response.BadRequest(c, "章节没有剧本内容")
			return
		}
		scriptContent = *ep.ScriptContent
	}

	updated, err := h.scriptService.ReextractCharacterFromScript(uint(characterID64), scriptContent, req.Model)
	if err != nil {
		h.log.Errorw("Failed to reextract character prompt", "error", err, "character_id", characterIDStr)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":   "角色提示词已重新提取",
		"character": updated,
	})
}

func (h *PromptExtractionHandler) ReextractScenePrompt(c *gin.Context) {
	sceneIDStr := c.Param("scene_id")
	sceneID64, err := strconv.ParseUint(sceneIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}

	var req reextractPromptRequest
	_ = c.ShouldBindJSON(&req)

	// Load scene
	var scene models.Scene
	if err := h.db.Where("id = ?", uint(sceneID64)).First(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "场景不存在")
			return
		}
		h.log.Errorw("Failed to find scene", "error", err, "scene_id", sceneIDStr)
		response.InternalError(c, "查询失败")
		return
	}

	// Resolve script content
	scriptContent := ""
	if req.ScriptContent != nil {
		scriptContent = *req.ScriptContent
	}
	if scriptContent == "" {
		episodeID := req.EpisodeID
		if episodeID == nil {
			// best effort: use scene.EpisodeID when provided
			episodeID = scene.EpisodeID
		}
		if episodeID == nil {
			response.BadRequest(c, "缺少 episode_id 或 script_content")
			return
		}
		var ep models.Episode
		if err := h.db.Where("id = ?", *episodeID).First(&ep).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "章节不存在")
				return
			}
			h.log.Errorw("Failed to find episode", "error", err, "episode_id", *episodeID)
			response.InternalError(c, "查询章节失败")
			return
		}
		if ep.ScriptContent == nil || *ep.ScriptContent == "" {
			response.BadRequest(c, "章节没有剧本内容")
			return
		}
		scriptContent = *ep.ScriptContent
	}

	updated, err := h.imageService.ReextractScenePrompt(uint(sceneID64), scriptContent, req.Model)
	if err != nil {
		h.log.Errorw("Failed to reextract scene prompt", "error", err, "scene_id", sceneIDStr)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "场景提示词已重新提取",
		"scene":   updated,
	})
}
