package handlers

import (
	"fmt"
	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/config"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/drama-generator/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

type StoryboardHandler struct {
	storyboardService *services.StoryboardService
	taskService       *services.TaskService
	log               *logger.Logger
}

func NewStoryboardHandler(db *gorm.DB, cfg *config.Config, log *logger.Logger) *StoryboardHandler {
	return &StoryboardHandler{
		storyboardService: services.NewStoryboardService(db, cfg, log),
		taskService:       services.NewTaskService(db, log),
		log:               log,
	}
}

// GenerateStoryboard 生成分镜头（异步）
func (h *StoryboardHandler) GenerateStoryboard(c *gin.Context) {
	episodeID := c.Param("episode_id")

	// 接收可选的 model 参数
	var req struct {
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供body或者解析失败，使用空字符串（使用默认模型）
		req.Model = ""
	}

	// 如果该集数已有进行中的分镜任务，复用它，避免重复发起（用户重复点击/刷新页面）
	if existing, err := h.taskService.FindActiveTask("storyboard_generation", episodeID, 30*time.Minute); err == nil && existing != nil {
		response.Success(c, gin.H{
			"task_id": existing.ID,
			"status":  existing.Status,
			"message": existing.Message,
		})
		return
	}

	// 创建异步任务
	task, err := h.taskService.CreateTask("storyboard_generation", episodeID)
	if err != nil {
		h.log.Errorw("Failed to create task", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	// 启动后台goroutine处理
	go h.processStoryboardGeneration(task.ID, episodeID, req.Model)

	// 立即返回任务ID
	response.Success(c, gin.H{
		"task_id": task.ID,
		"status":  "pending",
		"message": "分镜头生成任务已创建，正在后台处理...",
	})
}

// processStoryboardGeneration 后台处理分镜生成
func (h *StoryboardHandler) processStoryboardGeneration(taskID, episodeID, model string) {
	h.log.Infow("Starting storyboard generation", "task_id", taskID, "episode_id", episodeID, "model", model)

	// 更新任务状态为处理中
	if err := h.taskService.UpdateTaskStatus(taskID, "processing", 10, "开始生成分镜..."); err != nil {
		h.log.Errorw("Failed to update task status", "error", err)
	}

	// 心跳：长时间生成时刷新 updated_at，避免前端以为卡死
	stopHeartbeat := make(chan struct{})
	startedAt := time.Now()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				elapsed := int(time.Since(startedAt).Seconds())
				_ = h.taskService.UpdateTaskStatus(taskID, "processing", 20, fmt.Sprintf("生成分镜中...（已运行%d秒）", elapsed))
			case <-stopHeartbeat:
				return
			}
		}
	}()

	// 进入模型调用阶段
	_ = h.taskService.UpdateTaskStatus(taskID, "processing", 20, "请求模型生成分镜...")

	// 调用实际的生成逻辑
	result, err := h.storyboardService.GenerateStoryboard(episodeID, model)
	close(stopHeartbeat)
	if err != nil {
		h.log.Errorw("Failed to generate storyboard", "error", err, "task_id", taskID)
		if updateErr := h.taskService.UpdateTaskError(taskID, err); updateErr != nil {
			h.log.Errorw("Failed to update task error", "error", updateErr)
		}
		return
	}

	_ = h.taskService.UpdateTaskStatus(taskID, "processing", 80, "解析并保存分镜...")

	// 更新任务结果
	if err := h.taskService.UpdateTaskResult(taskID, result); err != nil {
		h.log.Errorw("Failed to update task result", "error", err)
		return
	}

	h.log.Infow("Storyboard generation completed", "task_id", taskID, "total", result.Total)
}

// UpdateStoryboard 更新分镜
func (h *StoryboardHandler) UpdateStoryboard(c *gin.Context) {
	storyboardID := c.Param("id")

	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	err := h.storyboardService.UpdateStoryboard(storyboardID, req)
	if err != nil {
		h.log.Errorw("Failed to update storyboard", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Storyboard updated successfully"})
}

// UpdateReferenceOverrides 更新参考图替换
func (h *StoryboardHandler) UpdateReferenceOverrides(c *gin.Context) {
	storyboardID := c.Param("id")

	var req map[string]interface{}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	err := h.storyboardService.UpdateReferenceOverrides(storyboardID, req)
	if err != nil {
		h.log.Errorw("Failed to update reference overrides", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Reference overrides updated successfully"})
}

// InsertStoryboard 在指定位置插入新分镜
func (h *StoryboardHandler) InsertStoryboard(c *gin.Context) {
	episodeID := c.Param("episode_id")

	var req struct {
		InsertAfter int `json:"insert_after"` // 在第几个分镜后插入（0表示插入到最前面）
		Duration    int `json:"duration"`     // 默认时长
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	// 设置默认时长
	if req.Duration <= 0 {
		req.Duration = 5
	}

	// 调用service层插入分镜
	storyboard, err := h.storyboardService.InsertStoryboard(episodeID, req.InsertAfter, req.Duration)
	if err != nil {
		h.log.Errorw("Failed to insert storyboard", "error", err)
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, storyboard)
}
