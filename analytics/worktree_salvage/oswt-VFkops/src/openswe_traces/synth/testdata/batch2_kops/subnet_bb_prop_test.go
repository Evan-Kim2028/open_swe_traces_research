package subnet_test

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"example.internal/kops/pkg/util/subnet"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func mustCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Detail 1: Overlap returns false when either side is nil (no panic).
func TestDetail01_OverlapNilSafe(t *testing.T) {
	n := mustCIDR(t, "10.0.0.0/8")
	if subnet.Overlap(nil, n) || subnet.Overlap(n, nil) || subnet.Overlap(nil, nil) {
		t.Fatalf("Overlap with nil returned true")
	}
}

// Detail 2: Overlap is symmetric and true iff either network's base IP is
// contained in the other — 0.0.0.0/0 overlaps everything.
func TestDetail02_OverlapBaseContainment(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	if !subnet.Overlap(mustCIDR(t, "0.0.0.0/0"), mustCIDR(t, "192.168.3.4/32")) {
		t.Fatalf("/0 does not overlap everything")
	}
	for i := 0; i < 300; i++ {
		a := fmt.Sprintf("%d.%d.%d.%d/%d", rng.Intn(224), rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(33))
		b := fmt.Sprintf("%d.%d.%d.%d/%d", rng.Intn(224), rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(33))
		na, nb := mustCIDR(t, a), mustCIDR(t, b)
		got := subnet.Overlap(na, nb)
		if got != subnet.Overlap(nb, na) {
			t.Fatalf("i=%d Overlap asymmetric for %s %s", i, a, b)
		}
		want := na.Contains(nb.IP) || nb.Contains(na.IP)
		if got != want {
			t.Fatalf("i=%d Overlap(%s,%s)=%v want %v", i, a, b, got, want)
		}
	}
}

// Detail 3: BelongsTo requires same address family, parent ones <= child
// ones, masked-base equality; host bits ignored; equal networks count.
func TestDetail03_BelongsTo(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		pOnes := rng.Intn(25)
		cOnes := pOnes + rng.Intn(33-pOnes)
		octets := [4]int{rng.Intn(224), rng.Intn(256), rng.Intn(256), rng.Intn(256)}
		parent := mustCIDR(t, fmt.Sprintf("%d.%d.%d.%d/%d", octets[0], octets[1], octets[2], octets[3], pOnes))
		// Build child inside parent: parent base + random host bits, longer mask.
		child := &net.IPNet{IP: make(net.IP, 4), Mask: net.CIDRMask(cOnes, 32)}
		copy(child.IP, parent.IP)
		for bit := pOnes; bit < 32; bit++ {
			if rng.Intn(2) == 0 {
				child.IP[bit/8] |= 1 << (7 - uint(bit%8))
			}
		}
		// Child must be masked within parent for containment — compute expected.
		contained := parent.Contains(child.IP.Mask(child.Mask)) && pOnes <= cOnes
		got := subnet.BelongsTo(parent, child)
		if got != contained {
			t.Fatalf("i=%d BelongsTo(%v,%v)=%v want %v", i, parent, child, got, contained)
		}
		// Host bits on the child IP beyond child.Mask are ignored? The
		// contract is masked-base equality: child masked vs parent masked.
	}
	// Equal networks count.
	n := mustCIDR(t, "10.1.2.3/16")
	if !subnet.BelongsTo(n, mustCIDR(t, "10.1.9.9/16")) {
		t.Fatalf("equal networks don't belong")
	}
	// Parent longer than child -> false.
	if subnet.BelongsTo(mustCIDR(t, "10.0.0.0/24"), mustCIDR(t, "10.0.0.0/16")) {
		t.Fatalf("longer parent belongs")
	}
	// Mixed family -> false.
	v6 := mustCIDR(t, "::/0")
	if subnet.BelongsTo(v6, mustCIDR(t, "10.0.0.0/8")) || subnet.BelongsTo(mustCIDR(t, "10.0.0.0/8"), v6) {
		t.Fatalf("mixed family belongs")
	}
}

// Detail 4: SplitInto yields 2^N subnets of length parentLen+N ascending
// from the parent base; SplitInto1/2/4/8 pass 0/1/2/3.
func TestDetail04_SplitIntoShape(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 150; i++ {
		ones := rng.Intn(25)
		bits := uint(rng.Intn(4)) // 0..3
		base := fmt.Sprintf("%d.%d.0.0/%d", rng.Intn(200), rng.Intn(256), ones)
		parent := mustCIDR(t, base)
		if ones+int(bits) > 32 {
			continue
		}
		var subs []*net.IPNet
		var err error
		switch bits {
		case 0:
			subs, err = subnet.SplitInto1(parent)
		case 1:
			subs, err = subnet.SplitInto2(parent)
		case 2:
			subs, err = subnet.SplitInto4(parent)
		case 3:
			subs, err = subnet.SplitInto8(parent)
		}
		if err != nil {
			t.Fatalf("i=%d SplitInto%d(%s) err %v", i, 1<<bits, base, err)
		}
		want := 1 << bits
		if len(subs) != want {
			t.Fatalf("i=%d got %d subnets want %d", i, len(subs), want)
		}
		newOnes := ones + int(bits)
		prev := uint32(0)
		for j, s := range subs {
			o, sz := s.Mask.Size()
			if o != newOnes || sz != 32 {
				t.Fatalf("i=%d sub %d mask %v want /%d", i, j, s.Mask, newOnes)
			}
			// Ascending order.
			ip := s.IP.To4()
			v := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
			if j > 0 && v <= prev {
				t.Fatalf("i=%d sub %d %v not ascending", i, j, s)
			}
			prev = v
			// First sub starts at parent base.
			if j == 0 && !s.IP.Equal(parent.IP) {
				t.Fatalf("i=%d first sub %v not parent base %v", i, s, parent)
			}
		}
	}
	// SplitInto general form matches the named helpers.
	subs, err := subnet.SplitInto(3, mustCIDR(t, "10.0.0.0/16"))
	if err != nil || len(subs) != 8 {
		t.Fatalf("SplitInto(3)=%d err %v", len(subs), err)
	}
}

