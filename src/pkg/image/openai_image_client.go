package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type OpenAIImageClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

type DALLERequest struct {
	Model   string   `json:"model"`
	Prompt  string   `json:"prompt"`
	Size    string   `json:"size,omitempty"`
	Quality string   `json:"quality,omitempty"`
	N       int      `json:"n"`
	Image   []string `json:"image,omitempty"`
}

type DALLEResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

func NewOpenAIImageClient(baseURL, apiKey, model, endpoint string) *OpenAIImageClient {
	if endpoint == "" {
		endpoint = "/v1/images/generations"
	}
	return &OpenAIImageClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *OpenAIImageClient) GenerateImage(prompt string, opts ...ImageOption) (*ImageResult, error) {
	options := &ImageOptions{
		Size:    "1920x1920",
		Quality: "standard",
	}

	for _, opt := range opts {
		opt(options)
	}

	model := c.Model
	if options.Model != "" {
		model = options.Model
	}

	size := options.Size
	if size == "" {
		size = "1920x1920"
	} else if strings.Contains(size, "x") {
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			width, _ := strconv.Atoi(parts[0])
			height, _ := strconv.Atoi(parts[1])
			totalPixels := width * height
			if totalPixels >= 1024*1024 && totalPixels <= 4096*4096 {
				size = "auto"
			} else if totalPixels > 4096*4096 {
				size = "4K"
			}
		}
	}

	// Process reference images
	var images []string
	if len(options.ReferenceImages) > 0 {
		images = options.ReferenceImages
	}

	reqBody := DALLERequest{
		Model:   model,
		Prompt:  prompt,
		Size:    size,
		Quality: options.Quality,
		N:       1,
		Image:   images,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.BaseURL + c.Endpoint
	fmt.Printf("[OpenAI Image] Request URL: %s\n", url)
	fmt.Printf("[OpenAI Image] Request Body: %s\n", string(jsonData))

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("OpenAI API Response: %s\n", string(body))

	var result DALLEResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no image generated, response: %s", string(body))
	}

	return &ImageResult{
		Status:    "completed",
		ImageURL:  result.Data[0].URL,
		Completed: true,
	}, nil
}

func (c *OpenAIImageClient) GetTaskStatus(taskID string) (*ImageResult, error) {
	return nil, fmt.Errorf("not supported for OpenAI/DALL-E")
}
