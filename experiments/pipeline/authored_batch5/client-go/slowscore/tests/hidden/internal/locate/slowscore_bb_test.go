package locate

import (
	"strconv"
	"testing"
	"time"
)

// Hidden suite for unit slowscore. One TestDetailNN per DETAILS.md line.

// TestDetail01: the window keeps at most slidingWindowSize samples, evicting
// the oldest; Avg is integer-divided sum/len.
func TestDetail01(t *testing.T) {
	w := &CountSlidingWindow{}
	for i := uint64(1); i <= 10; i++ {
		w.Append(i)
	}
	if w.Sum() != 55 || w.Avg() != 5 {
		t.Fatalf("below-cap: sum=%d avg=%d", w.Sum(), w.Avg())
	}
	w.Append(11)
	if w.Sum() != 65 || w.Avg() != 6 {
		t.Fatalf("at cap the oldest was not evicted: sum=%d avg=%d", w.Sum(), w.Avg())
	}
	w.Append(12)
	if w.Sum() != 75 || w.Avg() != 7 {
		t.Fatalf("window did not keep the newest samples: sum=%d avg=%d", w.Sum(), w.Avg())
	}
	// Integer division.
	w2 := &CountSlidingWindow{}
	w2.Append(1)
	w2.Append(2)
	if w2.Avg() != 1 {
		t.Fatalf("Avg = %d, want integer division 1", w2.Avg())
	}
}

// TestDetail02: Append returns a signed gradient reflecting direction;
// identical or baseline-free values return a small positive floor.
func TestDetail02(t *testing.T) {
	w := &CountSlidingWindow{}
	g := w.Append(10)
	if g <= 0 || g >= 0.01 {
		t.Fatalf("first append gradient = %v, want a small positive floor", g)
	}
	if g := w.Append(20); g <= 0 {
		t.Fatalf("rising value gave gradient %v", g)
	}
	if g := w.Append(5); g >= 0 {
		t.Fatalf("falling value gave gradient %v", g)
	}
	w2 := &CountSlidingWindow{}
	w2.Append(7)
	if g := w2.Append(7); g <= 0 || g >= 0.01 {
		t.Fatalf("equal value gave gradient %v, want a small positive floor", g)
	}
}

// TestDetail03: updateSlowScore on an uninitialized stat only seeds it.
func TestDetail03(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.updateSlowScore()
	if ss.getSlowScore() != 1 {
		t.Fatalf("seeded score = %d, want 1", ss.getSlowScore())
	}
	if ss.isSlow() {
		t.Fatalf("seeded stat is slow")
	}
	// The seed must have initialized the stat: a subsequent huge request
	// takes the initialized path rather than re-seeding.
	ss.recordSlowScoreStat(31 * time.Second)
	if ss.getSlowScore() != slowScoreMax {
		t.Fatalf("post-seed stat did not reach the max path: %d", ss.getSlowScore())
	}
}

// TestDetail04: with zero interval updates only the decay path can run —
// the score never rises without requests.
func TestDetail04(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.updateSlowScore() // seed
	prev := ss.getSlowScore()
	for i := 0; i < 5; i++ {
		ss.updateSlowScore()
		if ss.getSlowScore() > prev {
			t.Fatalf("score rose with zero updates: %d -> %d", prev, ss.getSlowScore())
		}
		prev = ss.getSlowScore()
	}
}

// TestDetail05: the score rises when throughput falls while latency rises,
// bounded by the max score.
func TestDetail05(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.updateSlowScore() // seed
	// Steady baseline: several ticks of identical traffic.
	for i := 0; i < 6; i++ {
		for j := 0; j < 10; j++ {
			ss.recordSlowScoreStat(100 * time.Microsecond)
		}
		ss.updateSlowScore()
	}
	// Degraded tick: much lower update count, much higher timecost.
	for j := 0; j < 2; j++ {
		ss.recordSlowScoreStat(5 * time.Millisecond)
	}
	ss.updateSlowScore()
	if ss.getSlowScore() <= 1 {
		t.Fatalf("score did not rise under falling throughput + rising latency: %d", ss.getSlowScore())
	}
	if ss.getSlowScore() > slowScoreMax {
		t.Fatalf("score exceeded max: %d", ss.getSlowScore())
	}
	// A healthy tick (more updates, lower latency) must not raise it further.
	before := ss.getSlowScore()
	for j := 0; j < 30; j++ {
		ss.recordSlowScoreStat(10 * time.Microsecond)
	}
	ss.updateSlowScore()
	if ss.getSlowScore() > before {
		t.Fatalf("score rose on a healthy tick: %d -> %d", before, ss.getSlowScore())
	}
}

