package completions

import (
	"chat2api/app/types"
	"encoding/json"
	"strings"
	"time"
)

type ApiRespStream struct {
	ID             string            `json:"id,omitempty"`
	Object         string            `json:"object,omitempty"`
	Created        int64             `json:"created,omitempty"`
	Model          string            `json:"model,omitempty"`
	ConversationId string            `json:"conversation_id,omitempty"`
	MessageId      string            `json:"message_id,omitempty"`
	Choices        []ApiStreamChoice `json:"choices,omitempty"`
}

type ApiStreamChoice struct {
	Delta        ApiStreamDelta `json:"delta,omitempty"`
	Index        int            `json:"index,omitempty"`
	FinishReason interface{}    `json:"finish_reason,omitempty"`
}

type ApiStreamDelta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

func NewApiRespStream(id string, model string, content string) *ApiRespStream {
	return &ApiRespStream{
		ID:      id,
		Created: time.Now().Unix(),
		Object:  "chat.completion.chunk",
		Model:   model,
		Choices: []ApiStreamChoice{
			{
				Delta: ApiStreamDelta{
					Content: content,
				},
				Index:        0,
				FinishReason: nil,
			},
		},
	}
}

func ConvertToString(id string, model string, chatResp *types.ChatResp, previousText *types.StringStruct, role bool) string {
	if len(chatResp.Message.Content.Parts) == 0 {
		return ""
	}
	text, ok := chatResp.Message.Content.Parts[0].(string)
	if !ok {
		return ""
	}
	apiRespJson := NewApiRespStream(id, model, strings.Replace(text, previousText.Text, "", 1))
	apiRespJson.ConversationId = chatResp.ConversationId
	apiRespJson.MessageId = chatResp.Message.Id
	if role {
		apiRespJson.Choices[0].Delta.Role = chatResp.Message.Author.Role
	} else if apiRespJson.Choices[0].Delta.Content == "" || (strings.HasPrefix(chatResp.Message.Metadata.ModelSlug, "gpt-4") && apiRespJson.Choices[0].Delta.Content == "【") {
		return apiRespJson.Choices[0].Delta.Content
	}
	previousText.Text = text
	data, _ := json.Marshal(apiRespJson)
	return "data: " + string(data) + "\n\n"
}

func StopChunk(id string, model string, finishReason string) ApiRespStream {
	return ApiRespStream{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []ApiStreamChoice{
			{
				Index:        0,
				FinishReason: finishReason,
			},
		},
	}
}

func (ARS *ApiRespStream) String() string {
	resp, _ := json.Marshal(ARS)
	return string(resp)
}
