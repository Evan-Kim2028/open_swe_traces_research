package retry

import (
	"math"
	"math/rand"
	"testing"
)

func envelope(base, cap, n int) int {
	return int(math.Min(float64(cap), float64(base)*math.Pow(2.0, float64(n))))
}

func TestBackoffExponentialProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(20260918))
	for i := 0; i < 10000; i++ {
		base := rng.Intn(64) + 1
		capv := base + rng.Intn(8000)
		n := rng.Intn(24)
		got := expo(base, capv, n)
		want := envelope(base, capv, n)
		if got != want {
			t.Fatalf("case %d: expo(%d,%d,%d)=%d want %d", i, base, capv, n, got, want)
		}
	}
	adversarial := [][4]int{
		{2, 500, 0, 2},
		{2, 500, 1, 4},
		{2, 500, 8, 500},
		{1, 1, 0, 1},
		{100, 2000, 0, 100},
		{100, 2000, 1, 200},
		{100, 2000, 10, 2000},
		{3, 3, 5, 3},
		{2, 500, 40, 500},
		{2, 500, 80, 500},
		{7, 7, 0, 7},
		{9, 100000, 20, 100000},
	}
	for _, c := range adversarial {
		got := expo(c[0], c[1], c[2])
		if got != c[3] {
			t.Fatalf("expo(%d,%d,%d)=%d want %d", c[0], c[1], c[2], got, c[3])
		}
	}
	prev := 0
	for n := 0; n < 20; n++ {
		v := expo(2, 500, n)
		if v < prev {
			t.Fatalf("not monotone at n=%d: %d < %d", n, v, prev)
		}
		if v > 500 {
			t.Fatalf("exceeds cap: %d", v)
		}
		prev = v
	}
}
