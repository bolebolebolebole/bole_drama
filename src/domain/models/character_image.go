package models

import "time"

// CharacterImage stores multiple reference images for a character.
// The first image by SortOrder is treated as the primary image.
type CharacterImage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CharacterID uint      `gorm:"index;not null" json:"character_id"`
	ImageURL    string    `gorm:"type:text;not null" json:"image_url"`
	SortOrder   int       `gorm:"index" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
