package chat_backend

import (
	"strings"

	"chat2api/app/types"
	"chat2api/app/types/completions"

	"github.com/google/uuid"
)

func BuildChatRequest(apiReq *completions.ApiReq) *types.ChatReq {
	messages := make([]types.ChatMessages, 0, len(apiReq.Messages))
	for _, apiMessage := range apiReq.Messages {
		messages = append(messages, types.ChatMessages{
			Id: uuid.New().String(),
			Author: types.ChatAuthor{
				Role: apiMessage.Role,
			},
			Content: types.ChatContent{
				ContentType: "text",
				Parts:       []string{apiMessage.Content},
			},
		})
	}
	parentMessageId := strings.TrimSpace(apiReq.ParentMessageId)
	if parentMessageId == "" {
		parentMessageId = uuid.New().String()
	}

	return &types.ChatReq{
		Action:                     "next",
		Messages:                   messages,
		ConversationId:             strings.TrimSpace(apiReq.ConversationId),
		ParentMessageId:            parentMessageId,
		Model:                      normalizeModel(apiReq.Model),
		Timezone:                   "Asia/Shanghai",
		TimeZoneOffsetMin:          -480,
		Suggestions:                make([]string, 0),
		SupportedEncodings:         make([]string, 0),
		SystemHints:                make([]string, 0),
		HistoryAndTrainingDisabled: true,
		ForceUseSse:                true,
		FaceUseSse:                 false,
		ForceParagen:               false,
		ForceParagenModelSlug:      "",
		ForceRateLimit:             false,
		ResetRateLimits:            false,
		VariantPurpose:             "comparison_implicit",
		ConversationMode: types.ChatConversationMode{
			Kind: "primary_assistant",
		},
		WebsocketRequestId: uuid.New().String(),
		ClientContextualInfo: types.ClientContextualInfo{
			IsDarkMode:      false,
			TimeSinceLoaded: 120,
			PageHeight:      900,
			PageWidth:       1400,
			PixelRatio:      2,
			ScreenHeight:    1440,
			ScreenWidth:     2560,
		},
	}
}

func normalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "auto"
	}
	return model
}
