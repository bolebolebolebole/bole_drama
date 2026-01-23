package ffmpeg

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/drama-generator/backend/pkg/logger"
)

type FFmpeg struct {
	log     *logger.Logger
	tempDir string
}

func NewFFmpeg(log *logger.Logger) *FFmpeg {
	tempDir := filepath.Join(os.TempDir(), "drama-video-merge")
	os.MkdirAll(tempDir, 0755)

	return &FFmpeg{
		log:     log,
		tempDir: tempDir,
	}
}

type VideoClip struct {
	URL        string
	Duration   float64
	StartTime  float64
	EndTime    float64
	Transition map[string]interface{}
}

type MergeOptions struct {
	OutputPath string
	Clips      []VideoClip
}

func (f *FFmpeg) MergeVideos(opts *MergeOptions) (string, error) {
	if len(opts.Clips) == 0 {
		return "", fmt.Errorf("no video clips to merge")
	}

	f.log.Infow("Starting video merge with trimming", "clips_count", len(opts.Clips))

	// 涓嬭浇骞惰鍓墍鏈夎棰戠墖娈?
	trimmedPaths := make([]string, 0, len(opts.Clips))
	downloadedPaths := make([]string, 0, len(opts.Clips))

	for i, clip := range opts.Clips {
		// 涓嬭浇鍘熷瑙嗛
		downloadPath := filepath.Join(f.tempDir, fmt.Sprintf("download_%d_%d.mp4", time.Now().Unix(), i))
		localPath, err := f.downloadVideo(clip.URL, downloadPath)
		if err != nil {
			f.cleanup(downloadedPaths)
			f.cleanup(trimmedPaths)
			return "", fmt.Errorf("failed to download clip %d: %w", i, err)
		}
		downloadedPaths = append(downloadedPaths, localPath)

		// 瑁佸壀瑙嗛鐗囨锛堟牴鎹甋tartTime鍜孍ndTime锛?
		trimmedPath := filepath.Join(f.tempDir, fmt.Sprintf("trimmed_%d_%d.mp4", time.Now().Unix(), i))
		err = f.trimVideo(localPath, trimmedPath, clip.StartTime, clip.EndTime)
		if err != nil {
			f.cleanup(downloadedPaths)
			f.cleanup(trimmedPaths)
			return "", fmt.Errorf("failed to trim clip %d: %w", i, err)
		}
		trimmedPaths = append(trimmedPaths, trimmedPath)

		f.log.Infow("Clip trimmed",
			"index", i,
			"start", clip.StartTime,
			"end", clip.EndTime,
			"duration", clip.EndTime-clip.StartTime)
	}

	// 娓呯悊涓嬭浇鐨勫師濮嬫枃浠?
	f.cleanup(downloadedPaths)

	// 纭繚杈撳嚭鐩綍瀛樺湪
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		f.cleanup(trimmedPaths)
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 鍚堝苟瑁佸壀鍚庣殑瑙嗛鐗囨锛堟敮鎸佽浆鍦烘晥鏋滐級
	err := f.concatenateVideosWithTransitions(trimmedPaths, opts.Clips, opts.OutputPath)

	// 娓呯悊瑁佸壀鍚庣殑涓存椂鏂囦欢
	f.cleanup(trimmedPaths)

	if err != nil {
		return "", fmt.Errorf("failed to concatenate videos: %w", err)
	}

	f.log.Infow("Video merge completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (f *FFmpeg) downloadVideo(url, destPath string) (string, error) {
	f.log.Infow("Downloading video", "url", url, "dest", destPath)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return destPath, nil
}

func (f *FFmpeg) trimVideo(inputPath, outputPath string, startTime, endTime float64) error {
	f.log.Infow("Trimming video",
		"input", inputPath,
		"output", outputPath,
		"start", startTime,
		"end", endTime)

	// 濡傛灉startTime鍜宔ndTime閮戒负0锛屾垨鑰卐ndTime <= startTime锛屽鍒舵暣涓棰?
	// 浣跨敤閲嶆柊缂栫爜鑰岄潪-c copy浠ョ‘淇濊緭鍑烘枃浠跺畬鏁存€?
	if (startTime == 0 && endTime == 0) || endTime <= startTime {
		f.log.Infow("No valid trim range, re-encoding entire video")

		cmd := exec.Command("ffmpeg",
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			"-y",
			outputPath,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			f.log.Errorw("FFmpeg re-encode failed", "error", err, "output", string(output))
			return fmt.Errorf("ffmpeg re-encode failed: %w, output: %s", err, string(output))
		}

		f.log.Infow("Video re-encoded successfully", "output", outputPath)
		return nil
	}

	// 浣跨敤FFmpeg瑁佸壀瑙嗛
	// -ss: 寮€濮嬫椂闂达紙绉掞級
	// -to/-t: 缁撴潫鏃堕棿鎴栨寔缁椂闂?
	// 浣跨敤閲嶆柊缂栫爜鑰岄潪-c copy浠ョ‘淇濊緭鍑烘枃浠跺畬鏁存€э紝閬垮厤Windows鐜涓嬫祦淇℃伅涓㈠け
	var cmd *exec.Cmd
	if endTime > 0 {
		// 鏈夋槑纭殑缁撴潫鏃堕棿
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-to", fmt.Sprintf("%.2f", endTime),
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			"-y",
			outputPath,
		)
	} else {
		// 鍙湁寮€濮嬫椂闂达紝瑁佸壀鍒拌棰戞湯灏?
		cmd = exec.Command("ffmpeg",
			"-i", inputPath,
			"-ss", fmt.Sprintf("%.2f", startTime),
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			"-y",
			outputPath,
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("FFmpeg trim failed", "error", err, "output", string(output))
		return fmt.Errorf("ffmpeg trim failed: %w, output: %s", err, string(output))
	}

	f.log.Infow("Video trimmed successfully", "output", outputPath)
	return nil
}

func (f *FFmpeg) concatenateVideosWithTransitions(inputPaths []string, clips []VideoClip, outputPath string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input paths")
	}

	// 濡傛灉鍙湁涓€涓棰戯紝鐩存帴澶嶅埗
	if len(inputPaths) == 1 {
		f.log.Infow("Only one clip, copying directly")
		return f.copyFile(inputPaths[0], outputPath)
	}

	// 妫€鏌ユ槸鍚︽湁杞満鏁堟灉
	hasTransitions := false
	for _, clip := range clips {
		if clip.Transition != nil && len(clip.Transition) > 0 {
			hasTransitions = true
			break
		}
	}

	// 濡傛灉娌℃湁杞満鏁堟灉锛屼娇鐢ㄧ畝鍗曟嫾鎺?
	if !hasTransitions {
		f.log.Infow("No transitions, using simple concatenation")
		return f.concatenateVideos(inputPaths, outputPath)
	}

	// 浣跨敤xfade婊ら暅娣诲姞杞満鏁堟灉
	f.log.Infow("Merging with transitions", "clips_count", len(inputPaths))
	return f.mergeWithXfade(inputPaths, clips, outputPath)
}

