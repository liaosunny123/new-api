package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestInputTokenDetailsParsesCacheWriteFields 验证 GPT-5.6 Responses usage 的两种缓存写入字段写法
// （cache_write_tokens / cache_creation_tokens）都能解析，且 EffectiveCacheWriteTokens 取较大者。
func TestInputTokenDetailsParsesCacheWriteFields(t *testing.T) {
	// 写法一：cache_write_tokens
	var u1 dto.Usage
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens":52006,"output_tokens":1300,"input_tokens_details":{"cached_tokens":48128,"cache_write_tokens":3200}}`, &u1))
	require.NotNil(t, u1.InputTokensDetails)
	require.Equal(t, 3200, u1.InputTokensDetails.EffectiveCacheWriteTokens())

	// 写法二：cache_creation_tokens
	var u2 dto.Usage
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens":52006,"input_tokens_details":{"cached_tokens":48128,"cache_creation_tokens":3200}}`, &u2))
	require.Equal(t, 3200, u2.InputTokensDetails.EffectiveCacheWriteTokens())

	// 两个都在：取较大者
	var u3 dto.Usage
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens_details":{"cache_write_tokens":100,"cache_creation_tokens":900}}`, &u3))
	require.Equal(t, 900, u3.InputTokensDetails.EffectiveCacheWriteTokens())

	// 都没有：0
	var u4 dto.Usage
	require.NoError(t, common.UnmarshalJsonStr(`{"input_tokens_details":{"cached_tokens":10}}`, &u4))
	require.Equal(t, 0, u4.InputTokensDetails.EffectiveCacheWriteTokens())
}

// TestCacheWriteBillingOpenAI 验证非 Claude（OpenAI/Responses）模型的缓存写入按 CacheCreationRatio(1.25) 计费。
// base = prompt - read - write = 1000-300-200 = 500；
// promptQuota = 500 + 300*0.1 + 200*1.25 = 780；completion = 100*2 = 200；quota = 980。
// 若不计写入（当作普通输入）则为 930，差值 = 200*(1.25-1) = 50。
func TestCacheWriteBillingOpenAI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	newInfo := func() *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{
			RelayFormat:     types.RelayFormatOpenAI,
			OriginModelName: "gpt-5.6-terra",
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
	newUsage := func(cacheWrite int) *dto.Usage {
		return &dto.Usage{
			PromptTokens:     1000,
			CompletionTokens: 100,
			PromptTokensDetails: dto.InputTokenDetails{
				CachedTokens:     300,
				CacheWriteTokens: cacheWrite,
			},
		}
	}

	withWrite := calculateTextQuotaSummary(ctx, newInfo(), newUsage(200))
	require.Equal(t, 200, withWrite.CacheCreationTokens, "缓存写入 token 应被捕获")
	require.Equal(t, 980, withWrite.Quota)

	noWrite := calculateTextQuotaSummary(ctx, newInfo(), newUsage(0))
	require.Equal(t, 930, noWrite.Quota)
	require.Greater(t, withWrite.Quota, noWrite.Quota, "缓存写入应按 1.25x 溢价计费，比不计更贵")
}
