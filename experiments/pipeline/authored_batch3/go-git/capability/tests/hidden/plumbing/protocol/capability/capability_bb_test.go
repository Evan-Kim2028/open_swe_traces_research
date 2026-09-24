package capability

import (
	"errors"
	"strings"
	"testing"
)

func decodeAll(t *testing.T, raw string) *List {
	t.Helper()
	l := &List{}
	DecodeList([]byte(raw), l)
	return l
}

// TestDetail01: wire form is space-separated tokens — bare name for flags,
// name=value for valued capabilities, one token per value.
func TestDetail01(t *testing.T) {
	l := &List{}
	l.Set("a", "1", "2")
	l.Add("b")
	got := string(EncodeList(l))
	if got != "a=1 a=2 b" {
		t.Fatalf("EncodeList = %q, want %q", got, "a=1 a=2 b")
	}
}

// TestDetail02: encoded order is first-insertion order, never sorted; re-adding
// an existing name appends values without moving it.
func TestDetail02(t *testing.T) {
	l := &List{}
	l.Add("zzz", "1")
	l.Add("aaa")
	l.Add("mmm")
	l.Add("zzz", "2")
	got := string(EncodeList(l))
	if got != "zzz=1 zzz=2 aaa mmm" {
		t.Fatalf("EncodeList = %q, want insertion order %q", got, "zzz=1 zzz=2 aaa mmm")
	}
}

// TestDetail03: Set on an existing capability replaces values in place but
// keeps its position; on a new name it appends.
func TestDetail03(t *testing.T) {
	l := &List{}
	l.Add("a", "1")
	l.Add("b", "9")
	l.Set("a", "x")
	got := string(EncodeList(l))
	if got != "a=x b=9" {
		t.Fatalf("Set moved existing name: %q, want %q", got, "a=x b=9")
	}
	l.Set("c", "7")
	got = string(EncodeList(l))
	if got != "a=x b=9 c=7" {
		t.Fatalf("Set on new name did not append: %q", got)
	}
}

// TestDetail04: Add on an existing name with NO values leaves its current
// values untouched.
func TestDetail04(t *testing.T) {
	l := &List{}
	l.Add("a", "1", "2")
	l.Add("a")
	got := l.Get("a")
	if len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Fatalf("Add(a) cleared values: got %v, want [1 2]", got)
	}
}

// TestDetail05: decoding "name=" produces ONE empty-string value; a bare
// "name" produces none.
func TestDetail05(t *testing.T) {
	l := decodeAll(t, "a= b")
	av := l.Get("a")
	if len(av) != 1 || av[0] != "" {
		t.Fatalf("DecodeList a= produced %v, want one empty-string value", av)
	}
	bv := l.Get("b")
	if len(bv) != 0 {
		t.Fatalf("DecodeList bare b produced %v, want no values", bv)
	}
	if !l.Supports("b") {
		t.Fatal("bare capability not present in list")
	}
}

// TestDetail06: decode tolerates leading/trailing runs of space; empty chunks
// are skipped, never turned into empty-named entries.
func TestDetail06(t *testing.T) {
	l := decodeAll(t, "   a=1    b   ")
	if names := l.All(); len(names) != 2 {
		t.Fatalf("All() = %v, want exactly [a b]", names)
	}
	if l.Supports("") {
		t.Fatal("empty-named capability exists after decoding space runs")
	}
	if got := string(EncodeList(l)); got != "a=1 b" {
		t.Fatalf("EncodeList = %q, want %q", got, "a=1 b")
	}
}

// TestDetail07: All on an empty list reports no capabilities; encoding an
// empty or nil list produces no bytes and no error.
func TestDetail07(t *testing.T) {
	l := &List{}
	if got := l.All(); len(got) != 0 {
		t.Fatalf("All() on empty list = %v", got)
	}
	if b := EncodeList(l); len(b) != 0 {
		t.Fatalf("EncodeList(empty) = %q, want no bytes", b)
	}
	var nilList *List
	if b := EncodeList(nilList); len(b) != 0 {
		t.Fatalf("EncodeList(nil) = %q, want no bytes", b)
	}
}