func (f *FFmpeg) concatenateVideos(inputPaths []string, outputPath string) error {
	// 鍒涘缓鏂囦欢鍒楄〃
	listFile := filepath.Join(f.tempDir, fmt.Sprintf("filelist_%d.txt", time.Now().Unix()))
	defer os.Remove(listFile)

	var content strings.Builder
	for _, path := range inputPaths {
		content.WriteString(fmt.Sprintf("file '%s'\n", path))
	}

	if err := os.WriteFile(listFile, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to create file list: %w", err)
	}

	// 浣跨敤FFmpeg鍚堝苟瑙嗛
	// -f concat: 浣跨敤concat demuxer
	// -safe 0: 鍏佽涓嶅畨鍏ㄧ殑鏂囦欢璺緞
	// -i: 杈撳叆鏂囦欢鍒楄〃
	// -c copy: 鐩存帴澶嶅埗娴侊紝涓嶉噸鏂扮紪鐮侊紙閫熷害蹇級
	cmd := exec.Command("ffmpeg",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y", // 瑕嗙洊杈撳嚭鏂囦欢
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("FFmpeg failed", "error", err, "output", string(output))
		return fmt.Errorf("ffmpeg execution failed: %w, output: %s", err, string(output))
	}

	f.log.Infow("FFmpeg concatenation completed", "output", outputPath)
	return nil
}

