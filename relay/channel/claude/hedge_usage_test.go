package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func makeResp(body, contentType string) *http.Response {
	h := http.Header{}
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestParseUsageOnly_Streaming(t *testing.T) {
	sse := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"type":"message","id":"msg_1","model":"claude-3-5-sonnet","usage":{"input_tokens":100,"output_tokens":1,"cache_read_input_tokens":20}}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	usage, err := ParseUsageOnly(makeResp(sse, "text/event-stream"), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", usage.PromptTokens)
	}
	if usage.CompletionTokens != 42 {
		t.Errorf("CompletionTokens = %d, want 42", usage.CompletionTokens)
	}
	if usage.TotalTokens != 142 {
		t.Errorf("TotalTokens = %d, want 142", usage.TotalTokens)
	}
	if usage.PromptTokensDetails.CachedTokens != 20 {
		t.Errorf("CachedTokens = %d, want 20", usage.PromptTokensDetails.CachedTokens)
	}
	if usage.UsageSemantic != "anthropic" {
		t.Errorf("UsageSemantic = %q, want anthropic", usage.UsageSemantic)
	}
}

func TestParseUsageOnly_NonStreaming(t *testing.T) {
	body := `{"type":"message","id":"msg_2","model":"claude-3","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":50,"output_tokens":10,"cache_read_input_tokens":5}}`

	usage, err := ParseUsageOnly(makeResp(body, "application/json"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.PromptTokens != 50 {
		t.Errorf("PromptTokens = %d, want 50", usage.PromptTokens)
	}
	if usage.CompletionTokens != 10 {
		t.Errorf("CompletionTokens = %d, want 10", usage.CompletionTokens)
	}
	if usage.TotalTokens != 60 {
		t.Errorf("TotalTokens = %d, want 60", usage.TotalTokens)
	}
	if usage.PromptTokensDetails.CachedTokens != 5 {
		t.Errorf("CachedTokens = %d, want 5", usage.PromptTokensDetails.CachedTokens)
	}
}

// TestParseUsageOnly_MalformedLinesSkipped 确认无法解析的 data 行被跳过而不影响整体 usage 提取。
func TestParseUsageOnly_MalformedLinesSkipped(t *testing.T) {
	sse := strings.Join([]string{
		`data: {not valid json`,
		`data: {"type":"message_start","message":{"type":"message","id":"m","model":"claude-3","usage":{"input_tokens":7,"output_tokens":3}}}`,
		`data: [DONE]`,
		``,
	}, "\n")

	usage, err := ParseUsageOnly(makeResp(sse, "text/event-stream"), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.PromptTokens != 7 || usage.CompletionTokens != 3 || usage.TotalTokens != 10 {
		t.Errorf("usage = (%d,%d,%d), want (7,3,10)", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}
}
