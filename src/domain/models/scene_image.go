package models

import "time"

// SceneImage stores multiple reference images for a scene/background.
// The first image by SortOrder is treated as the primary image.
type SceneImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SceneID   uint      `gorm:"index;not null" json:"scene_id"`
	ImageURL  string    `gorm:"type:text;not null" json:"image_url"`
	SortOrder int       `gorm:"index" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
