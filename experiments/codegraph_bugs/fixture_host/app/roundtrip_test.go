package app

import (
	"testing"

	"fixturehost/types"
)

func TestRoundTrip(t *testing.T) {
	p := types.Point{X: 3, Y: 4}
	got := RoundTrip(p)
	if got != p {
		t.Fatalf("RoundTrip=%v want %v", got, p)
	}
}
