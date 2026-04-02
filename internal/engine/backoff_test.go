package engine

import (
	"testing"
	"time"
)

func TestBackoffCalc_Delay(t *testing.T) {
	b := BackoffCalc{MinMs: 500, MaxMs: 15000, Factor: 1.7}

	d0 := b.Delay(0)
	if d0 < 500*time.Millisecond || d0 > 550*time.Millisecond {
		t.Errorf("attempt 0: %v, want ~500ms", d0)
	}

	d1 := b.Delay(1)
	if d1 < 850*time.Millisecond || d1 > 935*time.Millisecond {
		t.Errorf("attempt 1: %v, want ~850ms", d1)
	}

	d2 := b.Delay(2)
	if d2 < 1445*time.Millisecond || d2 > 1590*time.Millisecond {
		t.Errorf("attempt 2: %v, want ~1445ms", d2)
	}
}

func TestBackoffCalc_CapsAtMax(t *testing.T) {
	b := BackoffCalc{MinMs: 500, MaxMs: 15000, Factor: 1.7}

	d := b.Delay(100)
	if d > 16500*time.Millisecond {
		t.Errorf("attempt 100: %v, should be capped near 15000ms", d)
	}
	if d < 15000*time.Millisecond {
		t.Errorf("attempt 100: %v, should be at least 15000ms", d)
	}
}

func TestBackoffCalc_ZeroAttempt(t *testing.T) {
	b := BackoffCalc{MinMs: 1000, MaxMs: 30000, Factor: 2.0}
	d := b.Delay(0)
	if d < 1000*time.Millisecond || d > 1100*time.Millisecond {
		t.Errorf("attempt 0: %v, want ~1000ms", d)
	}
}