// Detail 5: SplitInto errors `unexpected IP address type: %s` on non-IPv4.
func TestDetail05_SplitIntoIPv6Errors(t *testing.T) {
	for i, s := range []string{"::/0", "2001:db8::/32", "fd00::/8"} {
		parent := mustCIDR(t, s)
		_, err := subnet.SplitInto(1, parent)
		if err == nil {
			t.Fatalf("i=%d SplitInto(%s) accepted IPv6", i, s)
		}
		if !strings.Contains(err.Error(), "unexpected IP address type") {
			t.Fatalf("i=%d err %q lacks 'unexpected IP address type'", i, err.Error())
		}
	}
}

// Detail 6: Allocate increments the candidate BEFORE the first in-range
// test — the `from` base block is never returned.
func TestDetail06_AllocateSkipsFromBase(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		from := fmt.Sprintf("10.%d.0.0/16", rng.Intn(200))
		cm := &subnet.CIDRMap{}
		got, err := cm.Allocate(from, net.CIDRMask(24, 32))
		if err != nil {
			t.Fatalf("i=%d Allocate err %v", i, err)
		}
		// Returned block must be inside `from` and not the base block.
		fn := mustCIDR(t, from)
		if !fn.Contains(got.IP) {
			t.Fatalf("i=%d allocated %v outside %v", i, got, fn)
		}
		baseBlock := &net.IPNet{IP: fn.IP, Mask: net.CIDRMask(24, 32)}
		if got.String() == baseBlock.String() {
			t.Fatalf("i=%d returned base block %v", i, got)
		}
	}
}

// Detail 7: Allocate bounds the search by overlap-with-from and returns
// `cannot allocate CIDR of size %v` when exhausted; records each success.
func TestDetail07_AllocateExhaustion(t *testing.T) {
	// /30 from has exactly 4 /32s; base skipped, so 3 allocations then error.
	cm := &subnet.CIDRMap{}
	var got []string
	for i := 0; i < 10; i++ {
		n, err := cm.Allocate("10.0.0.0/30", net.CIDRMask(32, 32))
		if err != nil {
			if !strings.Contains(err.Error(), "cannot allocate CIDR of size") {
				t.Fatalf("alloc %d err %q", i, err.Error())
			}
			break
		}
		got = append(got, n.String())
	}
	if len(got) != 3 {
		t.Fatalf("got %d allocations %v want 3", len(got), got)
	}
	for _, g := range got {
		if g == "10.0.0.0/32" {
			t.Fatalf("base block allocated")
		}
	}
	// Exhaustion error mentions the mask size.
	_, err := cm.Allocate("10.0.0.0/30", net.CIDRMask(32, 32))
	if err == nil || !strings.Contains(err.Error(), "cannot allocate CIDR of size") {
		t.Fatalf("exhaustion err %v", err)
	}
}

// Detail 8: used-range membership is overlap, not exact equality — marking
// a broad CIDR blocks sub-blocks; marking a /27 blocks enclosing allocs.
func TestDetail08_UsedOverlapNotEquality(t *testing.T) {
	cm := &subnet.CIDRMap{}
	if err := cm.MarkInUse("10.0.0.0/8"); err != nil {
		t.Fatal(err)
	}
	// /8 in use -> sub-block inside must not be returned; allocate /24 in
	// 10.0.0.0/16 — every /24 is inside /8, so allocation must exhaust.
	_, err := cm.Allocate("10.0.0.0/16", net.CIDRMask(24, 32))
	if err == nil {
		t.Fatalf("allocation inside used /8 succeeded")
	}
	cm2 := &subnet.CIDRMap{}
	if err := cm2.MarkInUse("10.0.1.0/27"); err != nil {
		t.Fatal(err)
	}
	// Allocating a /24 covering that /27 must skip past it.
	n, err := cm2.Allocate("10.0.0.0/16", net.CIDRMask(24, 32))
	if err != nil {
		t.Fatal(err)
	}
	if n.String() == "10.0.1.0/24" {
		t.Fatalf("allocated enclosing block over used /27")
	}
}

// Detail 9: MarkInUse/Allocate wrap parse failures as
// `error parsing network cidr %q: %v` / `error parsing CIDR %q: %v`;
// Allocate supports IPv6.
func TestDetail09_ParseErrorWrapsAndIPv6(t *testing.T) {
	cm := &subnet.CIDRMap{}
	err := cm.MarkInUse("bogus")
	if err == nil || !strings.HasPrefix(err.Error(), `error parsing network cidr "bogus": `) {
		t.Fatalf("MarkInUse err %v", err)
	}
	_, err = cm.Allocate("bogus", net.CIDRMask(24, 32))
	if err == nil || !strings.HasPrefix(err.Error(), `error parsing CIDR "bogus": `) {
		t.Fatalf("Allocate err %v", err)
	}
	// IPv6 allocate works.
	cm2 := &subnet.CIDRMap{}
	n, err := cm2.Allocate("fd00::/64", net.CIDRMask(80, 128))
	if err != nil {
		t.Fatalf("IPv6 Allocate err %v", err)
	}
	if n.IP.To16() == nil {
		t.Fatalf("IPv6 result %v", n)
	}
}
