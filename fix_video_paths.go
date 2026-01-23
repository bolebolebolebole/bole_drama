package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"modernc.org/sqlite"
)

type VideoGeneration struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string
	VideoURL  string
	LocalPath string
}

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("./data/drama_generator.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 获取所有已完成但没有local_path的视频
	var videos []VideoGeneration
	if err := db.Where("status = ? AND (local_path IS NULL OR local_path = '')", "completed").
		Order("created_at ASC").
		Find(&videos).Error; err != nil {
		log.Fatalf("Failed to query videos: %v", err)
	}

	fmt.Printf("Found %d videos without local_path\n", len(videos))

	// 获取所有视频文件
	videoFiles := make(map[string]os.FileInfo)
	videoDir := "./data/storage/videos"

	err = filepath.Walk(videoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".mp4") {
			videoFiles[info.Name()] = info
			return nil
		}
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to scan video directory: %v", err)
	}

	fmt.Printf("Found %d video files in storage\n", len(videoFiles))

	// 为每个视频文件创建时间映射
	type FileWithTime struct {
		Name    string
		ModTime time.Time
	}

	var sortedFiles []FileWithTime
	for name, info := range videoFiles {
		sortedFiles = append(sortedFiles, FileWithTime{
			Name:    name,
			ModTime: info.ModTime(),
		})
	}

	// 按修改时间排序文件
	for i := 0; i < len(sortedFiles); i++ {
		for j := i + 1; j < len(sortedFiles); j++ {
			if sortedFiles[i].ModTime.After(sortedFiles[j].ModTime) {
				sortedFiles[i], sortedFiles[j] = sortedFiles[j], sortedFiles[i]
			}
		}
	}

	// 匹配视频记录和文件
	// 策略：按创建时间顺序匹配，找到最接近的文件
	updated := 0
	failed := 0

	for _, video := range videos {
		// 找到最接近创建时间的文件（在创建时间之后的最近文件）
		var bestMatch *FileWithTime
		var minDiff time.Duration = time.Hour * 24 * 365 // 1年

		for i := range sortedFiles {
			file := &sortedFiles[i]
			// 文件修改时间应该在视频创建时间之后（下载需要时间）
			if file.ModTime.After(video.CreatedAt) {
				diff := file.ModTime.Sub(video.CreatedAt)
				// 通常下载在几秒到几分钟内完成
				if diff < minDiff && diff < time.Minute*10 {
					minDiff = diff
					bestMatch = file
				}
			}
		}

		if bestMatch != nil {
			localPath := fmt.Sprintf("/static/videos/%s", bestMatch.Name)

			// 更新数据库
			if err := db.Model(&VideoGeneration{}).
				Where("id = ?", video.ID).
				Update("local_path", localPath).Error; err != nil {
				log.Printf("Failed to update video %d: %v", video.ID, err)
				failed++
			} else {
				fmt.Printf("✓ Video ID %d -> %s (diff: %v)\n", video.ID, bestMatch.Name, minDiff)
				updated++

				// 从列表中移除已匹配的文件，避免重复匹配
				for i, f := range sortedFiles {
					if f.Name == bestMatch.Name {
						sortedFiles = append(sortedFiles[:i], sortedFiles[i+1:]...)
						break
					}
				}
			}
		} else {
			log.Printf("✗ No matching file found for video ID %d (created at %v)", video.ID, video.CreatedAt)
			failed++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total videos: %d\n", len(videos))
	fmt.Printf("Updated: %d\n", updated)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Remaining unmatched files: %d\n", len(sortedFiles))

	if len(sortedFiles) > 0 {
		fmt.Printf("\nUnmatched files:\n")
		for _, f := range sortedFiles {
			fmt.Printf("  - %s (modified: %v)\n", f.Name, f.ModTime)
		}
	}
}
