package image

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type VolcEngineImageClient struct {
	BaseURL       string
	APIKey        string
	Model         string
	Endpoint      string
	QueryEndpoint string
	HTTPClient    *http.Client
}

type VolcEngineImageRequest struct {
	Model                     string   `json:"model"`
	Prompt                    string   `json:"prompt"`
	Image                     []string `json:"image,omitempty"`
	SequentialImageGeneration string   `json:"sequential_image_generation,omitempty"`
	Size                      string   `json:"size,omitempty"`   // OpenAI 兼容 API 使用 size 参数
	Width                     int      `json:"width,omitempty"`  // 原生 API 使用 width 参数
	Height                    int      `json:"height,omitempty"` // 原生 API 使用 height 参数
	Watermark                 bool     `json:"watermark"`        // 移除omitempty以确保false值被发送
}

type VolcEngineImageResponse struct {
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Data    []struct {
		URL  string `json:"url"`
		Size string `json:"size"`
	} `json:"data"`
	Usage struct {
		GeneratedImages int `json:"generated_images"`
		OutputTokens    int `json:"output_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"usage"`
	Error interface{} `json:"error,omitempty"`
}

func NewVolcEngineImageClient(baseURL, apiKey, model, endpoint, queryEndpoint string) *VolcEngineImageClient {
	if endpoint == "" {
		// 使用 OpenAI 兼容的端点，支持自定义尺寸参数
		endpoint = "/v1/images/generations"
	}
	if queryEndpoint == "" {
		queryEndpoint = endpoint
	}
	return &VolcEngineImageClient{
		BaseURL:       baseURL,
		APIKey:        apiKey,
		Model:         model,
		Endpoint:      endpoint,
		QueryEndpoint: queryEndpoint,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
	}
}

func (c *VolcEngineImageClient) GenerateImage(prompt string, opts ...ImageOption) (*ImageResult, error) {
	options := &ImageOptions{
		Size:    "1024x1024",
		Quality: "standard",
	}

	for _, opt := range opts {
		opt(options)
	}

	model := c.Model
	if options.Model != "" {
		model = options.Model
	}

	promptText := prompt
	if options.NegativePrompt != "" {
		promptText += fmt.Sprintf(". Negative: %s", options.NegativePrompt)
	}

	// 处理尺寸参数 - 优先使用明确的width/height
	width := 1024
	height := 1024
	size := options.Size

	// 如果有明确的width和height，使用它们
	if options.Width > 0 && options.Height > 0 {
		width = options.Width
		height = options.Height
		size = fmt.Sprintf("%dx%d", width, height)
	} else if size != "" && strings.Contains(size, "x") {
		// 如果没有明确的width和height，但size字符串包含"x"，则解析size
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, errW := strconv.Atoi(parts[0])
			h, errH := strconv.Atoi(parts[1])
			if errW == nil && errH == nil {
				width = w
				height = h
			}
		}
	}

	if size == "" {
		size = "1024x1024"
	}

	fmt.Printf("[VolcEngine Image] Debug: options.Size = %s, options.Width = %d, options.Height = %d, final size = %s, parsed width=%d, height=%d\n", options.Size, options.Width, options.Height, size, width, height)

	images := options.ReferenceImages
	if len(images) > 0 {
		var processedImages []string
		for _, imgURL := range images {
			isLocal := strings.Contains(imgURL, "localhost") ||
				strings.Contains(imgURL, "127.0.0.1") ||
				strings.Contains(imgURL, "192.168.") ||
				strings.Contains(imgURL, "10.") ||
				strings.Contains(imgURL, "172.")

			if isLocal {
				fmt.Printf("[VolcEngine Image] Skipping local reference image (cannot be accessed by external API): %s\n", imgURL)
			} else {
				processedImages = append(processedImages, imgURL)
			}
		}
		images = processedImages
	}

	reqBody := VolcEngineImageRequest{
		Model:                     model,
		Prompt:                    promptText,
		Image:                     images,
		SequentialImageGeneration: "disabled",
		Size:                      size,   // 使用 size 字符串参数（火山引擎原生API也支持）
		Width:                     width,  // 明确指定宽高（火山引擎原生API支持）
		Height:                    height, // 明确指定宽高（火山引擎原生API支持）
		Watermark:                 false,  // 禁用水印
	}

	// 详细的日志输出，用于调试水印问题
	fmt.Printf("[VolcEngine Image] Watermark Parameter Value: %v\n", reqBody.Watermark)
	fmt.Printf("[VolcEngine Image] Watermark Type: %T\n", reqBody.Watermark)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 简单的URL拼接，正确处理斜杠和路径
	url := c.BaseURL

	// 确保BaseURL以/结尾
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	// 如果Endpoint以/开头，去掉/（因为BaseURL已经以/结尾）
	endpoint := c.Endpoint
	if strings.HasPrefix(endpoint, "/") {
		endpoint = endpoint[1:]
	}
	url += endpoint

	fmt.Printf("[VolcEngine Image] BaseURL: %s, Endpoint: %s, Final URL: %s\n", c.BaseURL, c.Endpoint, url)
	fmt.Printf("[VolcEngine Image] Request Body: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	fmt.Printf("VolcEngine Image API Response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result VolcEngineImageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("volcengine error: %v", result.Error)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	// Parse actual dimensions from response
	if result.Data[0].Size != "" && strings.Contains(result.Data[0].Size, "x") {
		parts := strings.Split(result.Data[0].Size, "x")
		if len(parts) == 2 {
			w, errW := strconv.Atoi(parts[0])
			h, errH := strconv.Atoi(parts[1])
			if errW == nil && errH == nil {
				width = w
				height = h
			}
		}
	}

	return &ImageResult{
		Status:    "completed",
		ImageURL:  result.Data[0].URL,
		Width:     width,
		Height:    height,
		Completed: true,
	}, nil
}

func (c *VolcEngineImageClient) urlToBase64(urlStr string) (string, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(data)
	switch mimeType {
	case "image/jpeg":
	case "image/png":
	case "image/webp":
	default:
	}

	base64Str := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

func (c *VolcEngineImageClient) GetTaskStatus(taskID string) (*ImageResult, error) {
	return nil, fmt.Errorf("not supported for VolcEngine Seedream (synchronous generation)")
}
