package responses

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ApiReq struct {
	Model        string      `json:"model"`
	Input        interface{} `json:"input"`
	Instructions string      `json:"instructions"`
	Stream       bool        `json:"stream"`
	Tools        []Tool      `json:"tools"`
}

type Tool struct {
	Type string `json:"type"`
}

type Event struct {
	Type         string      `json:"type"`
	Response     *Response   `json:"response,omitempty"`
	OutputIndex  int         `json:"output_index,omitempty"`
	ContentIndex int         `json:"content_index,omitempty"`
	ItemID       string      `json:"item_id,omitempty"`
	Item         *OutputItem `json:"item,omitempty"`
	Delta        string      `json:"delta,omitempty"`
	Text         string      `json:"text,omitempty"`
}

type Response struct {
	ID                string       `json:"id"`
	Object            string       `json:"object"`
	CreatedAt         int64        `json:"created_at"`
	Status            string       `json:"status"`
	Error             interface{}  `json:"error"`
	IncompleteDetails interface{}  `json:"incomplete_details"`
	Model             string       `json:"model"`
	Output            []OutputItem `json:"output"`
	ParallelToolCalls bool         `json:"parallel_tool_calls"`
}

type OutputItem struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Status  string        `json:"status"`
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

type ContentPart struct {
	Type        string        `json:"type"`
	Text        string        `json:"text"`
	Annotations []interface{} `json:"annotations"`
}

func TextOutputItem(id string, text string, status string) OutputItem {
	return OutputItem{
		ID:     id,
		Type:   "message",
		Status: status,
		Role:   "assistant",
		Content: []ContentPart{{
			Type:        "output_text",
			Text:        text,
			Annotations: []interface{}{},
		}},
	}
}

func CreatedEvent(responseID string, model string, created int64) Event {
	return Event{Type: "response.created", Response: &Response{
		ID:                responseID,
		Object:            "response",
		CreatedAt:         created,
		Status:            "in_progress",
		Error:             nil,
		IncompleteDetails: nil,
		Model:             model,
		Output:            []OutputItem{},
		ParallelToolCalls: false,
	}}
}

func CompletedEvent(responseID string, model string, created int64, output []OutputItem) Event {
	return Event{Type: "response.completed", Response: &Response{
		ID:                responseID,
		Object:            "response",
		CreatedAt:         created,
		Status:            "completed",
		Error:             nil,
		IncompleteDetails: nil,
		Model:             model,
		Output:            output,
		ParallelToolCalls: false,
	}}
}

func SSE(event Event) string {
	data, _ := json.Marshal(event)
	return "data: " + string(data) + "\n\n"
}

func NormalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "auto"
	}
	return model
}

func ResponseID() string {
	return fmt.Sprintf("resp_%d", time.Now().UnixNano())
}

func MessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
