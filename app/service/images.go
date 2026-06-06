package service

import (
	"chat2api/app/common"
	"chat2api/app/types/responses"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ImagesGenerationsRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	Size           string `json:"size"`
	Quality        string `json:"quality"`
	ResponseFormat string `json:"response_format"`
}

type ImagesEditsRequest struct {
	Model          string      `json:"model"`
	Prompt         string      `json:"prompt"`
	N              int         `json:"n"`
	Size           string      `json:"size"`
	Quality        string      `json:"quality"`
	ResponseFormat string      `json:"response_format"`
	Image          interface{} `json:"image"`
	Images         interface{} `json:"images"`
	ImageURL       interface{} `json:"image_url"`
}

type ImagesResponse struct {
	Created int64            `json:"created"`
	Data    []ImagesRespItem `json:"data"`
}

type ImagesRespItem struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

func ImagesGenerations(c *gin.Context) {
	req := &ImagesGenerationsRequest{}
	if err := c.BindJSON(req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid parameter", nil)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "prompt is required", nil)
		return
	}
	result, err := runOpenAIImages(c, req.Prompt, nil, imageOptions{
		Model:          req.Model,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		ResponseFormat: req.ResponseFormat,
	})
	if err != nil {
		common.ErrorResponse(c, http.StatusBadGateway, "image generation failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func ImagesEdits(c *gin.Context) {
	if strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data") {
		handleMultipartImagesEdits(c)
		return
	}
	req := &ImagesEditsRequest{}
	if err := c.BindJSON(req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid parameter", nil)
		return
	}
	images := imageValuesFromJSON(req.Image, req.Images, req.ImageURL)
	if len(images) == 0 {
		common.ErrorResponse(c, http.StatusBadRequest, "image is required", nil)
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "prompt is required", nil)
		return
	}
	result, err := runOpenAIImages(c, req.Prompt, images, imageOptions{
		Model:          req.Model,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		ResponseFormat: req.ResponseFormat,
	})
	if err != nil {
		common.ErrorResponse(c, http.StatusBadGateway, "image edit failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func handleMultipartImagesEdits(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid multipart form", err.Error())
		return
	}
	prompt := strings.TrimSpace(c.PostForm("prompt"))
	if prompt == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "prompt is required", nil)
		return
	}
	images, err := multipartImageValues(c)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid image", err.Error())
		return
	}
	if len(images) == 0 {
		common.ErrorResponse(c, http.StatusBadRequest, "image is required", nil)
		return
	}
	result, err := runOpenAIImages(c, prompt, images, imageOptions{
		Model:          c.PostForm("model"),
		N:              intFromString(c.PostForm("n")),
		Size:           c.PostForm("size"),
		Quality:        c.PostForm("quality"),
		ResponseFormat: c.PostForm("response_format"),
	})
	if err != nil {
		common.ErrorResponse(c, http.StatusBadGateway, "image edit failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

type imageOptions struct {
	Model          string
	N              int
	Size           string
	Quality        string
	ResponseFormat string
}

func runOpenAIImages(c *gin.Context, prompt string, images []string, opts imageOptions) (*ImagesResponse, error) {
	n := opts.N
	if n <= 0 {
		n = 1
	}
	if n > 4 {
		return nil, fmt.Errorf("n must be less than or equal to 4")
	}
	tool := normalizeCodexImageTool(responses.Tool{
		Type:         "image_generation",
		Model:        opts.Model,
		Size:         opts.Size,
		Quality:      opts.Quality,
		OutputFormat: outputFormatFromResponseFormat(opts.ResponseFormat),
	}, len(images) > 0)
	items := make([]ImagesRespItem, 0, n)
	for i := 0; i < n; i++ {
		completed, err := collectOpenAIImageResponse(c, prompt, images, tool)
		if err != nil {
			return nil, err
		}
		b64, revised := imageResultFromCompleted(completed)
		if b64 == "" {
			return nil, fmt.Errorf("upstream completed without generating images")
		}
		items = append(items, openAIImageItem(b64, revised, opts.ResponseFormat))
	}
	return &ImagesResponse{Created: time.Now().Unix(), Data: items}, nil
}

func collectOpenAIImageResponse(c *gin.Context, prompt string, images []string, tool responses.Tool) (map[string]interface{}, error) {
	payload := codexResponsesPayload{
		Model:             codexResponsesModel,
		Instructions:      codexResponsesInstructions,
		Store:             false,
		Input:             codexImageInput(prompt, images),
		Tools:             []responses.Tool{tool},
		ToolChoice:        map[string]string{"type": "image_generation"},
		Stream:            true,
		ParallelToolCalls: false,
	}
	resp, accessToken, err := sendCodexResponsesRequest(c, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("upstream returned status %d: %s (token=%s)", resp.StatusCode, string(body), accessToken)
	}
	return collectCodexResponse(resp.Body)
}

func openAIImageItem(b64 string, revisedPrompt string, responseFormat string) ImagesRespItem {
	if strings.EqualFold(strings.TrimSpace(responseFormat), "url") {
		return ImagesRespItem{URL: "data:image/png;base64," + b64, RevisedPrompt: revisedPrompt}
	}
	return ImagesRespItem{B64JSON: b64, RevisedPrompt: revisedPrompt}
}

func outputFormatFromResponseFormat(responseFormat string) string {
	responseFormat = strings.TrimSpace(strings.ToLower(responseFormat))
	if responseFormat == "" || responseFormat == "b64_json" || responseFormat == "url" {
		return "png"
	}
	return responseFormat
}

func imageResultFromCompleted(value interface{}) (string, string) {
	switch v := value.(type) {
	case map[string]interface{}:
		result := strings.TrimSpace(responseStringValue(v["result"], ""))
		if result != "" {
			return stripImageDataURL(result), responseStringValue(v["revised_prompt"], "")
		}
		for _, key := range []string{"b64_json", "image", "url"} {
			if text := strings.TrimSpace(responseStringValue(v[key], "")); text != "" {
				return stripImageDataURL(text), responseStringValue(v["revised_prompt"], "")
			}
		}
		for _, child := range v {
			if b64, revised := imageResultFromCompleted(child); b64 != "" {
				if revised == "" {
					revised = responseStringValue(v["revised_prompt"], "")
				}
				return b64, revised
			}
		}
	case []interface{}:
		for _, child := range v {
			if b64, revised := imageResultFromCompleted(child); b64 != "" {
				return b64, revised
			}
		}
	}
	return "", ""
}

func stripImageDataURL(value string) string {
	value = strings.TrimSpace(value)
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:image/") && comma >= 0 {
		return strings.TrimSpace(value[comma+1:])
	}
	return value
}

func imageValuesFromJSON(values ...interface{}) []string {
	images := make([]string, 0)
	for _, value := range values {
		collectImageValues(value, &images)
	}
	return images
}

func collectImageValues(value interface{}, images *[]string) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			*images = append(*images, strings.TrimSpace(v))
		}
	case []interface{}:
		for _, item := range v {
			collectImageValues(item, images)
		}
	case map[string]interface{}:
		for _, key := range []string{"image_url", "url", "base64", "b64_json"} {
			if text := responseStringValue(v[key], ""); strings.TrimSpace(text) != "" {
				*images = append(*images, strings.TrimSpace(text))
				return
			}
		}
		if source, ok := v["source"].(map[string]interface{}); ok {
			collectImageValues(source["data"], images)
		}
	}
}

func multipartImageValues(c *gin.Context) ([]string, error) {
	form := c.Request.MultipartForm
	if form == nil {
		return nil, nil
	}
	images := make([]string, 0)
	for _, key := range []string{"image", "image[]", "images", "images[]"} {
		for _, header := range form.File[key] {
			file, err := header.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				return nil, err
			}
			if len(data) > 0 {
				images = append(images, base64.StdEncoding.EncodeToString(data))
			}
		}
	}
	for _, key := range []string{"image_url", "image_url[]"} {
		for _, value := range form.Value[key] {
			if strings.TrimSpace(value) != "" {
				images = append(images, strings.TrimSpace(value))
			}
		}
	}
	return images, nil
}

func intFromString(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(value, "%d", &n)
	return n
}
