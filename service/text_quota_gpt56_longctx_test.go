package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestLongContextTierPricing 验证 gpt-5.4/5.5/5.6 输入上下文 > 272K 时的长上下文分档计费：
// 输入 / 缓存读取 / 缓存写入 ×2，输出 ×1.5；短上下文、非目标模型、阈值边界不受影响。
func TestLongContextTierPricing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	newInfo := func(model string) *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{
			RelayFormat:     types.RelayFormatOpenAI,
			OriginModelName: model,
			PriceData: types.PriceData{
				ModelRatio:         1,
				CompletionRatio:    2,
				CacheRatio:         0.1,
				CacheCreationRatio: 1.25,
				GroupRatioInfo:     types.GroupRatioInfo{GroupRatio: 1},
			},
			StartTime: time.Now(),
		}
	}
	newUsage := func(input int) *dto.Usage {
		return &dto.Usage{
			PromptTokens:     input,
			CompletionTokens: 1000,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:     100000,
				CacheWriteTokens: 50000,
			},
		}
	}

	// 长上下文（>272K）：ModelRatio ×2、CompletionRatio ×0.75(=输出净额 1.5×)
	longGpt56 := calculateTextQuotaSummary(ctx, newInfo("gpt-5.6-terra"), newUsage(300000))
	require.Equal(t, float64(2), longGpt56.ModelRatio)
	require.Equal(t, 1.5, longGpt56.CompletionRatio) // 2 × 0.75

	// 精确数值：
	//   输入 base = 300000-100000-50000 = 150000
	//   缓存读 = 100000*0.1 = 10000；缓存写 = 50000*1.25 = 62500
	//   promptQuota = 222500；completionQuota = 1000*1.5 = 1500
	//   (222500+1500) * ModelRatio(2) = 448000
	require.Equal(t, 448000, longGpt56.Quota)

	// 分量核对：输入+缓存部分应 ×2、输出部分应 ×1.5（相对普通模型）
	longOther := calculateTextQuotaSummary(ctx, newInfo("gpt-4o"), newUsage(300000))
	require.Equal(t, 224500, longOther.Quota) // (222500 + 1000*2) * 1
	// 输入+缓存: 222500*2=445000; 输出: 1000*2*1.5=3000 → 448000；非目标模型 222500+2000=224500

	// gpt-5.4 / gpt-5.5 同样启用
	require.Equal(t, float64(2), calculateTextQuotaSummary(ctx, newInfo("gpt-5.5-pro"), newUsage(300000)).ModelRatio)
	require.Equal(t, float64(2), calculateTextQuotaSummary(ctx, newInfo("gpt-5.4-nano"), newUsage(300000)).ModelRatio)

	// 短上下文（≤272K）不翻倍
	shortGpt56 := calculateTextQuotaSummary(ctx, newInfo("gpt-5.6-terra"), newUsage(200000))
	require.Equal(t, float64(1), shortGpt56.ModelRatio)
	require.Equal(t, float64(2), shortGpt56.CompletionRatio)

	// 边界：恰好 272000 属短档（≤），不翻倍；272001 翻倍
	require.Equal(t, float64(1), calculateTextQuotaSummary(ctx, newInfo("gpt-5.6-terra"), newUsage(272000)).ModelRatio)
	require.Equal(t, float64(2), calculateTextQuotaSummary(ctx, newInfo("gpt-5.6-terra"), newUsage(272001)).ModelRatio)

	// 非目标模型任何长度都不翻倍
	require.Equal(t, float64(1), calculateTextQuotaSummary(ctx, newInfo("gpt-4o"), newUsage(300000)).ModelRatio)
	require.Equal(t, float64(1), calculateTextQuotaSummary(ctx, newInfo("gpt-5.7-x"), newUsage(300000)).ModelRatio)
}