func (f *FFmpeg) mergeWithXfade(inputPaths []string, clips []VideoClip, outputPath string) error {
	// 浣跨敤xfade婊ら暅杩涜杞満
	// 鏋勫缓杈撳叆鍙傛暟
	args := []string{}
	for _, path := range inputPaths {
		args = append(args, "-i", path)
	}

	// 妫€娴嬫瘡涓棰戞槸鍚︽湁闊抽娴?
	audioStreams := make([]bool, len(inputPaths))
	hasAnyAudio := false
	for i, path := range inputPaths {
		audioStreams[i] = f.hasAudioStream(path)
		if audioStreams[i] {
			hasAnyAudio = true
		}
		f.log.Infow("Audio stream detection", "index", i, "path", path, "has_audio", audioStreams[i])
	}
	f.log.Infow("Overall audio detection", "has_any_audio", hasAnyAudio, "audio_streams", audioStreams)

	// 妫€娴嬭棰戝垎杈ㄧ巼锛屾壘鍒版渶澶у垎杈ㄧ巼浣滀负鐩爣鍒嗚鲸鐜?
	maxWidth := 0
	maxHeight := 0
	for i, path := range inputPaths {
		width, height := f.getVideoResolution(path)
		if width > maxWidth {
			maxWidth = width
		}
		if height > maxHeight {
			maxHeight = height
		}
		f.log.Infow("Video resolution detection", "index", i, "width", width, "height", height)
	}
	f.log.Infow("Target resolution", "width", maxWidth, "height", maxHeight)

	// 涓烘瘡涓棰戞祦娣诲姞缂╂斁婊ら暅锛岀粺涓€鍒嗚鲸鐜?
	// 鍚屾椂涓烘湁杞満鐨勮棰戞坊鍔?tpad 寤堕暱锛坒reeze 鏈€鍚庝竴甯э級
	var scaleFilters []string
	for i := 0; i < len(inputPaths); i++ {
		// 妫€鏌ュ綋鍓嶈棰戞槸鍚﹂渶瑕佽浆鍦哄埌涓嬩竴涓棰?
		var tpadDuration float64 = 0
		if i < len(clips)-1 && clips[i].Transition != nil {
			// 妫€鏌ヨ浆鍦虹被鍨?
			if tType, ok := clips[i].Transition["type"].(string); ok {
				// none 杞満涓嶉渶瑕?tpad
				if strings.ToLower(tType) != "none" && tType != "" {
					if tDuration, ok := clips[i].Transition["duration"].(float64); ok && tDuration > 0 {
						tpadDuration = tDuration
					} else {
						tpadDuration = 1.0 // 榛樿1绉?
					}
				}
			} else {
				// 娌℃湁鎸囧畾绫诲瀷锛岄粯璁ら渶瑕佽浆鍦?
				if tDuration, ok := clips[i].Transition["duration"].(float64); ok && tDuration > 0 {
					tpadDuration = tDuration
				} else {
					tpadDuration = 1.0
				}
			}
		}

		// 浣跨敤scale婊ら暅缂╂斁鍒扮洰鏍囧垎杈ㄧ巼锛宲ad娣诲姞榛戣竟淇濇寔闀垮姣?
		// 濡傛灉闇€瑕佽浆鍦猴紝浣跨敤 tpad 寤堕暱瑙嗛锛坒reeze鏈€鍚庝竴甯э級
		if tpadDuration > 0 {
			scaleFilters = append(scaleFilters,
				fmt.Sprintf("[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,tpad=stop_mode=clone:stop_duration=%.2f[v%d]",
					i, maxWidth, maxHeight, maxWidth, maxHeight, tpadDuration, i))
			f.log.Infow("Adding tpad to video", "index", i, "duration", tpadDuration)
		} else {
			scaleFilters = append(scaleFilters,
				fmt.Sprintf("[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[v%d]",
					i, maxWidth, maxHeight, maxWidth, maxHeight, i))
		}
	}

	// 鏋勫缓filter_complex
	// 妫€鏌ユ槸鍚︽湁浠讳綍杞満鏁堟灉
	hasAnyTransition := false
	for i := 0; i < len(inputPaths)-1; i++ {
		if clips[i].Transition != nil {
			if tType, ok := clips[i].Transition["type"].(string); ok {
				if strings.ToLower(tType) != "none" && tType != "" {
					hasAnyTransition = true
					break
				}
			}
		}
	}

	// 濡傛灉娌℃湁浠讳綍杞満锛屼娇鐢ㄧ畝鍗曟嫾鎺?
	if !hasAnyTransition {
		f.log.Infow("No transitions detected, using simple concatenation")
		return f.concatenateVideos(inputPaths, outputPath)
	}

	// 鏋勫缓杞満婊ら暅锛屼娇鐢ㄧ缉鏀惧悗鐨勮棰戞祦
	// 瀵规墍鏈夌浉閭昏棰戦兘搴旂敤 xfade锛宼ype=none 鏃朵娇鐢?0 绉掓椂闀垮疄鐜版棤缂濇嫾鎺?
	var transitionFilters []string
	var offset float64 = 0

	for i := 0; i < len(inputPaths)-1; i++ {
		// 鑾峰彇褰撳墠鐗囨鐨勬椂闀?
		clipDuration := clips[i].Duration
		if clips[i].EndTime > 0 && clips[i].StartTime >= 0 {
			clipDuration = clips[i].EndTime - clips[i].StartTime
		}

		// 榛樿杞満鍙傛暟
		transitionType := "fade"
		transitionDuration := 1.0

		if clips[i].Transition != nil {
			if tType, ok := clips[i].Transition["type"].(string); ok {
				if strings.ToLower(tType) == "none" || tType == "" {
					// none 杞満浣跨敤 0 绉掓椂闀匡紝瀹炵幇鏃犵紳鎷兼帴
					transitionDuration = 0.0
					f.log.Infow("Using no transition (0s xfade)", "clip_index", i)
				} else {
					transitionType = f.mapTransitionType(tType)
					f.log.Infow("Using transition type", "type", tType, "mapped", transitionType)
				}
			}
			// 鍙湁闈?none 杞満鎵嶈鍙栨椂闀?
			if transitionDuration > 0 {
				if tDuration, ok := clips[i].Transition["duration"].(float64); ok && tDuration > 0 {
					transitionDuration = tDuration
				}
			}
		}

		// 璁＄畻杞満寮€濮嬬殑鏃堕棿鐐?
		offset += clipDuration
		if offset < 0 {
			offset = 0
		}

		f.log.Infow("Transition settings",
			"clip_index", i,
			"type", transitionType,
			"duration", transitionDuration,
			"offset", offset,
			"clip_duration", clipDuration)

		var inputLabel, outputLabel string
		if i == 0 {
			inputLabel = fmt.Sprintf("[v0][v1]")
		} else {
			inputLabel = fmt.Sprintf("[vx%02d][v%d]", i-1, i+1)
		}

		if i == len(inputPaths)-2 {
			outputLabel = "[outv]"
		} else {
			outputLabel = fmt.Sprintf("[vx%02d]", i)
		}

		filterPart := fmt.Sprintf("%sxfade=transition=%s:duration=%.1f:offset=%.1f%s",
			inputLabel, transitionType, transitionDuration, offset, outputLabel)
		transitionFilters = append(transitionFilters, filterPart)
	}

	// 鍚堝苟缂╂斁鍜岃浆鍦烘护闀?
	var videoFilters []string
	videoFilters = append(videoFilters, scaleFilters...)
	videoFilters = append(videoFilters, transitionFilters...)
	filterComplex := strings.Join(videoFilters, ";")

	// 闊抽澶勭悊锛氬鏋滄湁浠讳綍瑙嗛鍖呭惈闊抽娴侊紝鍒欏鐞嗛煶棰?
	var fullFilter string
	if hasAnyAudio {
		// 涓洪煶棰戞祦娣诲姞澶勭悊锛氱敓鎴愰潤闊虫祦鎴栧欢闀块煶棰?
		var audioFilters []string
		for i := 0; i < len(inputPaths); i++ {
			// 璁＄畻璇ヨ棰戠殑鏃堕暱
			clipDuration := clips[i].Duration
			if clips[i].EndTime > 0 && clips[i].StartTime >= 0 {
				clipDuration = clips[i].EndTime - clips[i].StartTime
			}

			// 妫€鏌ユ槸鍚﹂渶瑕佷负杞満寤堕暱闊抽
			var padDuration float64 = 0
			if i < len(clips)-1 && clips[i].Transition != nil {
				// 妫€鏌ヨ浆鍦虹被鍨?
				needTransition := true
				if tType, ok := clips[i].Transition["type"].(string); ok {
					if strings.ToLower(tType) == "none" || tType == "" {
						needTransition = false
					}
				}

				// 鍙湁闇€瑕佽浆鍦烘椂鎵嶅欢闀块煶棰?
				if needTransition {
					if tDuration, ok := clips[i].Transition["duration"].(float64); ok && tDuration > 0 {
						padDuration = tDuration
					} else {
						padDuration = 1.0
					}
				}
			}

			if !audioStreams[i] {
				// 娌℃湁闊抽鐨勮棰戯細鐢熸垚闈欓煶杞ㄩ亾锛堝寘鎷浆鍦哄欢闀匡級
				totalDuration := clipDuration + padDuration
				audioFilters = append(audioFilters,
					fmt.Sprintf("anullsrc=channel_layout=stereo:sample_rate=44100:duration=%.2f[a%d]", totalDuration, i))
				f.log.Infow("Generated silence for audio", "index", i, "duration", totalDuration)
			} else if padDuration > 0 {
				// 鏈夐煶棰戜笖闇€瑕佸欢闀匡細浣跨敤apad娣诲姞闈欓煶寤堕暱锛堢◢鍚庝細鐢╝crossfade澶勭悊锛?
				audioFilters = append(audioFilters,
					fmt.Sprintf("[%d:a]apad=pad_dur=%.2f[a%d]", i, padDuration, i))
				f.log.Infow("Padding audio with silence", "index", i, "pad_duration", padDuration)
			} else {
				// 鏈夐煶棰戜絾涓嶉渶瑕佸欢闀匡細鐩存帴鏍囪
				audioFilters = append(audioFilters,
					fmt.Sprintf("[%d:a]acopy[a%d]", i, i))
			}
		}

		// 闊抽浜ゅ弶娣″叆娣″嚭锛堥伩鍏嶈浆鍦烘椂闈欓煶锛?
		// 瀵规墍鏈夌浉閭婚煶棰戦兘搴旂敤 acrossfade锛宼ype=none 鏃朵娇鐢?0 绉掓椂闀?
		var audioCrossfades []string

		for i := 0; i < len(inputPaths)-1; i++ {
			// 榛樿杞満鏃堕暱
			transitionDuration := 1.0
			if clips[i].Transition != nil {
				if tType, ok := clips[i].Transition["type"].(string); ok {
					if strings.ToLower(tType) == "none" || tType == "" {
						// none 杞満浣跨敤 0 绉?
						transitionDuration = 0.0
					}
				}
				// 鍙湁闈?none 杞満鎵嶈鍙栬嚜瀹氫箟鏃堕暱
				if transitionDuration > 0 {
					if tDuration, ok := clips[i].Transition["duration"].(float64); ok && tDuration > 0 {
						transitionDuration = tDuration
					}
				}
			}

			var inputLabel, outputLabel string
			if i == 0 {
				inputLabel = "[a0][a1]"
			} else {
				inputLabel = fmt.Sprintf("[ax%02d][a%d]", i-1, i+1)
			}

			if i == len(inputPaths)-2 {
				outputLabel = "[outa]"
			} else {
				outputLabel = fmt.Sprintf("[ax%02d]", i)
			}

			// acrossfade: d=杞満鏃堕暱锛宑1=绗竴涓煶棰戞贰鍑烘洸绾匡紝c2=绗簩涓煶棰戞贰鍏ユ洸绾?
			// 0 绉掓椂闀垮疄鐜版棤缂濋煶棰戞嫾鎺?
			audioCrossfades = append(audioCrossfades,
				fmt.Sprintf("%sacrossfade=d=%.2f:c1=tri:c2=tri%s", inputLabel, transitionDuration, outputLabel))

			f.log.Infow("Audio crossfade",
				"clip_index", i,
				"duration", transitionDuration)
		}

		// 鏋勫缓瀹屾暣婊ら暅锛氶煶棰戝鐞?+ 闊抽浜ゅ弶娣″叆娣″嚭
		var allAudioFilters []string
		allAudioFilters = append(allAudioFilters, audioFilters...)
		allAudioFilters = append(allAudioFilters, audioCrossfades...)
		fullFilter = filterComplex + ";" + strings.Join(allAudioFilters, ";")
	} else {
		// 鎵€鏈夎棰戦兘鏃犻煶棰戞祦锛屽彧澶勭悊瑙嗛
		fullFilter = filterComplex
	}

	// 鏋勫缓瀹屾暣鍛戒护
	args = append(args,
		"-filter_complex", fullFilter,
		"-map", "[outv]",
	)

	// 浠呭湪鏈変换浣曢煶棰戞椂鏄犲皠闊抽杈撳嚭
	if hasAnyAudio {
		args = append(args, "-map", "[outa]")
	}

	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
	)

	// 浠呭湪鏈変换浣曢煶棰戞椂璁剧疆闊抽缂栫爜鍙傛暟
	if hasAnyAudio {
		args = append(args,
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	args = append(args,
		"-y",
		outputPath,
	)

	f.log.Infow("Running FFmpeg with transitions", "filter", fullFilter, "has_any_audio", hasAnyAudio)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("FFmpeg xfade failed", "error", err, "output", string(output))
		return fmt.Errorf("ffmpeg xfade failed: %w, output: %s", err, string(output))
	}

	f.log.Infow("Video merged with transitions successfully")
	return nil
}

