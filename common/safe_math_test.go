package common

import (
	"math"
	"testing"
)

func TestSafeQuotaFromFloat(t *testing.T) {
	cases := []struct {
		name    string
		in      float64
		want    int
		wantSat bool
	}{
		{"normal", 12345, 12345, false},
		{"zero", 0, 0, false},
		{"negative_from_overflow", -1, 0, true},
		{"overflow_ceiling", 1e18, MaxQuotaValue, true},
		{"exact_ceiling", float64(MaxQuotaValue), MaxQuotaValue, true},
		{"positive_infinity", math.Inf(1), MaxQuotaValue, true},
		{"negative_infinity", math.Inf(-1), 0, true},
		{"nan", math.NaN(), 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, sat := SafeQuotaFromFloat(c.in)
			if got != c.want || sat != c.wantSat {
				t.Fatalf("SafeQuotaFromFloat(%v) = (%d, %v), want (%d, %v)", c.in, got, sat, c.want, c.wantSat)
			}
			if got < 0 {
				t.Fatalf("quota must never be negative, got %d", got)
			}
		})
	}
}
