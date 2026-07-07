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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestSafeQuotaFromDecimal(t *testing.T) {
	// Normal value passes through unchanged.
	q, sat := safeQuotaFromDecimal(decimal.NewFromInt(12345))
	require.Equal(t, 12345, q)
	require.False(t, sat)

	// A value larger than int64 (what previously wrapped negative via
	// big.Int.Int64()) saturates to the ceiling instead.
	huge, _ := decimal.NewFromString("100000000000000000000000")
	q, sat = safeQuotaFromDecimal(huge)
	require.Equal(t, common.MaxQuotaValue, q)
	require.True(t, sat)
	require.Greater(t, q, 0)

	// Negative (only reachable via overflow) collapses to zero, never a credit.
	q, sat = safeQuotaFromDecimal(decimal.NewFromInt(-500))
	require.Equal(t, 0, q)
	require.True(t, sat)
}

// TestCalculateTextQuotaSummaryClampsOversizedImageN reproduces the reported
// vulnerability: an oversized image count `n` (applied as an OtherRatio) used to
// overflow the final int conversion and wrap to a negative quota, which the
// billing layer then credited to the user. The charge must now stay positive
// and clamp to the safe ceiling.
func TestCalculateTextQuotaSummaryClampsOversizedImageN(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 0,
		TotalTokens:      1000,
	}

	priceData := types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 1,
		GroupRatioInfo:  types.GroupRatioInfo{GroupRatio: 1},
		// Oversized user-controlled image count, far beyond int64 when multiplied.
		OtherRatios: map[string]float64{"n": 1e18},
	}

	relayInfo := &relaycommon.RelayInfo{
		RelayFormat:             types.RelayFormatOpenAIImage,
		FinalRequestRelayFormat: types.RelayFormatOpenAI,
		OriginModelName:         "gpt-image-1",
		PriceData:               priceData,
		StartTime:               time.Now(),
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	require.Greater(t, summary.Quota, 0, "quota must never wrap negative (which would credit the user)")
	require.Equal(t, common.MaxQuotaValue, summary.Quota, "oversized n must clamp to the safe ceiling")
}
