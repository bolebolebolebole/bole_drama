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

type CharacterImageService struct {
	db           *gorm.DB
	log          *logger.Logger
	localStorage *storage.LocalStorage
}

func NewCharacterImageService(db *gorm.DB, log *logger.Logger, localStorage *storage.LocalStorage) *CharacterImageService {
	return &CharacterImageService{db: db, log: log, localStorage: localStorage}
}

func (s *CharacterImageService) List(characterID uint) ([]models.CharacterImage, error) {
	var items []models.CharacterImage
	if err := s.db.Where("character_id = ?", characterID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CharacterImageService) Add(characterID uint, imageURL string) (*models.CharacterImage, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil, fmt.Errorf("image_url is required")
	}

	// Ensure image becomes permanent when possible.
	if s.localStorage != nil && !s.localStorage.IsLocalURL(imageURL) && (strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://")) {
		if localURL, err := s.localStorage.DownloadFromURL(imageURL, "characters"); err == nil {
			imageURL = localURL
		}
	}

	// Determine next sort order.
	var maxSort int
	_ = s.db.Model(&models.CharacterImage{}).Where("character_id = ?", characterID).Select("COALESCE(MAX(sort_order), -1)").Scan(&maxSort).Error
	item := &models.CharacterImage{CharacterID: characterID, ImageURL: imageURL, SortOrder: maxSort + 1}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}

	// Keep Character.image_url synced to the first image.
	if err := s.syncPrimary(characterID); err != nil {
		s.log.Warnw("Failed to sync character primary image", "error", err, "character_id", characterID)
	}

	return item, nil
}

func (s *CharacterImageService) Reorder(characterID uint, imageIDs []uint) error {
	if len(imageIDs) == 0 {
		return fmt.Errorf("image_ids is required")
	}

	// Validate ownership.
	var count int64
	if err := s.db.Model(&models.CharacterImage{}).Where("character_id = ? AND id IN ?", characterID, imageIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(imageIDs)) {
		return fmt.Errorf("invalid image ids")
	}

	tx := s.db.Begin()
	for idx, id := range imageIDs {
		if err := tx.Model(&models.CharacterImage{}).Where("id = ? AND character_id = ?", id, characterID).Update("sort_order", idx).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return s.syncPrimary(characterID)
}

func (s *CharacterImageService) Delete(characterID uint, imageID uint) error {
	res := s.db.Where("id = ? AND character_id = ?", imageID, characterID).Delete(&models.CharacterImage{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("not found")
	}
	return s.syncPrimary(characterID)
}

func (s *CharacterImageService) syncPrimary(characterID uint) error {
	var first models.CharacterImage
	err := s.db.Where("character_id = ?", characterID).Order("sort_order ASC, id ASC").First(&first).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.db.Model(&models.Character{}).Where("id = ?", characterID).Update("image_url", nil).Error
		}
		return err
	}
	return s.db.Model(&models.Character{}).Where("id = ?", characterID).Update("image_url", first.ImageURL).Error
}
