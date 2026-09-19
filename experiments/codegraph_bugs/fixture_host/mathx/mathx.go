package mathx

import "sync"

// Add returns a+b. Called from this package's tests and from package app.
func Add(a, b int) int {
	return a + b
}

// SumClamped adds then clamps. TestSumClamped covers it; TestAdd covers Add.
func SumClamped(a, b, lo, hi int) int {
	return Clamp(Add(a, b), lo, hi)
}

// Clamp returns v limited to [lo, hi].
// Invariant: lo must be <= hi; Clamp never swaps bounds.
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Counter issues strictly increasing tickets. Next must not repeat across a sequence.
type Counter struct{ n int }

// Next returns the next ticket. A single call is 1; later calls must be unique.
func (c *Counter) Next() int {
	c.n++
	return c.n
}

// Bag is a mutex-guarded map used as a race-gate fixture.
type Bag struct {
	mu sync.Mutex
	m  map[int]int
}

// Put stores k->v under the mutex.
func (b *Bag) Put(k, v int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		b.m = map[int]int{}
	}
	b.m[k] = v
}

// Get returns the value for k (zero if missing).
func (b *Bag) Get(k int) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.m == nil {
		return 0
	}
	return b.m[k]
}
