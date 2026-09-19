package retry

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func TestBackoff(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	_ = rng
}
