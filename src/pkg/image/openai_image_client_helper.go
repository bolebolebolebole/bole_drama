package image

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

func (c *OpenAIImageClient) urlToBase64(urlStr string) (string, error) {
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
		// ok
	case "image/png":
		// ok
	case "image/webp":
		// ok
	default:
		// default to png if unknown or allow others
	}

	base64Str := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}
