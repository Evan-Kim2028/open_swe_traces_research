package subnet

import (
	"net"
	"strings"
	"testing"
	"time"
)

func mustCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", s, err)
	}
	return n
}

// TestDetail01 — Overlap tests base-address containment (either direction) and nil-guards.
func TestDetail01(t *testing.T) {
	if !Overlap(mustCIDR(t, "10.0.0.0/8"), mustCIDR(t, "10.64.0.0/10")) {
		t.Error("parent/child pair should overlap")
	}
	if !Overlap(mustCIDR(t, "10.64.0.0/10"), mustCIDR(t, "10.0.0.0/8")) {
		t.Error("overlap should be symmetric for parent/child")
	}
	if Overlap(mustCIDR(t, "10.1.0.0/16"), mustCIDR(t, "10.2.0.0/16")) {
		t.Error("disjoint same-family networks should not overlap")
	}
	if Overlap(nil, mustCIDR(t, "10.0.0.0/8")) || Overlap(mustCIDR(t, "10.0.0.0/8"), nil) || Overlap(nil, nil) {
		t.Error("Overlap should nil-guard")
	}
}

// TestDetail02 — BelongsTo requires equal family width, child prefix at least as long,
// and masked-base equality.
func TestDetail02(t *testing.T) {
	if !BelongsTo(mustCIDR(t, "10.0.0.0/8"), mustCIDR(t, "10.64.0.0/10")) {
		t.Error("10.64.0.0/10 belongs to 10.0.0.0/8")
	}
	if BelongsTo(mustCIDR(t, "10.0.0.0/16"), mustCIDR(t, "10.0.0.0/8")) {
		t.Error("child longer than parent is required: /8 does not belong to /16")
	}
	if !BelongsTo(mustCIDR(t, "10.0.0.0/8"), mustCIDR(t, "10.0.0.0/8")) {
		t.Error("a network belongs to itself")
	}
	if BelongsTo(mustCIDR(t, "10.0.0.0/8"), mustCIDR(t, "11.0.0.0/10")) {
		t.Error("masked bases differ: 11.0.0.0/10 does not belong to 10.0.0.0/8")
	}
	if BelongsTo(mustCIDR(t, "fd00::/8"), mustCIDR(t, "10.0.0.0/8")) {
		t.Error("different address families should not belong")
	}
	if !BelongsTo(mustCIDR(t, "fd00::/8"), mustCIDR(t, "fd00::/16")) {
		t.Error("v6 child should belong to v6 parent")
	}
}

// TestDetail03 — SplitInto enumerates subnets in order and errors on IPv6 parents.
func TestDetail03(t *testing.T) {
	got, err := SplitInto4(mustCIDR(t, "10.0.0.0/8"))
	if err != nil {
		t.Fatalf("SplitInto4: %v", err)
	}
	want := []string{"10.0.0.0/10", "10.64.0.0/10", "10.128.0.0/10", "10.192.0.0/10"}
	if len(got) != len(want) {
		t.Fatalf("SplitInto4 returned %d subnets, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Errorf("SplitInto4[%d] = %s, want %s", i, got[i], want[i])
		}
	}

	got, err = SplitInto2(mustCIDR(t, "10.0.0.0/8"))
	if err != nil {
		t.Fatalf("SplitInto2: %v", err)
	}
	if len(got) != 2 || got[0].String() != "10.0.0.0/9" || got[1].String() != "10.128.0.0/9" {
		t.Errorf("SplitInto2 = %v, want [10.0.0.0/9 10.128.0.0/9]", got)
	}
	got, err = SplitInto1(mustCIDR(t, "10.0.0.0/8"))
	if err != nil {
		t.Fatalf("SplitInto1: %v", err)
	}
	if len(got) != 1 || got[0].String() != "10.0.0.0/8" {
		t.Errorf("SplitInto1 = %v, want the parent subnet itself", got)
	}

	if _, err := SplitInto2(mustCIDR(t, "fd00::/8")); err == nil {
		t.Error("expected error splitting an IPv6 parent")
	}
}

