package completions

import "time"

func NewApiRespStream(id string, model string, content string) *ApiRespStream {
	contentPtr := content
	return &ApiRespStream{
		ID:      id,
		Created: time.Now().Unix(),
		Object:  "chat.completion.chunk",
		Model:   model,
		Choices: []ApiStreamChoice{
			{
				Delta: ApiStreamDelta{
					Content: &contentPtr,
				},
				Index:        0,
				FinishReason: nil,
			},
		},
	}
}

func NewToolCallsApiRespStreams(id string, model string, toolCalls []ToolCall) []*ApiRespStream {
	chunks := make([]*ApiRespStream, 0, len(toolCalls)*2)
	for i, call := range toolCalls {
		index := i
		if call.Index != nil {
			index = *call.Index
		}
		chunks = append(chunks, newToolCallApiRespStream(id, model, ToolCall{
			Index: toolCallIndexPtr(index),
			ID:    call.ID,
			Type:  call.Type,
			Function: ToolCallFunction{
				Name: call.Function.Name,
			},
		}))
		if call.Function.Arguments != "" {
			chunks = append(chunks, newToolCallApiRespStream(id, model, ToolCall{
				Index: toolCallIndexPtr(index),
				Function: ToolCallFunction{
					Arguments: call.Function.Arguments,
				},
			}))
		}
	}
	return chunks
}

func NewToolCallsApiRespStream(id string, model string, toolCalls []ToolCall) *ApiRespStream {
	chunks := NewToolCallsApiRespStreams(id, model, toolCalls)
	if len(chunks) == 0 {
		return newToolCallApiRespStream(id, model)
	}
	return chunks[0]
}

func newToolCallApiRespStream(id string, model string, toolCalls ...ToolCall) *ApiRespStream {
	return &ApiRespStream{
		ID:      id,
		Created: time.Now().Unix(),
		Object:  "chat.completion.chunk",
		Model:   model,
		Choices: []ApiStreamChoice{{
			Delta: ApiStreamDelta{
				ToolCalls: toolCalls,
			},
			Index:        0,
			FinishReason: nil,
		}},
	}
}

func toolCallIndexPtr(value int) *int {
	return &value
}

func StopChunk(id string, model string, finishReason string) ApiRespStream {
	if finishReason == "" {
		finishReason = "stop"
	}
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
