package economy

import (
	"slices"
	"time"
)

const (
	StartBalance   int64 = 2500
	MinStake       int64 = 100
	FeePercent     int64 = 5
	TopUpThreshold int64 = 100
	TopUpTo        int64 = 500
	TopUpCooldown        = time.Hour
)

var AllowedStakes = []int64{100, 250, 500, 1000, 2500, 5000}

func IsAllowedStake(s int64) bool {
	return slices.Contains(AllowedStakes, s)
}
