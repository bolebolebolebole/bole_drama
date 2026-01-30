package handlers

import (
	"strconv"

	services2 "github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CharacterLibraryHandler struct {
	libraryService *services2.CharacterLibraryService
	imageList      *services2.CharacterImageService
	imageService   *services2.ImageGenerationService
	log            *logger.Logger
}

func NewCharacterLibraryHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger, transferService *services2.ResourceTransferService, localStorage *storage.LocalStorage) *CharacterLibraryHandler {
	return &CharacterLibraryHandler{
		libraryService: services2.NewCharacterLibraryService(db, log, localStorage),
		imageList:      services2.NewCharacterImageService(db, log, localStorage),
		imageService:   services2.NewImageGenerationService(db, cfg, transferService, localStorage, log),
		log:            log,
	}
}

// ListCharacterImages returns all images for a character.
func (h *CharacterLibraryHandler) ListCharacterImages(c *gin.Context) {
	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}
	items, err := h.imageList.List(uint(characterID))
	if err != nil {
		h.log.Errorw("Failed to list character images", "error", err, "character_id", characterID)
		response.InternalError(c, "获取失败")
		return
	}
	response.Success(c, gin.H{"items": items})
}

// AddCharacterImage appends a new image for a character.
func (h *CharacterLibraryHandler) AddCharacterImage(c *gin.Context) {
	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}
	var req struct {
		ImageURL string `json:"image_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.imageList.Add(uint(characterID), req.ImageURL)
	if err != nil {
		h.log.Errorw("Failed to add character image", "error", err, "character_id", characterID)
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, item)
}

// ReorderCharacterImages updates sort order.
func (h *CharacterLibraryHandler) ReorderCharacterImages(c *gin.Context) {
	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}
	var req struct {
		ImageIDs []uint `json:"image_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.imageList.Reorder(uint(characterID), req.ImageIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

// DeleteCharacterImage deletes an image.
func (h *CharacterLibraryHandler) DeleteCharacterImage(c *gin.Context) {
	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}
	imageIDStr := c.Param("image_id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的图片ID")
		return
	}
	if err := h.imageList.Delete(uint(characterID), uint(imageID)); err != nil {
		if err.Error() == "not found" {
			response.NotFound(c, "图片不存在")
			return
		}
		response.InternalError(c, "删除失败")
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

// ListLibraryItems 获取角色库列表
func (h *CharacterLibraryHandler) ListLibraryItems(c *gin.Context) {

	var query services2.CharacterLibraryQuery
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
		h.log.Errorw("Failed to list library items", "error", err)
		response.InternalError(c, "获取角色库失败")
		return
	}

	response.SuccessWithPagination(c, items, total, query.Page, query.PageSize)
}

// CreateLibraryItem 添加到角色库
func (h *CharacterLibraryHandler) CreateLibraryItem(c *gin.Context) {

	var req services2.CreateLibraryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	item, err := h.libraryService.CreateLibraryItem(&req)
	if err != nil {
		h.log.Errorw("Failed to create library item", "error", err)
		response.InternalError(c, "添加到角色库失败")
		return
	}

	response.Created(c, item)
}

// GetLibraryItem 获取角色库项详情
func (h *CharacterLibraryHandler) GetLibraryItem(c *gin.Context) {

	itemID := c.Param("id")

	item, err := h.libraryService.GetLibraryItem(itemID)
	if err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "角色库项不存在")
			return
		}
		h.log.Errorw("Failed to get library item", "error", err)
		response.InternalError(c, "获取失败")
		return
	}

	response.Success(c, item)
}

// DeleteLibraryItem 删除角色库项
func (h *CharacterLibraryHandler) DeleteLibraryItem(c *gin.Context) {

	itemID := c.Param("id")

	if err := h.libraryService.DeleteLibraryItem(itemID); err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "角色库项不存在")
			return
		}
		h.log.Errorw("Failed to delete library item", "error", err)
		response.InternalError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// UploadCharacterImage 上传角色图片
