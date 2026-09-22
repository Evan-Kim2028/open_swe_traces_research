package packp

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
)

var (
	h1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	h2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
	h64   = strings.Repeat("ab", 32) // 64 hex chars
)

func mustHash(t *testing.T, s string) plumbing.Hash {
	t.Helper()
	h, ok := plumbing.FromHex(s)
	if !ok {
		t.Fatalf("bad test hash %q", s)
	}
	return h
}

// failReader returns a sentinel error mid-stream.
type failReader struct {
	data []byte
	err  error
	pos  int
}

func (r *failReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, r.err
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestDetail01: every non-empty line must be <hex>\t<name> — a line with no
// tab fails the whole body.
func TestDetail01(t *testing.T) {
	i := &InfoRefs{}
	err := i.Decode(strings.NewReader("no-tab-here-at-all\n" + h1.String() + "\trefs/heads/ok\n"))
	if !errors.Is(err, ErrInvalidInfoRefs) {
		t.Fatalf("Decode = %v, want ErrInvalidInfoRefs", err)
	}
}

// TestDetail02: the hash field must be exactly 40 or 64 hex digits and valid
// hex; any other length or character fails the whole body.
func TestDetail02(t *testing.T) {
	for _, bad := range []string{
		"123456789012345678901234567890123456789",   // 39
		"12345678901234567890123456789012345678901", // 41
		strings.Repeat("zz", 20),                    // 40 non-hex
		strings.Repeat("ab", 31),                    // 62
	} {
		i := &InfoRefs{}
		err := i.Decode(strings.NewReader(h1.String() + "\trefs/heads/good\n" + bad + "\trefs/heads/bad\n"))
		if !errors.Is(err, ErrInvalidInfoRefs) {
			t.Fatalf("hash %q: Decode = %v, want ErrInvalidInfoRefs", bad, err)
		}
	}
	// 64 hex is legal.
	i := &InfoRefs{}
	if err := i.Decode(strings.NewReader(h64 + "\trefs/heads/s256\n")); err != nil {
		t.Fatalf("64-hex hash rejected: %v", err)
	}
	if len(i.References) != 1 || i.References[0].Hash().String() != h64 {
		t.Fatalf("sha256 ref = %v", i.References)
	}
}

// TestDetail03: a rejected body leaves the receiver's list unchanged.
func TestDetail03(t *testing.T) {
	existing := plumbing.NewHashReference("refs/heads/keep", h2)
	i := &InfoRefs{References: []*plumbing.Reference{existing}}
	err := i.Decode(strings.NewReader(h1.String() + "\trefs/heads/a\n" + "BADLINE\trefs/heads/b\n"))
	if !errors.Is(err, ErrInvalidInfoRefs) {
		t.Fatalf("Decode = %v, want ErrInvalidInfoRefs", err)
	}
	if len(i.References) != 1 || i.References[0] != existing {
		t.Fatalf("receiver mutated on rejected body: %v", i.References)
	}
}

// TestDetail04: empty lines are skipped silently.
func TestDetail04(t *testing.T) {
	i := &InfoRefs{}
	err := i.Decode(strings.NewReader("\n" + h1.String() + "\trefs/heads/a\n\n\n" + h2.String() + "\trefs/heads/b\n"))
	if err != nil {
		t.Fatalf("Decode with empty lines: %v", err)
	}
	if len(i.References) != 2 {
		t.Fatalf("got %d refs, want 2", len(i.References))
	}
}

// TestDetail05: a name that fails validation after removing one ^{} suffix is
// skipped — the line is dropped but the body still decodes.
func TestDetail05(t *testing.T) {
	body := h1.String() + "\trefs/heads/good\n" +
		h1.String() + "\tbad..name\n" + // invalid per refname rules
		h2.String() + "\trefs/heads/after\n"
	i := &InfoRefs{}
	if err := i.Decode(strings.NewReader(body)); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(i.References) != 2 ||
		i.References[0].Name() != "refs/heads/good" ||
		i.References[1].Name() != "refs/heads/after" {
		t.Fatalf("bad-name line was not skipped: %v", i.References)
	}
}

// TestDetail06: the ^{} suffix is preserved in the decoded name.
func TestDetail06(t *testing.T) {
	i := &InfoRefs{}
	if err := i.Decode(strings.NewReader(h1.String() + "\trefs/tags/v1^{}\n")); err != nil {
		t.Fatal(err)
	}
	if len(i.References) != 1 || i.References[0].Name() != "refs/tags/v1^{}" {
		t.Fatalf("peeled name = %v, want refs/tags/v1^{}", i.References)
	}
}

// TestDetail07: a line carrying a valid hash and tab but no name is skipped.
func TestDetail07(t *testing.T) {
	i := &InfoRefs{}
	err := i.Decode(strings.NewReader(h1.String() + "\t\n" + h2.String() + "\trefs/heads/b\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(i.References) != 1 || i.References[0].Name() != "refs/heads/b" {
		t.Fatalf("empty-name line not skipped: %v", i.References)
	}
}

// TestDetail08: a line over the scanner's token limit is a malformed-body
// failure; a genuine read failure passes through unchanged.
func TestDetail08(t *testing.T) {
	i := &InfoRefs{}
	long := h1.String() + "\trefs/heads/" + strings.Repeat("x", 70000) + "\n"
	if err := i.Decode(strings.NewReader(long)); err == nil {
		t.Fatal("over-token-limit line decoded without error")
	}
	if len(i.References) != 0 {
		t.Fatalf("receiver mutated on malformed body: %d refs", len(i.References))
	}

	sentinel := errors.New("underlying reader died")
	i = &InfoRefs{}
	err := i.Decode(&failReader{data: []byte(h1.String() + "\trefs/heads/a\n"), err: sentinel})
	if !errors.Is(err, sentinel) {
		t.Fatalf("read failure = %v, want passthrough of %v", err, sentinel)
	}
}

// TestDetail09: one trailing carriage return is tolerated; a name holding a
// second CR fails the name rule and the line is skipped.
func TestDetail09(t *testing.T) {
	i := &InfoRefs{}
	err := i.Decode(strings.NewReader(h1.String() + "\trefs/heads/crlf\r\n"))
	if err != nil {
		t.Fatalf("CRLF line rejected: %v", err)
	}
	if len(i.References) != 1 || i.References[0].Name() != "refs/heads/crlf" {
		t.Fatalf("CRLF name = %v", i.References)
	}
	i = &InfoRefs{}
	err = i.Decode(strings.NewReader(h1.String() + "\trefs/heads/double\r\r\n" + h2.String() + "\trefs/heads/ok\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(i.References) != 1 || i.References[0].Name() != "refs/heads/ok" {
		t.Fatalf("double-CR name not skipped: %v", i.References)
	}
}

// TestDetail10: Encode writes <hash>\t<name>\n per reference in stored order —
// no header, no terminator.
func TestDetail10(t *testing.T) {
	i := &InfoRefs{References: []*plumbing.Reference{
		plumbing.NewHashReference("refs/heads/a", h1),
		plumbing.NewHashReference("refs/tags/v1^{}", h2),
	}}
	var buf bytes.Buffer
	if err := i.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	want := h1.String() + "\trefs/heads/a\n" + h2.String() + "\trefs/tags/v1^{}\n"
	if buf.String() != want {
		t.Fatalf("Encode = %q, want %q", buf.String(), want)
	}
}

// TestDetail11: the hash field is length-checked before hex parsing — a short
// hex string must fail rather than silently pad.
func TestDetail11(t *testing.T) {
	i := &InfoRefs{}
	err := i.Decode(strings.NewReader("deadbeef\trefs/heads/short\n"))
	if !errors.Is(err, ErrInvalidInfoRefs) {
		t.Fatalf("8-hex hash: Decode = %v, want ErrInvalidInfoRefs (no silent pad)", err)
	}
	if len(i.References) != 0 {
		t.Fatal("padded hash produced a reference")
	}
}
