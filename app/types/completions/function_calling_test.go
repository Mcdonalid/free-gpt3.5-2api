package completions

import (
	"strings"
	"testing"
)

func TestParseFunctionCallsXMLWithCDATAAndThink(t *testing.T) {
	content := `<think>` + ToolifyTriggerSignal + `</think>
prefix
` + ToolifyTriggerSignal + `
<function_calls>
  <function_call>
    <tool>search</tool>
    <args_json><![CDATA[{"query":"hello","limit":2}]]></args_json>
  </function_call>
</function_calls>`

	calls := ParseFunctionCallsXML(content, ToolifyTriggerSignal)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "search" {
		t.Fatalf("unexpected tool name: %s", calls[0].Name)
	}
	if calls[0].Args["query"] != "hello" {
		t.Fatalf("unexpected query arg: %#v", calls[0].Args["query"])
	}
	if calls[0].Args["limit"].(float64) != 2 {
		t.Fatalf("unexpected limit arg: %#v", calls[0].Args["limit"])
	}
}

func TestParseFunctionCallsXMLRejectsNonObjectArgsJSON(t *testing.T) {
	content := ToolifyTriggerSignal + `
<function_calls>
  <function_call>
    <tool>search</tool>
    <args_json>[1,2]</args_json>
  </function_call>
</function_calls>`

	if calls := ParseFunctionCallsXML(content, ToolifyTriggerSignal); len(calls) != 0 {
		t.Fatalf("expected no calls, got %#v", calls)
	}
}

func TestPreprocessMessagesConvertsToolResults(t *testing.T) {
	messages := []ApiMessage{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "search",
					Arguments: `{"query":"go"}`,
				},
			}},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "result text"},
	}

	processed, err := PreprocessMessages(messages)
	if err != nil {
		t.Fatal(err)
	}
	if len(processed) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(processed))
	}
	if processed[0].ToolCalls != nil {
		t.Fatalf("assistant tool_calls should be converted to content")
	}
	if !strings.Contains(processed[0].Content.(string), "<function_calls>") {
		t.Fatalf("assistant content missing function_calls XML: %s", processed[0].Content)
	}
	if processed[1].Role != "user" {
		t.Fatalf("tool result should be converted to user role, got %s", processed[1].Role)
	}
	if !strings.Contains(processed[1].Content.(string), "Tool name: search") {
		t.Fatalf("tool result missing tool context: %s", processed[1].Content)
	}
}

func TestBuildFunctionPromptValidatesToolChoice(t *testing.T) {
	tools := []Tool{{
		Type: "function",
		Function: ToolFunction{
			Name: "search",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"query"},
			},
		},
	}}
	_, err := BuildFunctionPrompt(tools, map[string]interface{}{
		"type":     "function",
		"function": map[string]interface{}{"name": "missing"},
	})
	if err == nil {
		t.Fatal("expected invalid tool_choice error")
	}
}

func TestStreamToolDetectorHoldsPartialTrigger(t *testing.T) {
	detector := NewStreamToolDetector(ToolifyTriggerSignal)
	detected, out := detector.ProcessChunk("hello " + ToolifyTriggerSignal[:10])
	if detected {
		t.Fatal("partial trigger should not detect")
	}
	if out != "hello " {
		t.Fatalf("unexpected output: %q", out)
	}
	detected, out = detector.ProcessChunk(ToolifyTriggerSignal[10:] + "\n<function_calls>")
	if !detected {
		t.Fatal("expected trigger detection")
	}
	if out != "" {
		t.Fatalf("trigger should not be emitted, got %q", out)
	}
	if detector.State() != "tool_parsing" {
		t.Fatalf("unexpected state: %s", detector.State())
	}
}
