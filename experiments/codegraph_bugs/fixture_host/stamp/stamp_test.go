package stamp

import (
	"testing"
	"time"
)

func TestComposeExtract(t *testing.T) {
	ts := Compose(9, 2)
	if Extract(ts) != 9 {
		t.Fatalf("Extract=%d", Extract(ts))
	}
	now := time.Unix(1, 0)
	if Extract(FromTime(now)) != now.UnixMilli() {
		t.Fatalf("FromTime mismatch")
	}
}
