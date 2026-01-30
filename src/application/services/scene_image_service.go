package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/drama-generator/backend/domain/models"
	"github.com/drama-generator/backend/infrastructure/storage"
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type SceneImageService struct {
	db           *gorm.DB
	log          *logger.Logger
	localStorage *storage.LocalStorage
}

func NewSceneImageService(db *gorm.DB, log *logger.Logger, localStorage *storage.LocalStorage) *SceneImageService {
	return &SceneImageService{db: db, log: log, localStorage: localStorage}
}

func (s *SceneImageService) List(sceneID uint) ([]models.SceneImage, error) {
	var items []models.SceneImage
	if err := s.db.Where("scene_id = ?", sceneID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SceneImageService) Add(sceneID uint, imageURL string) (*models.SceneImage, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil, fmt.Errorf("image_url is required")
	}

	if s.localStorage != nil && !s.localStorage.IsLocalURL(imageURL) && (strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://")) {
		if localURL, err := s.localStorage.DownloadFromURL(imageURL, "scenes"); err == nil {
			imageURL = localURL
		}
	}

	var maxSort int
	_ = s.db.Model(&models.SceneImage{}).Where("scene_id = ?", sceneID).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxSort).Error
	item := &models.SceneImage{SceneID: sceneID, ImageURL: imageURL, SortOrder: maxSort + 1}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}

	if err := s.syncPrimary(sceneID); err != nil {
		s.log.Warnw("Failed to sync scene primary image", "error", err, "scene_id", sceneID)
	}
	return item, nil
}

func (s *SceneImageService) Reorder(sceneID uint, imageIDs []uint) error {
	if len(imageIDs) == 0 {
		return fmt.Errorf("image_ids is required")
	}
	var count int64
	if err := s.db.Model(&models.SceneImage{}).Where("scene_id = ? AND id IN ?", sceneID, imageIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(imageIDs)) {
		return fmt.Errorf("invalid image ids")
	}

	tx := s.db.Begin()
	for idx, id := range imageIDs {
		if err := tx.Model(&models.SceneImage{}).Where("id = ? AND scene_id = ?", id, sceneID).Update("sort_order", idx).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return s.syncPrimary(sceneID)
}

func (s *SceneImageService) Delete(sceneID uint, imageID uint) error {
	res := s.db.Where("id = ? AND scene_id = ?", imageID, sceneID).Delete(&models.SceneImage{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("not found")
	}
	return s.syncPrimary(sceneID)
}

func (s *SceneImageService) syncPrimary(sceneID uint) error {
	var first models.SceneImage
	err := s.db.Where("scene_id = ?", sceneID).Order("sort_order ASC, id ASC").First(&first).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.db.Model(&models.Scene{}).Where("id = ?", sceneID).Update("image_url", nil).Error
		}
		return err
	}
	return s.db.Model(&models.Scene{}).Where("id = ?", sceneID).Update("image_url", first.ImageURL).Error
}