func (f *FFmpeg) mapTransitionType(transType string) string {
	// 灏嗗墠绔紶鍏ョ殑杞満绫诲瀷鏄犲皠涓篎Fmpeg xfade鏀寔鐨勭被鍨?
	// FFmpeg xfade鏀寔鐨勫畬鏁磋浆鍦哄垪琛? https://ffmpeg.org/ffmpeg-filters.html#xfade
	switch strings.ToLower(transType) {
	// 娣″叆娣″嚭绫?
	case "fade", "fadein", "fadeout":
		return "fade"
	case "fadeblack":
		return "fadeblack"
	case "fadewhite":
		return "fadewhite"
	case "fadegrays":
		return "fadegrays"

	// 婊戝姩绫?
	case "slideleft":
		return "slideleft"
	case "slideright":
		return "slideright"
	case "slideup":
		return "slideup"
	case "slidedown":
		return "slidedown"

	// 鎿﹂櫎绫?
	case "wipeleft":
		return "wipeleft"
	case "wiperight":
		return "wiperight"
	case "wipeup":
		return "wipeup"
	case "wipedown":
		return "wipedown"

	// 鍦嗗舰绫?
	case "circleopen":
		return "circleopen"
	case "circleclose":
		return "circleclose"

	// 鐭╁舰鎵撳紑/鍏抽棴绫?
	case "horzopen":
		return "horzopen"
	case "horzclose":
		return "horzclose"
	case "vertopen":
		return "vertopen"
	case "vertclose":
		return "vertclose"

	// 鍏朵粬鐗规晥
	case "dissolve":
		return "dissolve"
	case "distance":
		return "distance"
	case "pixelize":
		return "pixelize"

	default:
		return "fade" // 榛樿娣″叆娣″嚭
	}
}

