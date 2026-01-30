package handlers

import (
	"strconv"

	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SceneLibraryHandler struct {
	libraryService *services.SceneLibraryService
	imageList      *services.SceneImageService
	log            *logger.Logger
}

func NewSceneLibraryHandler(db *gorm.DB, log *logger.Logger, localStorage *storage.LocalStorage) *SceneLibraryHandler {
	return &SceneLibraryHandler{
		libraryService: services.NewSceneLibraryService(db, log, localStorage),
		imageList:      services.NewSceneImageService(db, log, localStorage),
		log:            log,
	}
}

func (h *SceneLibraryHandler) ListLibraryItems(c *gin.Context) {
	var query services.SceneLibraryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	items, total, err := h.libraryService.ListLibraryItems(&query)
	if err != nil {
		h.log.Errorw("Failed to list scene library items", "error", err)
		response.InternalError(c, "获取场景库失败")
		return
	}
	response.SuccessWithPagination(c, items, total, query.Page, query.PageSize)
}

func (h *SceneLibraryHandler) CreateLibraryItem(c *gin.Context) {
	var req services.CreateSceneLibraryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	item, err := h.libraryService.CreateLibraryItem(&req)
	if err != nil {
		h.log.Errorw("Failed to create scene library item", "error", err)
		response.InternalError(c, "添加到场景库失败")
		return
	}
	response.Created(c, item)
}

func (h *SceneLibraryHandler) GetLibraryItem(c *gin.Context) {
	itemID := c.Param("id")
	item, err := h.libraryService.GetLibraryItem(itemID)
	if err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "场景库项不存在")
			return
		}
		h.log.Errorw("Failed to get scene library item", "error", err)
		response.InternalError(c, "获取失败")
		return
	}
	response.Success(c, item)
}

func (h *SceneLibraryHandler) DeleteLibraryItem(c *gin.Context) {
	itemID := c.Param("id")
	if err := h.libraryService.DeleteLibraryItem(itemID); err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "场景库项不存在")
			return
		}
		h.log.Errorw("Failed to delete scene library item", "error", err)
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"message": "删除成功"})
}

func (h *SceneLibraryHandler) ApplyLibraryItemToScene(c *gin.Context) {
	sceneID := c.Param("scene_id")
	sceneIDUint, err := strconv.ParseUint(sceneID, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的场景ID")
		return
	}
	var req struct {
		LibraryItemID string `json:"library_item_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	libraryItem, err := h.libraryService.GetLibraryItem(req.LibraryItemID)
	if err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "场景库项不存在")
			return
		}
		h.log.Errorw("Failed to get scene library item", "error", err)
		response.InternalError(c, "获取失败")
		return
	}

	if err := h.libraryService.ApplyLibraryItemToScene(sceneID, req.LibraryItemID); err != nil {
		switch err.Error() {
		case "library item not found":
			response.NotFound(c, "场景库项不存在")
			return
		case "scene not found":
			response.NotFound(c, "场景不存在")
			return
		case "unauthorized":
			response.Forbidden(c, "无权限")
			return
		}
		h.log.Errorw("Failed to apply scene library item", "error", err)
		response.InternalError(c, "应用失败")
		return
	}

	prevImages, err := h.imageList.List(uint(sceneIDUint))
	if err != nil {
		h.log.Errorw("Failed to list scene images before apply", "error", err, "scene_id", sceneID)
		response.InternalError(c, "应用失败")
		return
	}
	added, err := h.imageList.Add(uint(sceneIDUint), libraryItem.ImageURL)
	if err != nil {
		h.log.Errorw("Failed to add scene image from library", "error", err, "scene_id", sceneID, "library_item_id", req.LibraryItemID)
		response.InternalError(c, "应用失败")
		return
	}
	newOrder := make([]uint, 0, len(prevImages)+1)
	newOrder = append(newOrder, added.ID)
	for _, img := range prevImages {
		newOrder = append(newOrder, img.ID)
	}
	if err := h.imageList.Reorder(uint(sceneIDUint), newOrder); err != nil {
		h.log.Errorw("Failed to reorder scene images after apply", "error", err, "scene_id", sceneID)
		response.InternalError(c, "应用失败")
		return
	}
	response.Success(c, gin.H{"message": "应用成功"})
}

func (h *SceneLibraryHandler) AddSceneToLibrary(c *gin.Context) {
	sceneID := c.Param("scene_id")
	var req struct {
		Category *string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Category = nil
	}

	item, err := h.libraryService.AddSceneToLibrary(sceneID, req.Category)
	if err != nil {
		switch err.Error() {
		case "scene not found":
			response.NotFound(c, "场景不存在")
			return
		case "unauthorized":
			response.Forbidden(c, "无权限")
			return
		case "scene has no image":
			response.BadRequest(c, "场景还没有图片")
			return
		}
		h.log.Errorw("Failed to add scene to library", "error", err)
		response.InternalError(c, "添加失败")
		return
	}
	response.Created(c, item)
}
