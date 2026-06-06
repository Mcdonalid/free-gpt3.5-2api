package responses

type ApiReq struct {
	Model        string      `json:"model"`
	Input        interface{} `json:"input"`
	Instructions string      `json:"instructions"`
	Stream       bool        `json:"stream"`
	Tools        []Tool      `json:"tools"`
	ToolChoice   interface{} `json:"tool_choice"`
}

type Tool struct {
	Type         string `json:"type"`
	Model        string `json:"model,omitempty"`
	Action       string `json:"action,omitempty"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}