func (f *FFmpeg) hasAudioStream(videoPath string) bool {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	result := strings.TrimSpace(string(output))
	return result == "audio"
}

func (f *FFmpeg) getVideoResolution(videoPath string) (int, int) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Warnw("Failed to get video resolution", "path", videoPath, "error", err)
		return 1920, 1080 // 榛樿鍒嗚鲸鐜?
	}

	result := strings.TrimSpace(string(output))
	parts := strings.Split(result, ",")
	if len(parts) != 2 {
		f.log.Warnw("Invalid resolution format", "output", result)
		return 1920, 1080
	}

	var width, height int
	fmt.Sscanf(parts[0], "%d", &width)
	fmt.Sscanf(parts[1], "%d", &height)

	if width <= 0 || height <= 0 {
		return 1920, 1080
	}

	return width, height
}

// GetVideoDuration 鑾峰彇瑙嗛鏃堕暱锛堢锛?
func (f *FFmpeg) GetVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("Failed to get video duration", "path", videoPath, "error", err)
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	result := strings.TrimSpace(string(output))
	var duration float64
	_, err = fmt.Sscanf(result, "%f", &duration)
	if err != nil {
		f.log.Errorw("Failed to parse duration", "output", result, "error", err)
		return 0, fmt.Errorf("parse duration failed: %w", err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("invalid duration: %f", duration)
	}

	return duration, nil
}

