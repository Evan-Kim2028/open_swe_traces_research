package kv

import (
	"encoding/hex"
	"strings"
	"testing"
)

// Hidden suite for unit keyops. One TestDetailNN per DETAILS.md line.

// TestDetail01: NextKey appends a single 0x00 — the immediate successor.
func TestDetail01(t *testing.T) {
	if got := NextKey([]byte("ab")); string(got) != "ab\x00" {
		t.Fatalf("NextKey(ab) = %v", got)
	}
	if got := NextKey([]byte{}); len(got) != 1 || got[0] != 0x00 {
		t.Fatalf("NextKey(empty) = %v", got)
	}
}

// TestDetail02: PrefixNextKey bumps the last byte and propagates carry left,
// never extending the slice. (Carry handling on a trailing 0xFF — drop vs
// zero-keep — is an underivable corner; only the successor shape is pinned.)
func TestDetail02(t *testing.T) {
	if got := PrefixNextKey([]byte("ab")); string(got) != "ac" {
		t.Fatalf("PrefixNextKey(ab) = %q", got)
	}
	if got := PrefixNextKey([]byte("rowkey1")); string(got) != "rowkey2" {
		t.Fatalf("PrefixNextKey(rowkey1) = %q", got)
	}
	got := PrefixNextKey([]byte("a\xff"))
	if CmpKey(got, []byte("a\xff")) <= 0 {
		t.Fatalf("PrefixNextKey(a ff) = %q, want a successor", got)
	}
	if len(got) > len("a\xff") {
		t.Fatalf("PrefixNextKey(a ff) = %q extended the slice", got)
	}
}

// TestDetail03: an all-0xFF key carries out and yields an empty result.
func TestDetail03(t *testing.T) {
	if got := PrefixNextKey([]byte{0xFF, 0xFF}); len(got) != 0 {
		t.Fatalf("PrefixNextKey(all-FF) = %v, want empty", got)
	}
	if got := PrefixNextKey([]byte{0xFF}); len(got) != 0 {
		t.Fatalf("PrefixNextKey(FF) = %v, want empty", got)
	}
}

// TestDetail04: PrefixNextKey on an empty key returns an empty result
// without panicking.
func TestDetail04(t *testing.T) {
	if got := PrefixNextKey([]byte{}); len(got) != 0 {
		t.Fatalf("PrefixNextKey(empty) = %v, want empty", got)
	}
}

// TestDetail05: CmpKey is lexicographic byte order.
func TestDetail05(t *testing.T) {
	if CmpKey([]byte("a"), []byte("b")) != -1 {
		t.Fatal("a !< b")
	}
	if CmpKey([]byte("b"), []byte("a")) != 1 {
		t.Fatal("b !> a")
	}
	if CmpKey([]byte("a"), []byte("a")) != 0 {
		t.Fatal("a != a")
	}
	if CmpKey([]byte("a"), []byte("ab")) != -1 {
		t.Fatal("a !< ab")
	}
	if CmpKey([]byte{0xFF}, []byte{0x00, 0x01}) != 1 {
		t.Fatal("byte order is not unsigned lexicographic")
	}
}

// TestDetail06: StrKey renders the key as hex (case unpinned).
func TestDetail06(t *testing.T) {
	k := []byte{0xde, 0xad, 0xbe}
	if got := StrKey(k); !strings.EqualFold(got, hex.EncodeToString(k)) {
		t.Fatalf("StrKey = %q, want the hex encoding of %x", got, k)
	}
}

// TestDetail07: IsFollowerRead is false for leader reads and true for the
// follower kind. (Which of the remaining kinds count is a choice and is not
// pinned.)
func TestDetail07(t *testing.T) {
	if ReplicaReadLeader.IsFollowerRead() {
		t.Fatalf("leader read reported as follower read")
	}
	if !ReplicaReadFollower.IsFollowerRead() {
		t.Fatalf("follower read not reported")
	}
}

// TestDetail08: String names the known kinds with their lowercase names and
// renders an unknown value distinctly.
func TestDetail08(t *testing.T) {
	names := map[ReplicaReadType]string{
		ReplicaReadLeader:   "leader",
		ReplicaReadFollower: "follower",
		ReplicaReadMixed:    "mixed",
		ReplicaReadLearner:  "learner",
	}
	for rt, want := range names {
		if got := rt.String(); got != want {
			t.Fatalf("%d.String() = %q, want %q", rt, got, want)
		}
	}
	// PreferLeader: lowercase, mentions leader. (Exact spelling unpinned.)
	if got := ReplicaReadPreferLeader.String(); got == "" || got != strings.ToLower(got) || !strings.Contains(got, "leader") {
		t.Fatalf("ReplicaReadPreferLeader.String() = %q", got)
	}
	// Unknown values get a marker distinct from every named kind.
	unk := ReplicaReadType(99).String()
	if unk == "" {
		t.Fatalf("unknown ReplicaReadType rendered as empty")
	}
	for _, want := range names {
		if unk == want {
			t.Fatalf("unknown ReplicaReadType collides with %q", want)
		}
	}
	if unk == ReplicaReadPreferLeader.String() {
		t.Fatalf("unknown ReplicaReadType collides with prefer-leader")
	}
}
