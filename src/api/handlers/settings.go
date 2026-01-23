package handlers

import (
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	config *config.Config
	log    *logger.Logger
}

func NewSettingsHandler(cfg *config.Config, log *logger.Logger) *SettingsHandler {
	return &SettingsHandler{
		config: cfg,
		log:    log,
	}
}

// GetLanguage 获取当前系统语言（固定返回中文）
func (h *SettingsHandler) GetLanguage(c *gin.Context) {
	response.Success(c, gin.H{
		"language": "zh", // 固定返回中文
	})
}

// UpdateLanguage 更新系统语言（已废弃，固定使用中文）
func (h *SettingsHandler) UpdateLanguage(c *gin.Context) {
	response.Success(c, gin.H{
		"message":  "系统已固定使用简体中文",
		"language": "zh",
	})
}
