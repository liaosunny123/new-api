package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// hedgeShadowMaxWait 迟到成功（影子请求）在后台存活的硬上限。
// 由于 RELAY_TIMEOUT 默认 0（上游无超时），必须对挂后台的旧路设置绝对上限，
// 避免连接/goroutine 无限泄漏。取值需大于常见的"迟到成功"耗时（用户观测约 1000s）。
const hedgeShadowMaxWait = 20 * time.Minute

// hedgeAttempt 表示一次对冲尝试的隔离状态。
type hedgeAttempt struct {
	idx       int
	info      *relaycommon.RelayInfo
	plan      *relay.ClaudeAttemptPlan
	channel   *model.Channel
	shadowCtx *gin.Context // c.Copy() 快照，供后台影子计费在 handler 返回后安全使用
	cancel    context.CancelFunc
}

// hedgeResult 是一次 client.Do 的结果。
type hedgeResult struct {
	idx  int
	resp *http.Response
	err  error
}

// hedgedClaudeRelay 为 Claude /v1/messages 实现请求对冲：
//   - 某路在 timeout 内未拿到响应头，则并行发起下一路（最多 maxAttempts 路），旧路不取消挂后台；
//   - 首个返回 2xx 的路成为 winner，其响应流式返回给用户并走正常预扣→结算计费；
//   - winner 决出后仍在途/迟到返回成功的旧路，在后台对其成功响应做独立"追补扣费"。
//
// 返回 handled=false 表示该请求走 Claude->OpenAI Responses 转换分支，对冲不处理，
// 调用方应回退到串行 ClaudeHelper。
func hedgedClaudeRelay(c *gin.Context, baseInfo *relaycommon.RelayInfo) (handled bool, apiErr *types.NewAPIError) {
	timeout := time.Duration(baseInfo.UserSetting.GetHedgeFirstResponseTimeout()) * time.Second
	maxAttempts := baseInfo.UserSetting.GetHedgeMaxAttempts()
	statusCodeMappingStr := c.GetString("status_code_mapping")

	// 让每一路 attempt 都经由缓存按 retry 递增选择渠道（而非复用 distributor 在 context 里的初始渠道）。
	// getChannel 在 info.ChannelMeta == nil 时会返回 context 里的固定渠道，导致所有 attempt 命中同一渠道；
	// 这里先填充 ChannelMeta，强制走 CacheGetRandomSatisfiedChannel(retryParam) 的按优先级递增选择。
	if baseInfo.ChannelMeta == nil {
		baseInfo.InitChannelMeta(c)
	}

	resCh := make(chan hedgeResult, maxAttempts)
	attempts := make([]*hedgeAttempt, 0, maxAttempts)

	retryParam := &service.RetryParam{
		Ctx:        c,
		TokenGroup: baseInfo.TokenGroup,
		ModelName:  baseInfo.OriginModelName,
		Retry:      common.GetPointer(0),
	}

	// launch 在主 goroutine 串行调用：选渠道、构建请求、起 client.Do goroutine。
	// 返回 (fallback, apiErr)：fallback=true 表示遇到 Responses 转换分支；apiErr!=nil 表示无法再启动新路。
	launch := func() (fallback bool, apiErr *types.NewAPIError) {
		idx := len(attempts)
		baseInfo.RetryIndex = retryParam.GetRetry()
		channel, chErr := getChannel(c, baseInfo, retryParam)
		if chErr != nil {
			return false, chErr
		}
		retryParam.IncreaseRetry()
		addUsedChannel(c, channel.Id)

		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			return false, types.NewErrorWithStatusCode(bodyErr, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		c.Request.Body = io.NopCloser(bodyStorage)

		infoClone := *baseInfo
		plan, prepErr := relay.PrepareClaudeAttempt(c, &infoClone)
		if prepErr != nil {
			return false, prepErr
		}
		if plan.UsesResponsesPath {
			return true, nil
		}

		client, cErr := hedgeHTTPClient(&infoClone)
		if cErr != nil {
			return false, types.NewOpenAIError(cErr, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
		}

		ctx, cancel := context.WithCancel(context.Background())
		req := plan.Request.WithContext(ctx)

		a := &hedgeAttempt{
			idx:       idx,
			info:      &infoClone,
			plan:      plan,
			channel:   channel,
			shadowCtx: c.Copy(),
			cancel:    cancel,
		}
		attempts = append(attempts, a)

		gopool.Go(func() {
			resp, err := client.Do(req)
			resCh <- hedgeResult{idx: idx, resp: resp, err: err}
		})
		return false, nil
	}

	// 首路
	fallback, launchErr := launch()
	if fallback {
		return false, nil // 回退到串行 ClaudeHelper
	}
	if launchErr != nil {
		return true, launchErr
	}

	launched := 1
	consumed := 0
	canLaunch := launched < maxAttempts
	lastLaunch := time.Now()
	var lastErr *types.NewAPIError

	for {
		var timerC <-chan time.Time
		if canLaunch {
			remaining := timeout - time.Since(lastLaunch)
			if remaining < 0 {
				remaining = 0
			}
			timerC = time.After(remaining)
		}

		select {
		case res := <-resCh:
			consumed++
			a := attempts[res.idx]
			if res.err == nil && res.resp != nil && res.resp.StatusCode == http.StatusOK {
				// WINNER：把其余（在途/迟到）尝试转为后台影子，主 goroutine 配送 winner 响应。
				spawnHedgeShadows(attempts, res.idx, resCh, launched-consumed)
				return true, finishHedgeWinner(c, baseInfo, a, res.resp, statusCodeMappingStr)
			}
			// 本路失败
			a.cancel()
			lastErr = hedgeAttemptError(c, a, res, statusCodeMappingStr)
			if consumed == launched && !canLaunch {
				return true, lastErr // 所有已发起的尝试均失败，且无法再启动新路
			}
			if canLaunch {
				fb, err := launch()
				if fb || err != nil {
					canLaunch = false // 无法再启动新路（渠道耗尽 / Responses 分支）
				} else {
					launched++
					lastLaunch = time.Now()
					canLaunch = launched < maxAttempts
				}
			}
			if consumed == launched && !canLaunch {
				return true, lastErr
			}
		case <-timerC:
			// timeout 内未出 winner：并行发起下一路（旧路不取消）。
			fb, err := launch()
			if fb || err != nil {
				canLaunch = false
			} else {
				launched++
				lastLaunch = time.Now()
				canLaunch = launched < maxAttempts
			}
		}
	}
}

// finishHedgeWinner 在主 goroutine 配送 winner 的上游响应并完成正常计费。
func finishHedgeWinner(c *gin.Context, baseInfo *relaycommon.RelayInfo, a *hedgeAttempt, resp *http.Response, statusCodeMappingStr string) *types.NewAPIError {
	// winner 存活到流式结束；结束后取消其 context 释放连接。
	defer a.cancel()

	// 恢复 c 上 winner 渠道的上下文（此前可能被后发尝试覆盖），确保计费/日志归属正确渠道。
	if setupErr := middleware.SetupContextForSelectedChannel(c, a.channel, baseInfo.OriginModelName); setupErr != nil {
		service.CloseResponseBodyGracefully(resp)
		return setupErr
	}

	usage, apiErr := relay.RespondClaudeWinner(c, a.info, a.plan, resp, statusCodeMappingStr)
	if apiErr != nil {
		return apiErr
	}
	service.PostTextConsumeQuota(c, a.info, usage, nil)
	logHedgeUsedChannels(c)
	return nil
}

// spawnHedgeShadows 把 winner 决出后其余尝试的结果收集到后台，对迟到成功者做追补计费。
// remaining 为 resCh 上尚未消费的结果数（不含 winner）。
func spawnHedgeShadows(attempts []*hedgeAttempt, winnerIdx int, resCh chan hedgeResult, remaining int) {
	if remaining <= 0 {
		return
	}
	// 为所有非 winner 尝试设置硬上限，避免挂后台的旧路无限存活。
	for _, a := range attempts {
		if a.idx == winnerIdx {
			continue
		}
		time.AfterFunc(hedgeShadowMaxWait, a.cancel)
	}
	gopool.Go(func() {
		for i := 0; i < remaining; i++ {
			res := <-resCh
			a := attempts[res.idx]
			gopool.Go(func() {
				handleHedgeShadow(a, res)
			})
		}
	})
}

// handleHedgeShadow 处理一条迟到的旧路结果：成功则追补扣费，否则丢弃。
func handleHedgeShadow(a *hedgeAttempt, res hedgeResult) {
	defer a.cancel()
	if res.err != nil || res.resp == nil {
		return
	}
	defer service.CloseResponseBodyGracefully(res.resp)
	if res.resp.StatusCode != http.StatusOK {
		return
	}

	isStream := a.info.IsStream || strings.HasPrefix(res.resp.Header.Get("Content-Type"), "text/event-stream")
	usage, err := relay.ParseClaudeShadowUsage(res.resp, isStream)
	if err != nil {
		logger.LogError(a.shadowCtx, "hedge shadow parse usage failed: "+err.Error())
		return
	}
	if usage == nil || usage.TotalTokens == 0 {
		return
	}

	// 追补扣费：独立于 winner 的计费会话，做一次性全额扣费 + 消费日志。
	shadowInfo := a.info
	shadowInfo.Billing = nil
	shadowInfo.FinalPreConsumedQuota = 0
	shadowInfo.IsStream = isStream
	service.PostTextConsumeQuota(a.shadowCtx, shadowInfo, usage, []string{"对冲追补扣费（影子请求，上游迟到成功）"})
}

// hedgeAttemptError 将一条失败的对冲尝试结果转换为 NewAPIError，并触发渠道禁用/错误日志。
func hedgeAttemptError(c *gin.Context, a *hedgeAttempt, res hedgeResult, statusCodeMappingStr string) *types.NewAPIError {
	var apiErr *types.NewAPIError
	if res.err != nil {
		apiErr = types.NewOpenAIError(res.err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	} else if res.resp != nil {
		apiErr = service.RelayErrorHandler(c.Request.Context(), res.resp, false)
		service.ResetStatusCode(apiErr, statusCodeMappingStr)
		service.CloseResponseBodyGracefully(res.resp)
	} else {
		apiErr = types.NewError(io.ErrUnexpectedEOF, types.ErrorCodeDoRequestFailed)
	}
	processChannelError(c, *types.NewChannelError(a.channel.Id, a.channel.Type, a.channel.Name,
		a.channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), a.channel.GetAutoBan()), apiErr)
	return apiErr
}

// hedgeHTTPClient 按渠道代理配置 + 首字节硬超时返回上游 HTTP 客户端（与 doRequest 选择逻辑一致），
// 保证对冲各路也受首字节超时兜底。
func hedgeHTTPClient(info *relaycommon.RelayInfo) (*http.Client, error) {
	rht := service.EffectiveFirstByteTimeout(info.UserSetting.GetFirstByteTimeout())
	return service.GetHttpClientWithResponseHeaderTimeout(rht, info.ChannelSetting.Proxy)
}

func logHedgeUsedChannels(c *gin.Context) {
	useChannel := c.GetStringSlice("use_channel")
	if len(useChannel) > 1 {
		retryLogStr := fmt.Sprintf("对冲：%s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(useChannel)), "->"), "[]"))
		logger.LogInfo(c, retryLogStr)
	}
}