func (f *FFmpeg) copyFile(src, dst string) error {
  // Pure-Go copy (Windows/macOS/Linux). Avoids shell commands like `cp`.
  if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
    return fmt.Errorf("failed to create dst directory: %w", err)
  }

  in, err := os.Open(src)
  if err != nil {
    return fmt.Errorf("failed to open src: %w", err)
  }
  defer in.Close()

  tmpDst := dst + ".tmp"
  out, err := os.Create(tmpDst)
  if err != nil {
    return fmt.Errorf("failed to create tmp dst: %w", err)
  }

  _, copyErr := io.Copy(out, in)
  closeErr := out.Close()
  if copyErr != nil {
    _ = os.Remove(tmpDst)
    return fmt.Errorf("copy failed: %w", copyErr)
  }
  if closeErr != nil {
    _ = os.Remove(tmpDst)
    return fmt.Errorf("failed to close tmp dst: %w", closeErr)
  }

  _ = os.Remove(dst)
  if err := os.Rename(tmpDst, dst); err != nil {
    _ = os.Remove(tmpDst)
    return fmt.Errorf("failed to move tmp dst into place: %w", err)
  }
  return nil
}

func (f *FFmpeg) cleanup(paths []string) {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			f.log.Warnw("Failed to cleanup file", "path", path, "error", err)
		}
	}
}

