package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestEffectiveFirstByteTimeout(t *testing.T) {
	orig := common.FirstByteTimeout
	defer func() { common.FirstByteTimeout = orig }()

	cases := []struct {
		name    string
		system  int
		userVal int
		want    int
	}{
		{"system3000 user0 -> 3000", 3000, 0, 3000},
		{"system3000 user600 -> 3000", 3000, 600, 3000},
		{"system3000 user5000 -> capped 3000", 3000, 5000, 3000},
		{"system600 user0 -> 600", 600, 0, 600},
		{"system600 user3000 -> upgrade 3000", 600, 3000, 3000},
		{"system600 user100 -> 600 (take max)", 600, 100, 600},
		{"system0 user0 -> 0 (unlimited)", 0, 0, 0},
		{"system0 user1500 -> 1500", 0, 1500, 1500},
		{"system5000 clamped to 3000", 5000, 0, 3000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			common.FirstByteTimeout = tc.system
			if got := EffectiveFirstByteTimeout(tc.userVal); got != tc.want {
				t.Errorf("EffectiveFirstByteTimeout(system=%d, user=%d) = %d, want %d", tc.system, tc.userVal, got, tc.want)
			}
		})
	}
}
