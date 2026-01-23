package handlers

import (
	"errors"

	services2 "github.com/drama-generator/backend/application/services"
	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SceneHandler struct {
	db           *gorm.DB
	sceneService *services2.StoryboardCompositionService
	log          *logger.Logger
}

func NewSceneHandler(db *gorm.DB, log *logger.Logger, imageGenService *services2.ImageGenerationService) *SceneHandler {
	return &SceneHandler{
		db:           db,
		sceneService: services2.NewStoryboardCompositionService(db, log, imageGenService),
		log:          log,
	}
}

func (h *SceneHandler) GetStoryboardsForEpisode(c *gin.Context) {
	episodeID := c.Param("episode_id")

	storyboards, err := h.sceneService.GetScenesForEpisode(episodeID)
	if err != nil {
		h.log.Errorw("Failed to get storyboards for episode", "error", err, "episode_id", episodeID)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"storyboards": storyboards,
		"total":       len(storyboards),
	})
}

func (h *SceneHandler) UpdateScene(c *gin.Context) {
	sceneID := c.Param("scene_id")

	// Note: This endpoint updates the Scene library item (models.Scene),
	// not a storyboard shot.
	var req struct {
		Location *string `json:"location"`
		Time     *string `json:"time"`
		Prompt   *string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	// Find the scene
	var scene models.Scene
	if err := h.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "场景不存在")
			return
		}
		h.log.Errorw("Failed to find scene", "error", err, "scene_id", sceneID)
		response.InternalError(c, "更新失败")
		return
	}

	updates := make(map[string]interface{})
	if req.Location != nil {
		updates["location"] = *req.Location
	}
	if req.Time != nil {
		updates["time"] = *req.Time
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if len(updates) == 0 {
		response.BadRequest(c, "No fields to update")
		return
	}

	if err := h.db.Model(&scene).Updates(updates).Error; err != nil {
		h.log.Errorw("Failed to update scene", "error", err, "scene_id", sceneID)
		response.InternalError(c, "更新失败")
		return
	}

	response.Success(c, gin.H{"message": "Scene updated successfully"})
}

func (h *SceneHandler) GenerateSceneImage(c *gin.Context) {
	var req services2.GenerateSceneImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	imageGen, err := h.sceneService.GenerateSceneImage(&req)
	if err != nil {
		h.log.Errorw("Failed to generate scene image", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":          "Scene image generation started",
		"image_generation": imageGen,
	})
}

func (h *SceneHandler) DeleteScene(c *gin.Context) {
	sceneID := c.Param("scene_id")

	if err := h.sceneService.DeleteScene(sceneID); err != nil {
		h.log.Errorw("Failed to delete scene", "error", err, "scene_id", sceneID)
		if err.Error() == "scene not found" {
			response.NotFound(c, "场景不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "场景已删除"})
}
