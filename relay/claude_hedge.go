package relay

import (
	"net/http"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// ClaudeAttemptPlan 保存为某个渠道准备好的一次对冲（hedging）尝试要素。
type ClaudeAttemptPlan struct {
	Adaptor channel.Adaptor
	// Request 已构建好的上游请求（未设置 context / 未执行 Do）。
	Request *http.Request
	// UsesResponsesPath 为 true 表示该请求走 Claude->OpenAI Responses 转换分支，
	// 对冲编排不处理该分支，调用方应回退到串行 ClaudeHelper。
	UsesResponsesPath bool
}

// PrepareClaudeAttempt 为当前 c 上"已选定渠道"准备一次上游请求：完成 Claude 请求的通用准备、
// 请求体转换与 *http.Request 构建；不发起请求、不写 c.Writer、不启动 ping。
// info 应为调用方按渠道克隆出的独立 RelayInfo。
func PrepareClaudeAttempt(c *gin.Context, info *relaycommon.RelayInfo) (*ClaudeAttemptPlan, *types.NewAPIError) {
	adaptor, request, apiErr := claudePrepareRequest(c, info)
	if apiErr != nil {
		return nil, apiErr
	}

	// Claude -> OpenAI Responses 转换分支：对冲不处理，交由调用方回退。
	if !model_setting.GetGlobalSettings().PassThroughRequestEnabled &&
		!info.ChannelSetting.PassThroughBodyEnabled &&
		service.ShouldChatCompletionsUseResponsesGlobal(info.ChannelId, info.ChannelType, info.OriginModelName) {
		return &ClaudeAttemptPlan{Adaptor: adaptor, UsesResponsesPath: true}, nil
	}

	requestBody, apiErr := claudeBuildRequestBody(c, info, adaptor, request)
	if apiErr != nil {
		return nil, apiErr
	}

	req, err := channel.BuildApiRequest(adaptor, c, info, requestBody)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	return &ClaudeAttemptPlan{Adaptor: adaptor, Request: req}, nil
}

// RespondClaudeWinner 处理对冲 winner 的上游响应：状态码检查 + DoResponse（写 c.Writer），返回 usage。
// 必须在主 goroutine（HTTP handler 返回前）调用。
func RespondClaudeWinner(c *gin.Context, info *relaycommon.RelayInfo, plan *ClaudeAttemptPlan, httpResp *http.Response, statusCodeMappingStr string) (*dto.Usage, *types.NewAPIError) {
	return claudeRespondFromResponse(c, info, plan.Adaptor, httpResp, statusCodeMappingStr)
}

// ParseClaudeShadowUsage 从迟到成功的上游响应中仅解析计费所需 usage（不写客户端），供追补扣费使用。
// 调用方负责关闭 resp.Body。
func ParseClaudeShadowUsage(resp *http.Response, isStream bool) (*dto.Usage, error) {
	return claude.ParseUsageOnly(resp, isStream)
}
