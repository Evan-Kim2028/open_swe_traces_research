package app

import "testing"

func TestSumClamped(t *testing.T) {
	if got := SumClamped(40, 70); got != 100 {
		t.Fatalf("SumClamped(40,70)=%d want 100", got)
	}
	if got := SumClamped(2, 3); got != 5 {
		t.Fatalf("SumClamped(2,3)=%d want 5", got)
	}
}