// TestDetail08: Delete removes the name from lookup and order; re-adding puts
// it at the end.
func TestDetail08(t *testing.T) {
	l := &List{}
	l.Add("a", "1")
	l.Add("b", "2")
	l.Add("c", "3")
	l.Delete("b")
	if l.Supports("b") {
		t.Fatal("Delete(b) left it in the lookup map")
	}
	l.Add("b", "9")
	got := string(EncodeList(l))
	if got != "a=1 c=3 b=9" {
		t.Fatalf("re-added name did not land at end: %q", got)
	}
}

// TestDetail09: validation reports the empty-argument failure for a "bogus="
// entry — not an unknown-capability failure.
func TestDetail09(t *testing.T) {
	l := decodeAll(t, "bogus=")
	err := Validate(l)
	if err == nil {
		t.Fatal("Validate(bogus=) returned nil")
	}
	if !errors.Is(err, ErrEmptyArgument) {
		t.Fatalf("Validate(bogus=) = %v, want the empty-argument failure", err)
	}
}

// TestDetail10: a fixed subset requires an argument; argument on a flag
// capability, missing required argument, and multi-value on a single-value
// capability are three distinct failures; only symref takes several values.
func TestDetail10(t *testing.T) {
	// Missing required argument.
	l := &List{}
	l.Add(Agent)
	err := Validate(l)
	if !errors.Is(err, ErrArgumentsRequired) {
		t.Fatalf("agent with no argument: %v, want ErrArgumentsRequired", err)
	}

	// Argument on a flag capability.
	l = &List{}
	l.Add(ThinPack, "x")
	err = Validate(l)
	if !errors.Is(err, ErrArguments) {
		t.Fatalf("flag capability with argument: %v, want ErrArguments", err)
	}

	// Multiple values on a single-value capability.
	l = &List{}
	l.Add(Agent, "a", "b")
	err = Validate(l)
	if !errors.Is(err, ErrMultipleArguments) {
		t.Fatalf("agent with two values: %v, want ErrMultipleArguments", err)
	}

	// symref accepts multiple values.
	l = &List{}
	l.Add(SymRef, "HEAD:refs/heads/main", "refs/tags/v1:refs/tags/v1")
	if err := Validate(l); err != nil {
		t.Fatalf("symref with two values: %v, want nil", err)
	}

	// Another required-argument member.
	l = &List{}
	l.Add(SessionID)
	if err := Validate(l); !errors.Is(err, ErrArgumentsRequired) {
		t.Fatalf("session-id with no argument: %v, want ErrArgumentsRequired", err)
	}
}

// TestDetail11: session-id is printable-ASCII-only (no byte <= 32 or >= 127)
// and must fit in one maximum pkt-line payload.
func TestDetail11(t *testing.T) {
	for _, bad := range []string{"has space", "ctrl\x1f", "del\x7f", "caf\xc3\xa9", "tab\there"} {
		l := &List{}
		l.Add(SessionID, bad)
		if err := Validate(l); err == nil {
			t.Fatalf("session-id %q accepted, want failure", bad)
		}
	}

	l := &List{}
	l.Add(SessionID, "abc123-._~ok")
	if err := Validate(l); err != nil {
		t.Fatalf("printable session-id rejected: %v", err)
	}

	l = &List{}
	l.Add(SessionID, strings.Repeat("a", 70000))
	if err := Validate(l); err == nil {
		t.Fatal("oversized session-id accepted, want failure")
	}
}

// TestDetail12: default agent is the kept base string; a non-blank
// GO_GIT_USER_AGENT_EXTRA is appended after a separator; a whitespace-only
// override is ignored.
func TestDetail12(t *testing.T) {
	t.Setenv("GO_GIT_USER_AGENT_EXTRA", "")
	base := DefaultAgent()
	if base != userAgent {
		t.Fatalf("DefaultAgent() = %q, want the kept base %q", base, userAgent)
	}

	t.Setenv("GO_GIT_USER_AGENT_EXTRA", "custom/1.0")
	got := DefaultAgent()
	if !strings.HasPrefix(got, base) || !strings.Contains(got, "custom/1.0") {
		t.Fatalf("DefaultAgent() with extra = %q, want %q followed by the extra", got, base)
	}

	t.Setenv("GO_GIT_USER_AGENT_EXTRA", "   ")
	if got := DefaultAgent(); got != base {
		t.Fatalf("whitespace-only extra not ignored: %q", got)
	}
}
