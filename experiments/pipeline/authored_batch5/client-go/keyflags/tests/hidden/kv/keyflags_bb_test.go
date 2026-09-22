package kv

import (
	"testing"
)

// Hidden suite for unit keyflags. One TestDetailNN per DETAILS.md line.

var bbAllOps = []FlagsOp{
	SetPresumeKeyNotExists, DelPresumeKeyNotExists,
	SetKeyLocked, DelKeyLocked,
	SetNeedLocked, DelNeedLocked,
	SetKeyLockedValueExists, SetKeyLockedValueNotExists,
	DelNeedCheckExists,
	SetPrewriteOnly, SetIgnoredIn2PC, SetReadable, SetNewlyInserted,
	SetAssertExist, SetAssertNotExist, SetAssertUnknown, SetAssertNone,
	SetNeedConstraintCheckInPrewrite, DelNeedConstraintCheckInPrewrite,
	SetPreviousPresumeKNE,
}

// TestDetail01: no flag op ever sets the reserved top bit of the uint16.
func TestDetail01(t *testing.T) {
	f := ApplyFlagsOps(0, bbAllOps...)
	if f&(1<<15) != 0 {
		t.Fatalf("top bit set after applying all ops: %#x", f)
	}
	// Individually as well.
	for _, op := range bbAllOps {
		if g := ApplyFlagsOps(0, op); g&(1<<15) != 0 {
			t.Fatalf("op %d set the reserved top bit: %#x", op, g)
		}
	}
}

// TestDetail02: the assertion pair truth table — exclusive Exist/NotExist,
// both = unknown, either = assertion set.
func TestDetail02(t *testing.T) {
	base := KeyFlags(0)
	if base.HasAssertExist() || base.HasAssertNotExist() || base.HasAssertUnknown() || base.HasAssertionFlags() {
		t.Fatalf("assertion predicates true on zero flags")
	}
	e := ApplyFlagsOps(0, SetAssertExist)
	if !e.HasAssertExist() || e.HasAssertNotExist() || e.HasAssertUnknown() || !e.HasAssertionFlags() {
		t.Fatalf("SetAssertExist: exist=%v notexist=%v unknown=%v any=%v",
			e.HasAssertExist(), e.HasAssertNotExist(), e.HasAssertUnknown(), e.HasAssertionFlags())
	}
	n := ApplyFlagsOps(0, SetAssertNotExist)
	if n.HasAssertExist() || !n.HasAssertNotExist() || n.HasAssertUnknown() || !n.HasAssertionFlags() {
		t.Fatalf("SetAssertNotExist: exist=%v notexist=%v unknown=%v any=%v",
			n.HasAssertExist(), n.HasAssertNotExist(), n.HasAssertUnknown(), n.HasAssertionFlags())
	}
	u := ApplyFlagsOps(0, SetAssertUnknown)
	if u.HasAssertExist() || u.HasAssertNotExist() || !u.HasAssertUnknown() || !u.HasAssertionFlags() {
		t.Fatalf("SetAssertUnknown: exist=%v notexist=%v unknown=%v any=%v",
			u.HasAssertExist(), u.HasAssertNotExist(), u.HasAssertUnknown(), u.HasAssertionFlags())
	}
	z := ApplyFlagsOps(u, SetAssertNone)
	if z.HasAssertExist() || z.HasAssertNotExist() || z.HasAssertUnknown() || z.HasAssertionFlags() {
		t.Fatalf("SetAssertNone did not clear the pair")
	}
}

// TestDetail03: HasPresumeKeyNotExists reads the live OR the previous presume
// bit.
func TestDetail03(t *testing.T) {
	if ApplyFlagsOps(0).HasPresumeKeyNotExists() {
		t.Fatalf("HasPresumeKeyNotExists on zero flags")
	}
	if !ApplyFlagsOps(0, SetPresumeKeyNotExists).HasPresumeKeyNotExists() {
		t.Fatalf("SetPresumeKeyNotExists not observed")
	}
	if !ApplyFlagsOps(0, SetPreviousPresumeKNE).HasPresumeKeyNotExists() {
		t.Fatalf("SetPreviousPresumeKNE not observed by HasPresumeKeyNotExists")
	}
	if !ApplyFlagsOps(0, SetPresumeKeyNotExists, SetPreviousPresumeKNE).HasPresumeKeyNotExists() {
		t.Fatalf("both presume bits set but predicate false")
	}
}

// TestDetail04: ops apply left to right; SetPresumeKeyNotExists implies
// HasNeedCheckExists and DelPresumeKeyNotExists clears both.
func TestDetail04(t *testing.T) {
	f := ApplyFlagsOps(0, SetPresumeKeyNotExists)
	if !f.HasPresumeKeyNotExists() || !f.HasNeedCheckExists() {
		t.Fatalf("SetPresumeKeyNotExists: presume=%v needCheck=%v", f.HasPresumeKeyNotExists(), f.HasNeedCheckExists())
	}
	g := ApplyFlagsOps(0, SetPresumeKeyNotExists, DelPresumeKeyNotExists)
	if g.HasPresumeKeyNotExists() || g.HasNeedCheckExists() {
		t.Fatalf("DelPresumeKeyNotExists left presume=%v needCheck=%v", g.HasPresumeKeyNotExists(), g.HasNeedCheckExists())
	}
	// Order matters: the later op wins.
	h := ApplyFlagsOps(0, DelPresumeKeyNotExists, SetPresumeKeyNotExists)
	if !h.HasPresumeKeyNotExists() {
		t.Fatalf("left-to-right violated: Del then Set did not set")
	}
}

