package common

import "math"

// MaxQuotaValue is the saturating ceiling applied to every single-request
// quota/billing computation.
//
// User-controlled quantity parameters (image count `n`, video `duration`/
// `seconds`, `max_tokens`, ...) are multiplied by price/ratio into an int
// quota. Without a ceiling, a crafted oversized value overflows the
// float64->int or decimal->int64 conversion and wraps to a NEGATIVE quota; a
// negative quota is then treated as a refund by PostConsumeQuota /
// taskAdjustFunding (IncreaseUserQuota), turning a charge into free credit.
//
// Clamping every quota conversion to the int32 range makes that class of
// overflow impossible while still allowing any realistic single-request cost
// (MaxInt32 quota is ~$4294 at the default QuotaPerUnit of 500000).
const MaxQuotaValue = math.MaxInt32

// MaxTokensLimit is the upper bound accepted for a user-supplied max_tokens
// style field, matching the long-standing check in the OpenAI text validator.
const MaxTokensLimit = math.MaxInt32 / 2

// MaxImageN caps the requested number of images per request. Real providers
// allow single digits; this is a generous sanity bound that rejects clearly
// abusive values before they reach quota math.
const MaxImageN = 100000

// MaxVideoSeconds caps the requested video/audio duration in seconds. Far above
// any real generation length, it exists purely to reject overflow-scale input.
const MaxVideoSeconds = 86400

// SafeQuotaFromFloat converts a computed float64 quota to int, saturating into
// [0, MaxQuotaValue]. NaN/Inf and overflowing or negative inputs (a charge is
// never legitimately negative, so a negative here can only come from overflow
// upstream) collapse to a safe bound instead of wrapping. The returned bool
// reports whether saturation clamped the value, so callers can surface it for
// admin auditing.
func SafeQuotaFromFloat(f float64) (int, bool) {
	switch {
	case math.IsNaN(f):
		return 0, true
	case f < 0:
		return 0, true
	case f >= float64(MaxQuotaValue):
		return MaxQuotaValue, true
	default:
		return int(f), false
	}
}
