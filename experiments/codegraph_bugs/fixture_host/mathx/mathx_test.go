package mathx

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Fatalf("Add(2,3)=%d", got)
	}
}

func TestSumClamped(t *testing.T) {
	if got := SumClamped(2, 3, 0, 4); got != 4 {
		t.Fatalf("SumClamped=%d", got)
	}
}

func TestClamp(t *testing.T) {
	if got := Clamp(10, 0, 5); got != 5 {
		t.Fatalf("Clamp=%d", got)
	}
}

func TestNextOnce(t *testing.T) {
	var c Counter
	if got := c.Next(); got != 1 {
		t.Fatalf("Next=%d", got)
	}
}

func TestNextSequence(t *testing.T) {
	var c Counter
	seen := map[int]struct{}{}
	for i := 0; i < 8; i++ {
		n := c.Next()
		if _, ok := seen[n]; ok {
			t.Fatalf("dup %d", n)
		}
		seen[n] = struct{}{}
	}
	if len(seen) != 8 {
		t.Fatalf("unique=%d", len(seen))
	}
}

func TestBag(t *testing.T) {
	var b Bag
	b.Put(1, 2)
	if got := b.Get(1); got != 2 {
		t.Fatalf("Get= %d", got)
	}
}

func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(i, i)
	}
}
