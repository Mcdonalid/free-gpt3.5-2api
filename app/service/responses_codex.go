package service

import (
	"bufio"
	"bytes"
	"chat2api/app/chatgpt_backend"
	"chat2api/app/common"
	"chat2api/app/types/responses"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aurorax-neo/tls_client_httpi"
	"github.com/gin-gonic/gin"
)

const codexResponsesModel = "gpt-5.5"
const codexResponsesInstructions = "Use the image_generation tool to create exactly one image for the user's request. Return the generated image result."

type codexResponsesPayload struct {
	Model        string                   `json:"model"`
	Instructions string                   `json:"instructions"`
	Store        bool                     `json:"store"`
	Input        []map[string]interface{} `json:"input"`
	Tools        []responses.Tool         `json:"tools"`
	ToolChoice   map[string]string        `json:"tool_choice"`
	Stream       bool                     `json:"stream"`
}

func runCodexImageResponses(c *gin.Context, apiReq *responses.ApiReq) error {
	prompt := extractResponsesPrompt(apiReq.Input)
	if strings.TrimSpace(prompt) == "" {
		common.ErrorResponse(c, http.StatusBadRequest, "input text is required", nil)
		return nil
	}
	images := extractResponsesImages(apiReq.Input)
	tool := normalizeCodexImageTool(firstResponsesImageGenerationTool(apiReq.Tools), len(images) > 0)
	payload := codexResponsesPayload{
		Model:        codexResponsesModel,
		Instructions: codexResponsesInstructions,
		Store:        false,
		Input:        codexImageInput(prompt, images),
		Tools:        []responses.Tool{tool},
		ToolChoice:   map[string]string{"type": "image_generation"},
		Stream:       true,
	}
	resp, accessToken, err := sendCodexResponsesRequest(c, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if handleResponseError(c, resp, accessToken) {
		return nil
	}
	if apiReq.Stream {
		return streamCodexResponses(c, resp)
	}
	completed, err := collectCodexResponse(resp.Body)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, completed)
	return nil
}

func sendCodexResponsesRequest(c *gin.Context, payload codexResponsesPayload) (*http.Response, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	backend, err := chatgpt_backend.New(c.Request.Header.Get("Authorization"), chatgpt_backend.Retry())
	if err != nil {
		return nil, "", err
	}
	url := backend.BaseURL + "/backend-api/codex/responses"
	headers, cookies := backend.Headers(url)
	headers.Set("accept", "text/event-stream")
	headers.Set("content-type", "application/json")
	if backend.AccAuth == "" {
		return nil, backend.AccAuth, fmt.Errorf("codex responses endpoint requires access token auth")
	}
	resp, err := backend.HTTP.Request(tls_client_httpi.POST, url, headers, cookies, bytes.NewBuffer(body))
	if err != nil {
		return nil, backend.AccAuth, fmt.Errorf("upstream codex responses request failed: %w", err)
	}
	return resp, backend.AccAuth, nil
}

func streamCodexResponses(c *gin.Context, resp *http.Response) error {
	c.Header("Content-Type", "text/event-stream")
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "data: [DONE]" {
			if _, err := c.Writer.WriteString("data: [DONE]\n\n"); err != nil {
				return err
			}
			break
		}
		if !strings.HasPrefix(trimmed, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		if !isPublicCodexResponseEvent(payload) {
			continue
		}
		if _, err := c.Writer.WriteString("data: " + payload + "\n\n"); err != nil {
			return err
		}
		c.Writer.Flush()
	}
	c.Writer.Flush()
	return nil
}

func isPublicCodexResponseEvent(payload string) bool {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return false
	}
	switch responseStringValue(event["type"], "") {
	case "response.created",
		"response.output_item.added",
		"response.output_text.delta",
		"response.output_text.done",
		"response.output_item.done",
		"response.completed",
		"response.failed",
		"response.incomplete":
		return true
	default:
		return false
	}
}