func (h *CharacterLibraryHandler) UploadCharacterImage(c *gin.Context) {

	characterID := c.Param("id")

	// TODO: 处理文件上传
	// 这里需要实现文件上传逻辑，保存到OSS或本地
	// 暂时使用简单的实现
	var req struct {
		ImageURL string `json:"image_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.libraryService.UploadCharacterImage(characterID, req.ImageURL); err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "角色不存在")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "无权限")
			return
		}
		h.log.Errorw("Failed to upload character image", "error", err)
		response.InternalError(c, "上传失败")
		return
	}

	response.Success(c, gin.H{"message": "上传成功"})
}

// ApplyLibraryItemToCharacter 从角色库应用形象
func (h *CharacterLibraryHandler) ApplyLibraryItemToCharacter(c *gin.Context) {

	characterID := c.Param("id")
	characterIDUint, err := strconv.ParseUint(characterID, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
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
			response.NotFound(c, "角色库项不存在")
			return
		}
		h.log.Errorw("Failed to get library item", "error", err)
		response.InternalError(c, "获取角色库项失败")
		return
	}

	if err := h.libraryService.ApplyLibraryItemToCharacter(characterID, req.LibraryItemID); err != nil {
		if err.Error() == "library item not found" {
			response.NotFound(c, "角色库项不存在")
			return
		}
		if err.Error() == "character not found" {
			response.NotFound(c, "角色不存在")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "无权限")
			return
		}
		h.log.Errorw("Failed to apply library item", "error", err)
		response.InternalError(c, "应用失败")
		return
	}

	// Ensure applied image is also tracked in multi-image list.
	prevImages, err := h.imageList.List(uint(characterIDUint))
	if err != nil {
		h.log.Errorw("Failed to list character images before apply", "error", err, "character_id", characterID)
		response.InternalError(c, "应用失败")
		return
	}
	added, err := h.imageList.Add(uint(characterIDUint), libraryItem.ImageURL)
	if err != nil {
		h.log.Errorw("Failed to add character image from library", "error", err, "character_id", characterID, "library_item_id", req.LibraryItemID)
		response.InternalError(c, "应用失败")
		return
	}
	newOrder := make([]uint, 0, len(prevImages)+1)
	newOrder = append(newOrder, added.ID)
	for _, img := range prevImages {
		newOrder = append(newOrder, img.ID)
	}
	if err := h.imageList.Reorder(uint(characterIDUint), newOrder); err != nil {
		h.log.Errorw("Failed to reorder character images after apply", "error", err, "character_id", characterID)
		response.InternalError(c, "应用失败")
		return
	}

	response.Success(c, gin.H{"message": "应用成功"})
}

// AddCharacterToLibrary 将角色添加到角色库
func (h *CharacterLibraryHandler) AddCharacterToLibrary(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		Category *string `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空body
		req.Category = nil
	}

	item, err := h.libraryService.AddCharacterToLibrary(characterID, req.Category)
	if err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "角色不存在")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "无权限")
			return
		}
		if err.Error() == "character has no image" {
			response.BadRequest(c, "角色还没有形象图片")
			return
		}
		h.log.Errorw("Failed to add character to library", "error", err)
		response.InternalError(c, "添加失败")
		return
	}

	response.Created(c, item)
}

// UpdateCharacter 更新角色信息
func (h *CharacterLibraryHandler) UpdateCharacter(c *gin.Context) {

	characterID := c.Param("id")

	var req struct {
		Name        *string `json:"name"`
		Role        *string `json:"role"`
		Appearance  *string `json:"appearance"`
		Personality *string `json:"personality"`
		Description *string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.libraryService.UpdateCharacter(characterID, &req); err != nil {
		if err.Error() == "character not found" {
			response.NotFound(c, "角色不存在")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "无权限")
			return
		}
		h.log.Errorw("Failed to update character", "error", err)
		response.InternalError(c, "更新失败")
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteCharacter 删除单个角色
func (h *CharacterLibraryHandler) DeleteCharacter(c *gin.Context) {

	characterIDStr := c.Param("id")
	characterID, err := strconv.ParseUint(characterIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的角色ID")
		return
	}

	if err := h.libraryService.DeleteCharacter(uint(characterID)); err != nil {
		h.log.Errorw("Failed to delete character", "error", err, "id", characterID)
		if err.Error() == "character not found" {
			response.NotFound(c, "角色不存在")
			return
		}
		if err.Error() == "unauthorized" {
			response.Forbidden(c, "无权删除此角色")
			return
		}
		response.InternalError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "角色已删除"})
}