// TestDetail04 — incrementIP steps by subnet size (1 << (bits - ones)) with carry
// propagation into the high half for v6.
func TestDetail04(t *testing.T) {
	ip := net.ParseIP("10.0.0.0").To4()
	if err := incrementIP(ip, net.CIDRMask(24, 32)); err != nil {
		t.Fatalf("incrementIP: %v", err)
	}
	if !ip.Equal(net.ParseIP("10.0.1.0")) {
		t.Errorf("10.0.0.0/24 incremented = %v, want 10.0.1.0", ip)
	}

	ip = net.ParseIP("10.0.255.0").To4()
	if err := incrementIP(ip, net.CIDRMask(24, 32)); err != nil {
		t.Fatalf("incrementIP: %v", err)
	}
	if !ip.Equal(net.ParseIP("10.1.0.0")) {
		t.Errorf("carry within v4 failed: got %v, want 10.1.0.0", ip)
	}

	// /64 on v6 steps 1<<64 — lands on the high/low half boundary (increments high half).
	ip = net.ParseIP("2001:db8::ffff:ffff:ffff:ffff").To16()
	if err := incrementIP(ip, net.CIDRMask(64, 128)); err != nil {
		t.Fatalf("incrementIP v6: %v", err)
	}
	if !ip.Equal(net.ParseIP("2001:db8:0:1:ffff:ffff:ffff:ffff")) {
		t.Errorf("v6 /64 step failed: got %v, want 2001:db8:0:1:ffff:ffff:ffff:ffff", ip)
	}

	// /65 steps 1<<63 within the low half; a full low half carries into the high half.
	ip = net.ParseIP("2001:db8::ffff:ffff:ffff:ffff").To16()
	if err := incrementIP(ip, net.CIDRMask(65, 128)); err != nil {
		t.Fatalf("incrementIP v6 /65: %v", err)
	}
	if !ip.Equal(net.ParseIP("2001:db8:0:1:7fff:ffff:ffff:ffff")) {
		t.Errorf("v6 carry into high half failed: got %v, want 2001:db8:0:1:7fff:ffff:ffff:ffff", ip)
	}
}

// TestDetail05 — Allocate increments before the first candidate (base subnet skipped),
// marks the winner in use, and fails when nothing non-overlapping remains.
func TestDetail05(t *testing.T) {
	c := &CIDRMap{}
	got, err := c.Allocate("10.0.0.0/8", net.CIDRMask(24, 32))
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if got.String() != "10.0.1.0/24" {
		t.Errorf("Allocate returned %s, want 10.0.1.0/24 (base subnet skipped)", got)
	}

	got, err = c.Allocate("10.0.0.0/8", net.CIDRMask(24, 32))
	if err != nil {
		t.Fatalf("second Allocate: %v", err)
	}
	if got.String() != "10.0.2.0/24" {
		t.Errorf("second Allocate returned %s, want 10.0.2.0/24 (winner marked in use)", got)
	}

	// Exhaustion: mark the whole range in use; allocation must fail (bounded, not hang).
	c2 := &CIDRMap{}
	if err := c2.MarkInUse("10.0.0.0/8"); err != nil {
		t.Fatalf("MarkInUse: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		var err2 error
		_, err2 = c2.Allocate("10.0.0.0/8", net.CIDRMask(24, 32))
		done <- err2
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected exhaustion error when the whole range is in use")
		}
	case <-time.After(30 * time.Second):
		t.Error("Allocate did not terminate on an exhausted range")
	}
}

// TestDetail06 — parse errors are wrapped; the error names the offending input (shape only).
func TestDetail06(t *testing.T) {
	c := &CIDRMap{}
	if err := c.MarkInUse("bogus-cidr"); err == nil {
		t.Error("MarkInUse should error on a bad CIDR")
	} else if !strings.Contains(err.Error(), "bogus-cidr") {
		t.Errorf("MarkInUse error should name the offending input, got %q", err.Error())
	}
	if _, err := c.Allocate("bogus-cidr", net.CIDRMask(24, 32)); err == nil {
		t.Error("Allocate should error on a bad CIDR")
	} else if !strings.Contains(err.Error(), "bogus-cidr") {
		t.Errorf("Allocate error should name the offending input, got %q", err.Error())
	}
}
