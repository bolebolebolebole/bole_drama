package services

import (
	"errors"
	"fmt"
	"strings"

	models "github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type SceneLibraryService struct {
	db           *gorm.DB
	log          *logger.Logger
	localStorage *storage.LocalStorage
}

func NewSceneLibraryService(db *gorm.DB, log *logger.Logger, localStorage *storage.LocalStorage) *SceneLibraryService {
	return &SceneLibraryService{db: db, log: log, localStorage: localStorage}
}

func (s *SceneLibraryService) ensurePermanentImageURL(imageURL string, category string) string {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return imageURL
	}
	if s.localStorage == nil {
		return imageURL
	}
	if s.localStorage.IsLocalURL(trimmed) {
		return imageURL
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		localURL, err := s.localStorage.DownloadFromURL(trimmed, category)
		if err != nil {
			errStr := err.Error()
			if len(errStr) > 200 {
				errStr = errStr[:200] + "..."
			}
			s.log.Warnw("Failed to persist image to local storage", "error", errStr, "category", category)
			return imageURL
		}
		return localURL
	}
	return imageURL
}

type SceneLibraryQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
}

type CreateSceneLibraryItemRequest struct {
	Location   string  `json:"location" binding:"required,min=1,max=200"`
	Time       string  `json:"time" binding:"required,min=1,max=100"`
	Prompt     string  `json:"prompt" binding:"required,min=5,max=5000"`
	Category   *string `json:"category"`
	ImageURL   string  `json:"image_url" binding:"required"`
	SourceType string  `json:"source_type"`
}

func (s *SceneLibraryService) ListLibraryItems(query *SceneLibraryQuery) ([]models.SceneLibrary, int64, error) {
	var items []models.SceneLibrary
	var total int64

	db := s.db.Model(&models.SceneLibrary{})
	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		db = db.Where("location LIKE ? OR time LIKE ? OR prompt LIKE ?", like, like, like)
	}

	if err := db.Count(&total).Error; err != nil {
		s.log.Errorw("Failed to count scene library", "error", err)
		return nil, 0, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		s.log.Errorw("Failed to list scene library", "error", err)
		return nil, 0, err
	}

	// 兼容历史数据：库里可能保存了外部临时URL（如TOS签名URL），尝试转存到本地。
	for i := range items {
		orig := items[i].ImageURL
		fixed := s.ensurePermanentImageURL(orig, "library/scenes")
		if fixed != orig {
			items[i].ImageURL = fixed
			_ = s.db.Model(&models.SceneLibrary{}).Where("id = ?", items[i].ID).Update("image_url", fixed).Error
		}
	}

	return items, total, nil
}

func (s *SceneLibraryService) CreateLibraryItem(req *CreateSceneLibraryItemRequest) (*models.SceneLibrary, error) {
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "generated"
	}

	imageURL := s.ensurePermanentImageURL(req.ImageURL, "library/scenes")

	item := &models.SceneLibrary{
		Location:   req.Location,
		Time:       req.Time,
		Prompt:     req.Prompt,
		Category:   req.Category,
		ImageURL:   imageURL,
		SourceType: sourceType,
	}

	if err := s.db.Create(item).Error; err != nil {
		s.log.Errorw("Failed to create scene library item", "error", err)
		return nil, err
	}
	return item, nil
}

func (s *SceneLibraryService) GetLibraryItem(itemID string) (*models.SceneLibrary, error) {
	var item models.SceneLibrary
	if err := s.db.Where("id = ?", itemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("library item not found")
		}
		return nil, err
	}

	// 兼容历史数据：尝试把外部临时URL转存为本地URL
	orig := item.ImageURL
	fixed := s.ensurePermanentImageURL(orig, "library/scenes")
	if fixed != orig {
		item.ImageURL = fixed
		_ = s.db.Model(&models.SceneLibrary{}).Where("id = ?", item.ID).Update("image_url", fixed).Error
	}
	return &item, nil
}

func (s *SceneLibraryService) DeleteLibraryItem(itemID string) error {
	result := s.db.Where("id = ?", itemID).Delete(&models.SceneLibrary{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("library item not found")
	}
	return nil
}

// ApplyLibraryItemToScene 将场景库形象应用到场景（覆盖 image_url + prompt）
func (s *SceneLibraryService) ApplyLibraryItemToScene(sceneID string, libraryItemID string) error {
	var item models.SceneLibrary
	if err := s.db.Where("id = ?", libraryItemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("library item not found")
		}
		return err
	}

	var scene models.Scene
	if err := s.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("scene not found")
		}
		return err
	}

	// 验证 scene 所属剧本存在（开源版本不做用户鉴权）
	var drama models.Drama
	if err := s.db.Where("id = ?", scene.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("unauthorized")
		}
		return err
	}

	updates := map[string]interface{}{
		"image_url": item.ImageURL,
		"prompt":    item.Prompt,
	}
	if err := s.db.Model(&scene).Updates(updates).Error; err != nil {
		s.log.Errorw("Failed to apply scene library item", "error", err, "scene_id", sceneID, "library_item_id", libraryItemID)
		return err
	}

	return nil
}

// AddSceneToLibrary 将场景添加到场景库
func (s *SceneLibraryService) AddSceneToLibrary(sceneID string, category *string) (*models.SceneLibrary, error) {
	var scene models.Scene
	if err := s.db.Where("id = ?", sceneID).First(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("scene not found")
		}
		return nil, err
	}

	var drama models.Drama
	if err := s.db.Where("id = ?", scene.DramaID).First(&drama).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("unauthorized")
		}
		return nil, err
	}

	if scene.ImageURL == nil || *scene.ImageURL == "" {
		return nil, fmt.Errorf("scene has no image")
	}

	imageURL := s.ensurePermanentImageURL(*scene.ImageURL, "library/scenes")

	item := &models.SceneLibrary{
		Location:   scene.Location,
		Time:       scene.Time,
		Prompt:     scene.Prompt,
		Category:   category,
		ImageURL:   imageURL,
		SourceType: "scene",
	}
	if err := s.db.Create(item).Error; err != nil {
		s.log.Errorw("Failed to add scene to library", "error", err, "scene_id", sceneID)
		return nil, err
	}
	return item, nil
}
