package engine

import (
	"math"
	"math/rand/v2"
	"time"
)

// BackoffCalc computes exponential backoff delays with jitter.
type BackoffCalc struct {
	MinMs  int
	MaxMs  int
	Factor float64
}

// Delay returns the backoff duration for the given attempt number.
// Formula: min(minMs * factor^attempt, maxMs) + 0~10% jitter.
func (b *BackoffCalc) Delay(attempt int) time.Duration {
	base := float64(b.MinMs) * math.Pow(b.Factor, float64(attempt))
	if base > float64(b.MaxMs) {
		base = float64(b.MaxMs)
	}
	jitter := base * 0.1 * rand.Float64()
	ms := base + jitter
	return time.Duration(ms) * time.Millisecond
}
