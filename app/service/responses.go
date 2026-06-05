package service

import (
	"chat2api/app/chat_backend"
	"chat2api/app/common"
	"chat2api/app/types/completions"
	"chat2api/app/types/responses"
	"chat2api/pkg/logx"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aurorax-neo/tls_client_httpi"
	"github.com/gin-gonic/gin"
)

const toolUnavailableSystemMessage = "This compatibility backend cannot execute local tools, shell commands, web searches, or file operations. Do not claim to have run tools or inspected external resources. If a user asks you to use a tool, say that tool execution is unavailable through this backend."

func Responses(c *gin.Context) {
	apiReq := &responses.ApiReq{}
	if err := c.BindJSON(apiReq); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, "Invalid parameter", nil)
		return
	}
	if hasImageGenerationTool(apiReq.Tools) {
		common.ErrorResponse(c, http.StatusNotImplemented, "responses image_generation tool is not implemented", nil)
		return
	}
	compReq := &completions.ApiReq{
		Model:    responses.NormalizeModel(apiReq.Model),
		Stream:   false,
		Messages: responseMessages(apiReq),
	}
	if len(compReq.Messages) == 0 {
		common.ErrorResponse(c, http.StatusBadRequest, "input text is required", nil)
		return
	}
	result, err := runTextConversation(c, compReq)
	if err != nil {
		logx.WithContext(c.Request.Context()).Error(err.Error())
		common.ErrorResponse(c, http.StatusBadGateway, "", err.Error())
		return
	}
	if result == nil {
		return
	}
	if apiReq.Stream {
		emitResponseEvents(c, compReq.Model, result.Content)
		return
	}
	item := responses.TextOutputItem(responses.MessageID(), result.Content, "completed")
	c.JSON(http.StatusOK, responses.CompletedEvent(responses.ResponseID(), compReq.Model, time.Now().Unix(), []responses.OutputItem{item}).Response)
}

func runTextConversation(c *gin.Context, apiReq *completions.ApiReq) (*chatResult, error) {
	chatReq := chat_backend.BuildChatRequest(apiReq)
	body, err := common.Struct2BytesBuffer(chatReq)
	if err != nil {
		return nil, err
	}
	backend, err := chat_backend.New(c.Request.Header.Get("Authorization"), chat_backend.Retry())
	if err != nil {
		return nil, err
	}
	headers, cookies := backend.Headers(backend.ChatURL)
	headers.Set("accept", "text/event-stream")
	headers.Set("content-type", "application/json")
	headers.Set("openai-sentinel-chat-requirements-token", backend.Auth.Token)
	if backend.Auth.ProofWork.Ospt != "" {
		headers.Set("openai-sentinel-proof-token", backend.Auth.ProofWork.Ospt)
	}
	if backend.Auth.TurnstileToken != "" {
		headers.Set("openai-sentinel-turnstile-token", backend.Auth.TurnstileToken)
	}
	if backend.Auth.SoToken != "" {
		headers.Set("openai-sentinel-so-token", backend.Auth.SoToken)
	}
	resp, err := backend.HTTP.Request(tls_client_httpi.POST, backend.ChatURL, headers, cookies, body)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()
	if handleResponseError(c, resp, backend.AccAuth) {
		return nil, nil
	}
	return handlerResponse(c, apiReq, resp)
}

func emitResponseEvents(c *gin.Context, model string, text string) {
	c.Header("Content-Type", "text/event-stream")
	responseID := responses.ResponseID()
	itemID := responses.MessageID()
	created := time.Now().Unix()
	_, _ = c.Writer.WriteString(responses.SSE(responses.CreatedEvent(responseID, model, created)))
	item := responses.TextOutputItem("", "", "in_progress")
	item.ID = itemID
	_, _ = c.Writer.WriteString(responses.SSE(responses.Event{Type: "response.output_item.added", OutputIndex: 0, Item: &item}))
	_, _ = c.Writer.WriteString(responses.SSE(responses.Event{Type: "response.output_text.delta", ItemID: itemID, OutputIndex: 0, ContentIndex: 0, Delta: text}))
	_, _ = c.Writer.WriteString(responses.SSE(responses.Event{Type: "response.output_text.done", ItemID: itemID, OutputIndex: 0, ContentIndex: 0, Text: text}))
	completedItem := responses.TextOutputItem(itemID, text, "completed")
	_, _ = c.Writer.WriteString(responses.SSE(responses.Event{Type: "response.output_item.done", OutputIndex: 0, Item: &completedItem}))
	_, _ = c.Writer.WriteString(responses.SSE(responses.CompletedEvent(responseID, model, created, []responses.OutputItem{completedItem})))
	_, _ = c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()
}

func responseMessages(req *responses.ApiReq) []completions.ApiMessage {
	messages := make([]completions.ApiMessage, 0)
	if strings.TrimSpace(req.Instructions) != "" {
		messages = append(messages, completions.ApiMessage{Role: "system", Content: strings.TrimSpace(req.Instructions)})
	}
	if hasNonImageTools(req.Tools) {
		messages = append(messages, completions.ApiMessage{Role: "system", Content: toolUnavailableSystemMessage})
	}
	return append(messages, inputMessages(req.Input)...)
}

func inputMessages(input interface{}) []completions.ApiMessage {
	switch v := input.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []completions.ApiMessage{{Role: "user", Content: strings.TrimSpace(v)}}
	case map[string]interface{}:
		return []completions.ApiMessage{{Role: stringValue(v["role"], "user"), Content: messageContentText(v)}}
	case []interface{}:
		messages := make([]completions.ApiMessage, 0, len(v))
		for _, item := range v {
			if part, ok := item.(map[string]interface{}); ok {
				text := messageContentText(part)
				if strings.TrimSpace(text) != "" {
					messages = append(messages, completions.ApiMessage{Role: stringValue(part["role"], "user"), Content: text})
				}
			}
		}
		return messages
	default:
		return nil
	}
}

func messageContentText(item map[string]interface{}) string {
	if text := stringValue(item["text"], ""); text != "" {
		return text
	}
	if text := stringValue(item["content"], ""); text != "" {
		return text
	}
	if content, ok := item["content"].([]interface{}); ok {
		parts := make([]string, 0, len(content))
		for _, raw := range content {
			if part, ok := raw.(map[string]interface{}); ok {
				if text := stringValue(part["text"], ""); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "")
	}
	if isContentPart(item) {
		return stringValue(item["text"], "")
	}
	data, _ := json.Marshal(item)
	return string(data)
}

func isContentPart(item map[string]interface{}) bool {
	switch stringValue(item["type"], "") {
	case "text", "input_text", "output_text":
		return true
	default:
		return false
	}
}

func hasImageGenerationTool(tools []responses.Tool) bool {
	for _, tool := range tools {
		if strings.TrimSpace(tool.Type) == "image_generation" {
			return true
		}
	}
	return false
}

func hasNonImageTools(tools []responses.Tool) bool {
	for _, tool := range tools {
		if strings.TrimSpace(tool.Type) != "" && strings.TrimSpace(tool.Type) != "image_generation" {
			return true
		}
	}
	return false
}

func stringValue(value interface{}, fallback string) string {
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}
