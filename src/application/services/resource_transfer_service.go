package services

import (
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type ResourceTransferService struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewResourceTransferService(db *gorm.DB, log *logger.Logger) *ResourceTransferService {
	return &ResourceTransferService{
		db:  db,
		log: log,
	}
}

// ResourceTransferService 现在只保留基本结构，MinIO相关功能已移除
// 如需资源转存功能，请使用本地存储

// BatchTransferImagesToMinio 批量转存图片到MinIO（已废弃 - MinIO功能已移除）
// 返回 (0, nil) 表示不执行任何操作
func (s *ResourceTransferService) BatchTransferImagesToMinio(dramaID string, limit int) (int, error) {
	s.log.Warnw("BatchTransferImagesToMinio called but MinIO is disabled - no-op", "drama_id", dramaID, "limit", limit)
	return 0, nil
}

// BatchTransferVideosToMinio 批量转存视频到MinIO（已废弃 - MinIO功能已移除）
// 返回 (0, nil) 表示不执行任何操作
func (s *ResourceTransferService) BatchTransferVideosToMinio(dramaID string, limit int) (int, error) {
	s.log.Warnw("BatchTransferVideosToMinio called but MinIO is disabled - no-op", "drama_id", dramaID, "limit", limit)
	return 0, nil
}
