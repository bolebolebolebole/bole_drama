package models

import (
	"time"

	"gorm.io/gorm"
)

// SceneLibrary 场景库模型（可复用的背景场景图与提示词）
type SceneLibrary struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Location   string         `gorm:"type:varchar(200);not null" json:"location"`
	Time       string         `gorm:"type:varchar(100);not null" json:"time"`
	Prompt     string         `gorm:"type:text;not null" json:"prompt"`
	Category   *string        `gorm:"type:varchar(50)" json:"category"`
	ImageURL   string         `gorm:"type:varchar(500);not null" json:"image_url"`
	SourceType string         `gorm:"type:varchar(20);default:'generated'" json:"source_type"` // generated, uploaded
	CreatedAt  time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *SceneLibrary) TableName() string {
	return "scene_libraries"
}