func collectCodexResponse(body io.Reader) (map[string]interface{}, error) {
	reader := bufio.NewReader(body)
	var completed map[string]interface{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if event["type"] == "response.completed" {
			if response, ok := event["response"].(map[string]interface{}); ok {
				completed = response
			}
		}
	}
	if completed == nil {
		return nil, fmt.Errorf("codex response generation failed")
	}
	return completed, nil
}

func firstResponsesImageGenerationTool(tools []responses.Tool) responses.Tool {
	for _, tool := range tools {
		if strings.TrimSpace(tool.Type) == "image_generation" {
			return tool
		}
	}
	return responses.Tool{Type: "image_generation"}
}

func normalizeCodexImageTool(tool responses.Tool, hasImages bool) responses.Tool {
	tool.Type = "image_generation"
	if strings.TrimSpace(tool.Model) == "" {
		tool.Model = "gpt-image-2"
	}
	if strings.TrimSpace(tool.Action) == "" {
		if hasImages {
			tool.Action = "edit"
		} else {
			tool.Action = "generate"
		}
	}
	if strings.TrimSpace(tool.Size) == "" {
		tool.Size = "1024x1024"
	}
	if strings.TrimSpace(tool.Quality) == "" {
		tool.Quality = "auto"
	}
	if strings.TrimSpace(tool.OutputFormat) == "" {
		tool.OutputFormat = "png"
	}
	return tool
}

func codexImageInput(prompt string, images []string) []map[string]interface{} {
	content := []map[string]interface{}{{"type": "input_text", "text": prompt}}
	for _, image := range images {
		content = append(content, map[string]interface{}{"type": "input_image", "image_url": normalizeImageDataURL(image)})
	}
	return []map[string]interface{}{{"role": "user", "content": content}}
}

func normalizeImageDataURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "data:image/") {
		return value
	}
	if value != "" && len(value) < 512 && !strings.ContainsAny(value, "\r\n") {
		if data, err := os.ReadFile(filepath.Clean(value)); err == nil && len(data) > 0 {
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
		}
	}
	return "data:image/png;base64," + value
}

func extractResponsesPrompt(input interface{}) string {
	parts := make([]string, 0)
	collectResponsesText(input, &parts)
	return strings.TrimSpace(strings.Join(parts, ""))
}

func collectResponsesText(value interface{}, parts *[]string) {
	switch v := value.(type) {
	case string:
		*parts = append(*parts, v)
	case []interface{}:
		for _, item := range v {
			collectResponsesText(item, parts)
		}
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok && text != "" {
			*parts = append(*parts, text)
			return
		}
		if content, ok := v["content"]; ok {
			collectResponsesText(content, parts)
		}
	}
}

func extractResponsesImages(input interface{}) []string {
	images := make([]string, 0)
	collectResponsesImages(input, &images)
	return images
}

func collectResponsesImages(value interface{}, images *[]string) {
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			collectResponsesImages(item, images)
		}
	case map[string]interface{}:
		partType := strings.TrimSpace(responseStringValue(v["type"], ""))
		if partType == "input_image" || partType == "image" || partType == "image_url" || v["image_url"] != nil {
			if image := responseImageValue(v); image != "" {
				*images = append(*images, image)
			}
		}
		if content, ok := v["content"]; ok {
			collectResponsesImages(content, images)
		}
	}
}

func responseImageValue(item map[string]interface{}) string {
	for _, key := range []string{"image_url", "url", "base64", "b64_json"} {
		value, ok := item[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
		if obj, ok := value.(map[string]interface{}); ok {
			for _, nested := range []string{"url", "image_url", "base64", "b64_json"} {
				if text, ok := obj[nested].(string); ok {
					return strings.TrimSpace(text)
				}
			}
		}
	}
	if source, ok := item["source"].(map[string]interface{}); ok && responseStringValue(source["type"], "") == "base64" {
		return strings.TrimSpace(responseStringValue(source["data"], ""))
	}
	return ""
}