// TestDetail06: otherwise the score decays toward 1 and snaps to it.
func TestDetail06(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.updateSlowScore()
	for i := 0; i < 6; i++ {
		for j := 0; j < 10; j++ {
			ss.recordSlowScoreStat(100 * time.Microsecond)
		}
		ss.updateSlowScore()
	}
	for j := 0; j < 2; j++ {
		ss.recordSlowScoreStat(5 * time.Millisecond)
	}
	ss.updateSlowScore()
	if ss.getSlowScore() <= 1 {
		t.Fatalf("setup did not raise score: %d", ss.getSlowScore())
	}
	// Quiet ticks decay monotonically and reach the init value.
	prev := ss.getSlowScore()
	for i := 0; i < 120 && ss.getSlowScore() != slowScoreInitVal; i++ {
		ss.updateSlowScore()
		if ss.getSlowScore() > prev {
			t.Fatalf("score rose during decay: %d -> %d", prev, ss.getSlowScore())
		}
		prev = ss.getSlowScore()
	}
	if ss.getSlowScore() != slowScoreInitVal {
		t.Fatalf("score did not decay to init: %d", ss.getSlowScore())
	}
}

// TestDetail07: after a tick the interval counters are zeroed — a second
// tick without new records behaves like a zero-update tick.
func TestDetail07(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.updateSlowScore() // seed
	for j := 0; j < 10; j++ {
		ss.recordSlowScoreStat(100 * time.Microsecond)
	}
	ss.updateSlowScore()
	scoreAfterLoad := ss.getSlowScore()
	// If interval counters were not reset, the stale count would keep the
	// update path alive; two consecutive quiet ticks must behave identically.
	ss.updateSlowScore()
	a := ss.getSlowScore()
	ss.updateSlowScore()
	b := ss.getSlowScore()
	if a > scoreAfterLoad || b > a {
		t.Fatalf("quiet ticks did not behave identically: load=%d a=%d b=%d", scoreAfterLoad, a, b)
	}
}

// TestDetail08: recordSlowScoreStat on an uninitialized stat only seeds;
// once initialized, a single request at the max timeout pins the score.
func TestDetail08(t *testing.T) {
	ss := &SlowScoreStat{}
	ss.recordSlowScoreStat(31 * time.Second)
	if ss.getSlowScore() != slowScoreInitVal {
		t.Fatalf("uninitialized stat did not only seed: %d", ss.getSlowScore())
	}
	// Now initialized: the same request pins to max.
	ss.recordSlowScoreStat(31 * time.Second)
	if ss.getSlowScore() != slowScoreMax {
		t.Fatalf("max-timeout request did not pin the score: %d", ss.getSlowScore())
	}
}

// TestDetail09: isSlow at the threshold; markAlreadySlow and resetSlowScore
// pin their ends.
func TestDetail09(t *testing.T) {
	if (&SlowScoreStat{avgScore: slowScoreThreshold}).isSlow() != true {
		t.Fatalf("threshold score not slow")
	}
	if (&SlowScoreStat{avgScore: slowScoreThreshold - 1}).isSlow() {
		t.Fatalf("below-threshold score is slow")
	}
	ss := &SlowScoreStat{}
	ss.markAlreadySlow()
	if !ss.isSlow() || ss.getSlowScore() != slowScoreMax {
		t.Fatalf("markAlreadySlow did not pin the max: %d", ss.getSlowScore())
	}
	ss.resetSlowScore()
	if ss.isSlow() || ss.getSlowScore() != 0 {
		t.Fatalf("resetSlowScore did not zero: %d", ss.getSlowScore())
	}
}

// TestDetail10: replicaFlowsType renders the named flows distinctly and any
// other value as its decimal.
func TestDetail10(t *testing.T) {
	if toLeader.String() == toFollower.String() {
		t.Fatalf("leader/follower rendered identically: %q", toLeader.String())
	}
	for _, v := range []replicaFlowsType{toLeader, toFollower} {
		if s := v.String(); s == "" || s == strconv.Itoa(int(v)) {
			t.Fatalf("named flow %d rendered as %q", v, s)
		}
	}
	for _, v := range []replicaFlowsType{numReplicaFlowsType, replicaFlowsType(9), replicaFlowsType(255)} {
		if got := v.String(); got != strconv.Itoa(int(v)) {
			t.Fatalf("unlisted flow %d rendered as %q, want its decimal", v, got)
		}
	}
}