func (f *FFmpeg) CleanupTempDir() error {
	return os.RemoveAll(f.tempDir)
}

// ExtractAudio 浠庤棰戞枃浠朵腑鎻愬彇闊抽杞ㄩ亾
// 杩斿洖鎻愬彇鐨勯煶棰戞枃浠惰矾寰?
func (f *FFmpeg) ExtractAudio(videoURL, outputPath string) (string, error) {
	f.log.Infow("Extracting audio from video", "url", videoURL, "output", outputPath)

	// 涓嬭浇瑙嗛鏂囦欢
	downloadPath := filepath.Join(f.tempDir, fmt.Sprintf("video_%d.mp4", time.Now().Unix()))
	localVideoPath, err := f.downloadVideo(videoURL, downloadPath)
	if err != nil {
		return "", fmt.Errorf("failed to download video: %w", err)
	}
	defer os.Remove(localVideoPath)

	// 妫€鏌ヨ棰戞槸鍚︽湁闊抽娴?
	if !f.hasAudioStream(localVideoPath) {
		f.log.Warnw("Video has no audio stream, generating silence", "video", videoURL)
		// 鑾峰彇瑙嗛鏃堕暱
		duration, err := f.GetVideoDuration(localVideoPath)
		if err != nil {
			return "", fmt.Errorf("failed to get video duration: %w", err)
		}
		// 鐢熸垚闈欓煶闊抽鏂囦欢
		return f.generateSilence(outputPath, duration)
	}

	// 纭繚杈撳嚭鐩綍瀛樺湪
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 浣跨敤FFmpeg鎻愬彇闊抽
	// -vn: 绂佺敤瑙嗛
	// -acodec: 闊抽缂栫爜鍣?
	// -ar: 闊抽閲囨牱鐜?
	// -ac: 闊抽澹伴亾鏁?
	// -ab: 闊抽姣旂壒鐜?
	cmd := exec.Command("ffmpeg",
		"-i", localVideoPath,
		"-vn",
		"-acodec", "aac",
		"-ar", "44100",
		"-ac", "2",
		"-ab", "128k",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("FFmpeg audio extraction failed", "error", err, "output", string(output))
		return "", fmt.Errorf("ffmpeg audio extraction failed: %w, output: %s", err, string(output))
	}

	f.log.Infow("Audio extracted successfully", "output", outputPath)
	return outputPath, nil
}

// generateSilence 鐢熸垚鎸囧畾鏃堕暱鐨勯潤闊抽煶棰戞枃浠?
func (f *FFmpeg) generateSilence(outputPath string, duration float64) (string, error) {
	f.log.Infow("Generating silence audio", "duration", duration, "output", outputPath)

	// 纭繚杈撳嚭鐩綍瀛樺湪
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 浣跨敤FFmpeg鐢熸垚闈欓煶
	// -f lavfi: 浣跨敤lavfi锛坙ibavfilter锛夎緭鍏?
	// -i anullsrc: 鐢熸垚闈欓煶闊抽婧?
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("anullsrc=channel_layout=stereo:sample_rate=44100"),
		"-t", fmt.Sprintf("%.2f", duration),
		"-acodec", "aac",
		"-ab", "128k",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		f.log.Errorw("FFmpeg silence generation failed", "error", err, "output", string(output))
		return "", fmt.Errorf("ffmpeg silence generation failed: %w, output: %s", err, string(output))
	}

	f.log.Infow("Silence audio generated successfully", "output", outputPath)
	return outputPath, nil
}
