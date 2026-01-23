package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/drama-generator/backend/pkg/logger"
	"github.com/gin-gonic/gin"
)

type VideoProxyHandler struct {
	log *logger.Logger
}

func NewVideoProxyHandler(log *logger.Logger) *VideoProxyHandler {
	return &VideoProxyHandler{
		log: log,
	}
}

func (h *VideoProxyHandler) ProxyVideo(c *gin.Context) {
	encodedURL := c.Query("url")
	if encodedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	decodedURL, err := url.QueryUnescape(encodedURL)
	if err != nil {
		h.log.Errorw("Failed to decode URL", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url parameter"})
		return
	}

	if decodedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter cannot be empty"})
		return
	}

	parsedURL, err := url.Parse(decodedURL)
	if err != nil {
		h.log.Errorw("Failed to parse URL", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url format"})
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only http and https URLs are supported"})
		return
	}

	req, err := http.NewRequest("GET", decodedURL, nil)
	if err != nil {
		h.log.Errorw("Failed to create request", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	for key, values := range c.Request.Header {
		if strings.HasPrefix(strings.ToLower(key), "accept") ||
			strings.HasPrefix(strings.ToLower(key), "user-agent") {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		h.log.Errorw("Failed to fetch video", "error", err, "url", decodedURL)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch video"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.log.Warnw("Video fetch returned non-OK status", "status", resp.StatusCode, "url", decodedURL)
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("upstream returned status %d", resp.StatusCode)})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	contentLength := resp.ContentLength

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"video%s\"", getFileExtensionFromURL(decodedURL)))
	if contentLength > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Range, Content-Type")

	c.Writer.WriteHeader(http.StatusOK)

	written, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		h.log.Warnw("Error streaming video", "error", err, "written", written)
	}

	h.log.Infow("Video proxy completed", "url", decodedURL, "bytes", written)
}

func (h *VideoProxyHandler) ProxyVideoRange(c *gin.Context) {
	encodedURL := c.Query("url")
	if encodedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter is required"})
		return
	}

	decodedURL, err := url.QueryUnescape(encodedURL)
	if err != nil {
		h.log.Errorw("Failed to decode URL", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url parameter"})
		return
	}

	if decodedURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter cannot be empty"})
		return
	}

	rangeHeader := c.GetHeader("Range")

	req, err := http.NewRequest("GET", decodedURL, nil)
	if err != nil {
		h.log.Errorw("Failed to create request", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	for key, values := range c.Request.Header {
		if strings.HasPrefix(strings.ToLower(key), "accept") ||
			strings.HasPrefix(strings.ToLower(key), "user-agent") {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		h.log.Errorw("Failed to fetch video", "error", err, "url", decodedURL)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch video"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		h.log.Warnw("Video fetch returned non-OK status", "status", resp.StatusCode, "url", decodedURL)
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("upstream returned status %d", resp.StatusCode)})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", resp.Header.Get("Accept-Ranges"))

	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		c.Header("Content-Range", contentRange)
	}

	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Header("Content-Length", contentLength)
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Range, Content-Type")

	c.Writer.WriteHeader(resp.StatusCode)

	written, err := io.Copy(c.Writer, resp.Body)
	if err != nil {
		h.log.Warnw("Error streaming video", "error", err, "written", written)
	}

	h.log.Infow("Video range proxy completed", "url", decodedURL, "bytes", written, "range", rangeHeader)
}

func getFileExtensionFromURL(url string) string {
	if idx := strings.LastIndex(url, "."); idx != -1 {
		ext := url[idx:]
		if qIdx := strings.Index(ext, "?"); qIdx != -1 {
			ext = ext[:qIdx]
		}
		if len(ext) <= 5 && len(ext) > 0 {
			return ext
		}
	}
	return ".mp4"
}
