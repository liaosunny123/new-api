package claude

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// shadowScannerMaxBuffer 影子解析单行最大缓冲，Claude 单个 SSE 事件可能较大（长文本/思考）。
const shadowScannerMaxBuffer = 10 * 1024 * 1024

// ParseUsageOnly 读取上游 Claude 响应体，仅解析出计费所需的 usage，不写任何客户端。
// 供请求对冲（hedging）的后台影子路径对迟到成功的响应做追补计费使用。
// 调用方负责关闭 resp.Body。
func ParseUsageOnly(resp *http.Response, isStream bool) (*dto.Usage, error) {
	claudeInfo := &ClaudeResponseInfo{
		Created:      common.GetTimestamp(),
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}

	if isStream {
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024), shadowScannerMaxBuffer)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(line[len("data:"):])
			if data == "" || data == "[DONE]" {
				continue
			}
			var claudeResponse dto.ClaudeResponse
			if err := common.UnmarshalJsonStr(data, &claudeResponse); err != nil {
				// 跳过无法解析的行，尽力从其余事件中提取 usage
				continue
			}
			// FormatClaudeResponseInfo 仅更新 claudeInfo（含 Usage），不写客户端。
			FormatClaudeResponseInfo(&claudeResponse, nil, claudeInfo)
		}
		if err := scanner.Err(); err != nil {
			return claudeInfo.Usage, err
		}
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var claudeResponse dto.ClaudeResponse
		if err := common.Unmarshal(body, &claudeResponse); err != nil {
			return nil, err
		}
		if claudeResponse.Usage != nil {
			claudeInfo.Usage.PromptTokens = claudeResponse.Usage.InputTokens
			claudeInfo.Usage.CompletionTokens = claudeResponse.Usage.OutputTokens
			claudeInfo.Usage.PromptTokensDetails.CachedTokens = claudeResponse.Usage.CacheReadInputTokens
			claudeInfo.Usage.PromptTokensDetails.CachedCreationTokens = claudeResponse.Usage.CacheCreationInputTokens
			claudeInfo.Usage.ClaudeCacheCreation5mTokens = claudeResponse.Usage.GetCacheCreation5mTokens()
			claudeInfo.Usage.ClaudeCacheCreation1hTokens = claudeResponse.Usage.GetCacheCreation1hTokens()
		}
	}

	u := claudeInfo.Usage
	u.TotalTokens = u.PromptTokens + u.CompletionTokens
	u.UsageSemantic = "anthropic"
	return u, nil
}