// TestDetail05: SetKeyLockedValueExists sets the value-exists bit and
// SetKeyLockedValueNotExists clears it. (The cross-flag clearing of the
// constraint-check bit is internal and not asserted.)
func TestDetail05(t *testing.T) {
	f := ApplyFlagsOps(0, SetKeyLockedValueExists)
	if !f.HasLockedValueExists() {
		t.Fatalf("SetKeyLockedValueExists not observed")
	}
	g := ApplyFlagsOps(f, SetKeyLockedValueNotExists)
	if g.HasLockedValueExists() {
		t.Fatalf("SetKeyLockedValueNotExists did not clear the value bit")
	}
}

// TestDetail06: assert ops keep the pair consistent — setting one side clears
// the other.
func TestDetail06(t *testing.T) {
	f := ApplyFlagsOps(0, SetAssertExist, SetAssertNotExist)
	if f.HasAssertExist() || !f.HasAssertNotExist() || f.HasAssertUnknown() {
		t.Fatalf("Exist then NotExist: exist=%v notexist=%v unknown=%v",
			f.HasAssertExist(), f.HasAssertNotExist(), f.HasAssertUnknown())
	}
	g := ApplyFlagsOps(0, SetAssertNotExist, SetAssertExist)
	if !g.HasAssertExist() || g.HasAssertNotExist() || g.HasAssertUnknown() {
		t.Fatalf("NotExist then Exist: exist=%v notexist=%v unknown=%v",
			g.HasAssertExist(), g.HasAssertNotExist(), g.HasAssertUnknown())
	}
	u := ApplyFlagsOps(0, SetAssertExist, SetAssertUnknown)
	if !u.HasAssertUnknown() {
		t.Fatalf("SetAssertUnknown after Exist did not mark unknown")
	}
	n := ApplyFlagsOps(u, SetAssertNone)
	if n.HasAssertionFlags() {
		t.Fatalf("SetAssertNone left assertion flags")
	}
}

// TestDetail07: AndPersistent keeps only the persistent flag set (visible in
// the persistentFlags constant).
func TestDetail07(t *testing.T) {
	f := ApplyFlagsOps(0, SetKeyLocked, SetKeyLockedValueExists,
		SetNeedConstraintCheckInPrewrite, SetReadable, SetPresumeKeyNotExists)
	if got := f.AndPersistent(); got != persistentFlags {
		t.Fatalf("AndPersistent = %#x, want %#x", got, persistentFlags)
	}
	if got := ApplyFlagsOps(0, SetReadable, SetPresumeKeyNotExists).AndPersistent(); got != 0 {
		t.Fatalf("AndPersistent kept non-persistent bits: %#x", got)
	}
}

// TestDetail08: unknown ops leave the value unchanged.
func TestDetail08(t *testing.T) {
	for _, f := range []KeyFlags{0, ApplyFlagsOps(0, SetKeyLocked, SetReadable)} {
		if got := ApplyFlagsOps(f, FlagsOp(0)); got != f {
			t.Fatalf("FlagsOp(0) changed %#x to %#x", f, got)
		}
		if got := ApplyFlagsOps(f, FlagsOp(1<<30)); got != f {
			t.Fatalf("unknown op changed %#x to %#x", f, got)
		}
	}
}

// TestDetail09: the single-bit predicates observe their raw bits.
func TestDetail09(t *testing.T) {
	cases := []struct {
		op   FlagsOp
		pred func(KeyFlags) bool
		name string
	}{
		{SetKeyLocked, func(f KeyFlags) bool { return f.HasLocked() }, "HasLocked"},
		{SetNeedLocked, func(f KeyFlags) bool { return f.HasNeedLocked() }, "HasNeedLocked"},
		{SetKeyLockedValueExists, func(f KeyFlags) bool { return f.HasLockedValueExists() }, "HasLockedValueExists"},
		{SetPresumeKeyNotExists, func(f KeyFlags) bool { return f.HasNeedCheckExists() }, "HasNeedCheckExists"},
		{SetPrewriteOnly, func(f KeyFlags) bool { return f.HasPrewriteOnly() }, "HasPrewriteOnly"},
		{SetIgnoredIn2PC, func(f KeyFlags) bool { return f.HasIgnoredIn2PC() }, "HasIgnoredIn2PC"},
		{SetReadable, func(f KeyFlags) bool { return f.HasReadable() }, "HasReadable"},
		{SetNeedConstraintCheckInPrewrite, func(f KeyFlags) bool { return f.HasNeedConstraintCheckInPrewrite() }, "HasNeedConstraintCheckInPrewrite"},
		{SetNewlyInserted, func(f KeyFlags) bool { return f.HasNewlyInserted() }, "HasNewlyInserted"},
	}
	for _, c := range cases {
		if c.pred(KeyFlags(0)) {
			t.Fatalf("%s true on zero flags", c.name)
		}
		if !c.pred(ApplyFlagsOps(0, c.op)) {
			t.Fatalf("%s false after its Set op", c.name)
		}
	}
}
