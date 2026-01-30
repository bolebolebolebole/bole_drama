package handlers

import (
	"errors"
	"strconv"

	"github.com/drama-generator/backend/infrastructure/storage"

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
	imageList    *services2.SceneImageService
	log          *logger.Logger
}

func (h *SceneHandler) ListSceneImages(c *gin.Context) {
	sceneIDStr := c.Param("scene_id")
	sceneID, err := strconv.ParseUint(sceneIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}
	items, err := h.imageList.List(uint(sceneID))
	if err != nil {
		h.log.Errorw("Failed to list scene images", "error", err, "scene_id", sceneID)
		response.InternalError(c, "获取失败")
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *SceneHandler) AddSceneImage(c *gin.Context) {
	sceneIDStr := c.Param("scene_id")
	sceneID, err := strconv.ParseUint(sceneIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}
	var req struct {
		ImageURL string `json:"image_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.imageList.Add(uint(sceneID), req.ImageURL)
	if err != nil {
		h.log.Errorw("Failed to add scene image", "error", err, "scene_id", sceneID)
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, item)
}

func (h *SceneHandler) ReorderSceneImages(c *gin.Context) {
	sceneIDStr := c.Param("scene_id")
	sceneID, err := strconv.ParseUint(sceneIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}
	var req struct {
		ImageIDs []uint `json:"image_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.imageList.Reorder(uint(sceneID), req.ImageIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func (h *SceneHandler) DeleteSceneImage(c *gin.Context) {
	sceneIDStr := c.Param("scene_id")
	sceneID, err := strconv.ParseUint(sceneIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}
	imageIDStr := c.Param("image_id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的图片ID")
		return
	}
	if err := h.imageList.Delete(uint(sceneID), uint(imageID)); err != nil {
		if err.Error() == "not found" {
			response.NotFound(c, "图片不存在")
			return
		}
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func NewSceneHandler(db *gorm.DB, log *logger.Logger, imageGenService *services2.ImageGenerationService, localStorage *storage.LocalStorage) *SceneHandler {
	return &SceneHandler{
		db:           db,
		sceneService: services2.NewStoryboardCompositionService(db, log, imageGenService),
		imageList:    services2.NewSceneImageService(db, log, localStorage),
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
