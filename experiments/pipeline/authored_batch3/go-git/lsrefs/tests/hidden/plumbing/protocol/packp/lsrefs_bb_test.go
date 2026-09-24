package packp

import (
	"bytes"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

var (
	lrH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	lrH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
	lrH3, _ = plumbing.FromHex("3333333333333333333333333333333333333333")
)

func lrFrames(t *testing.T, b []byte) []string {
	t.Helper()
	var out []string
	s := pktline.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		out = append(out, s.Text())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return out
}

// TestDetail01: argument encode order is peel, symrefs, unborn, then one
// `ref-prefix <p>` line per prefix; unset flags emit nothing.
func TestDetail01(t *testing.T) {
	args := &LsRefsArgs{
		Peel:        true,
		Symrefs:     true,
		Unborn:      true,
		RefPrefixes: []string{"refs/heads/", "refs/tags/"},
	}
	var buf bytes.Buffer
	if err := args.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := lrFrames(t, buf.Bytes())
	if len(frames) != 5 {
		t.Fatalf("frames = %q, want peel+symrefs+unborn+2 prefixes", frames)
	}
	if frames[0] != "peel\n" && frames[0] != "peel" {
		t.Fatalf("first arg = %q, want peel", frames[0])
	}
	if !strings.HasPrefix(frames[1], "symrefs") || !strings.HasPrefix(frames[2], "unborn") {
		t.Fatalf("flag order = %q", frames[:3])
	}
	if !strings.Contains(frames[3], "ref-prefix refs/heads/") ||
		!strings.Contains(frames[4], "ref-prefix refs/tags/") {
		t.Fatalf("prefix frames = %q", frames[3:5])
	}

	// Unset flags emit nothing.
	buf.Reset()
	if err := (&LsRefsArgs{}).Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames = lrFrames(t, buf.Bytes())
	for _, f := range frames {
		if strings.HasPrefix(f, "peel") || strings.HasPrefix(f, "symrefs") ||
			strings.HasPrefix(f, "unborn") {
			t.Fatalf("unset flag emitted: %q", f)
		}
	}
}

// TestDetail02 (shape — Inferable: no): a bad ref-prefix fails the whole
// encode; the committed part is that an encode failure produces no argument
// lines on the wire.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	err := (&LsRefsArgs{
		Peel:        true,
		RefPrefixes: []string{"bad prefix"},
	}).Encode(&buf)
	if err == nil {
		t.Fatal("bad ref-prefix encoded")
	}
	for _, f := range lrFrames(t, buf.Bytes()) {
		if strings.HasPrefix(f, "peel") || strings.HasPrefix(f, "ref-prefix") {
			t.Fatalf("argument emitted before validation failure: %q", f)
		}
	}
}

// TestDetail03: a ref-prefix is rejected when empty or carrying whitespace /
// control characters.
func TestDetail03(t *testing.T) {
	for _, bad := range []string{"", "has space", "has\ttab", "has\nnl", "nul\x00inside", "del\x7f"} {
		var buf bytes.Buffer
		if err := (&LsRefsArgs{RefPrefixes: []string{bad}}).Encode(&buf); err == nil {
			t.Fatalf("ref-prefix %q accepted", bad)
		}
	}
	var buf bytes.Buffer
	if err := (&LsRefsArgs{RefPrefixes: []string{"refs/heads/"}}).Encode(&buf); err != nil {
		t.Fatalf("clean prefix rejected: %v", err)
	}
}

// TestDetail04: argument decode ignores unrecognised and empty lines; a
// flush or end-of-input terminates.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "unrecognised-arg\n")
	pktline.WriteString(&buf, "")
	pktline.WriteString(&buf, "peel\n")
	pktline.WriteString(&buf, "ref-prefix refs/heads/\n")
	pktline.WriteFlush(&buf)
	args := &LsRefsArgs{}
	if err := args.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if !args.Peel || len(args.RefPrefixes) != 1 || args.RefPrefixes[0] != "refs/heads/" {
		t.Fatalf("decoded args = %+v", args)
	}
}

// TestDetail05 (shape — Inferable: no): at the ref-prefix bound (65536) the
// accumulated list does not behave like a capped list — decode must return
// promptly; any result is either empty or at the bound, never a capped 65536.
func TestDetail05(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "ref-prefix refs/heads/\n")
	for i := 0; i < 65536; i++ {
		if _, err := pktline.WriteString(&buf, "ref-prefix r\n"); err != nil {
			t.Fatal(err)
		}
	}
	pktline.WriteFlush(&buf)
	args := &LsRefsArgs{}
	if err := args.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(args.RefPrefixes) == 65536 {
		t.Fatal("prefix list capped at the bound instead of dropped")
	}
}

// TestDetail06: ref-line grammar — `<oid> SP <refname> [SP symref-target:<t>]
// [SP peeled:<oid>]`, or `unborn SP <refname> SP symref-target:<t>`.
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, lrH1.String()+" refs/heads/a\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	if err := out.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(out.References) != 1 || out.References[0].Name() != "refs/heads/a" ||
		out.References[0].Hash() != lrH1 {
		t.Fatalf("plain ref line = %v", out.References)
	}
}

// TestDetail07: on encode a `^{}` entry never gets its own line — it folds
// into the base ref's peeled attribute.
func TestDetail07(t *testing.T) {
	out := &LsRefsOutput{References: []*plumbing.Reference{
		plumbing.NewHashReference("refs/tags/v1", lrH1),
		plumbing.NewHashReference("refs/tags/v1^{}", lrH2),
		plumbing.NewHashReference("refs/heads/x", lrH3),
	}}
	var buf bytes.Buffer
	if err := out.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := lrFrames(t, buf.Bytes())
	if len(frames) != 2 {
		t.Fatalf("peeled ref got its own line: %q", frames)
	}
	if !strings.HasPrefix(frames[0], lrH1.String()+" refs/tags/v1") ||
		!strings.Contains(frames[0], "peeled:"+lrH2.String()) {
		t.Fatalf("base line = %q, want peeled attribute", frames[0])
	}
	if frames[1] != lrH3.String()+" refs/heads/x\n" {
		t.Fatalf("second line = %q", frames[1])
	}
}

// TestDetail08 (shape — Inferable: no): a symbolic ref encodes with a
// symref-target attribute; when the target's hash is in the same list the
// oid position carries an object id, otherwise the line is marked unborn.
func TestDetail08(t *testing.T) {
	out := &LsRefsOutput{References: []*plumbing.Reference{
		plumbing.NewSymbolicReference("HEAD", "refs/heads/a"),
		plumbing.NewHashReference("refs/heads/a", lrH1),
	}}
	var buf bytes.Buffer
	if err := out.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := lrFrames(t, buf.Bytes())
	if len(frames) != 2 {
		t.Fatalf("frames = %q", frames)
	}
	var head string
	for _, f := range frames {
		if strings.Contains(f, "HEAD") {
			head = f
		}
	}
	if head == "" || !strings.Contains(head, "symref-target:refs/heads/a") {
		t.Fatalf("symbolic HEAD line = %q", head)
	}
	// The oid position is either the resolved target hash or the unborn
	// marker — never an arbitrary value.
	if !strings.Contains(head, lrH1.String()) && !strings.Contains(head, "unborn") {
		t.Fatalf("symbolic oid position = %q", head)
	}

	// Unborn case: target hash absent.
	out = &LsRefsOutput{References: []*plumbing.Reference{
		plumbing.NewSymbolicReference("HEAD", "refs/heads/missing"),
	}}
	buf.Reset()
	if err := out.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames = lrFrames(t, buf.Bytes())
	if len(frames) != 1 || !strings.Contains(frames[0], "symref-target:refs/heads/missing") {
		t.Fatalf("unborn symref line = %q", frames)
	}
}

// TestDetail09 (shape — Inferable: no): an `unborn` line without a
// `symref-target:` attribute is rejected — decode must not silently produce
// a reference from it.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "unborn refs/heads/x\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	err := out.Decode(&buf)
	if err == nil && len(out.References) != 0 {
		t.Fatal("bare unborn line produced a reference")
	}
}

// TestDetail10: a `peeled:` attribute produces a second `<base>^{}`
// reference right after the base entry.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, lrH1.String()+" refs/tags/v1 peeled:"+lrH2.String()+"\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	if err := out.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(out.References) != 2 {
		t.Fatalf("peeled decode = %v", out.References)
	}
	if out.References[0].Name() != "refs/tags/v1" || out.References[0].Hash() != lrH1 {
		t.Fatalf("base = %v", out.References[0])
	}
	if out.References[1].Name() != "refs/tags/v1^{}" || out.References[1].Hash() != lrH2 {
		t.Fatalf("peeled = %v", out.References[1])
	}
}

// TestDetail11 (shape — Inferable: no): a line with `symref-target:` decodes
// to a symbolic reference — the committed part is that the result is a
// symbolic ref, not a hash ref.
func TestDetail11(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, lrH1.String()+" HEAD symref-target:refs/heads/main\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	if err := out.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(out.References) == 0 {
		t.Fatal("symref line produced no reference")
	}
	if out.References[0].Type() != plumbing.SymbolicReference {
		t.Fatalf("symref line decoded as %v, want symbolic", out.References[0].Type())
	}
}

// TestDetail12: object ids parse strictly — exactly 40 or 64 hex; shorter
// hex must not be padded into an id.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "deadbeef refs/heads/short\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	err := out.Decode(&buf)
	if err == nil && len(out.References) != 0 {
		t.Fatal("short hex id accepted as an object id")
	}
}

// TestDetail13: unknown attributes are ignored; fields split on runs of
// spaces.
func TestDetail13(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, lrH1.String()+"  refs/heads/a   bogus-attr:x\n")
	pktline.WriteFlush(&buf)
	out := &LsRefsOutput{}
	if err := out.Decode(&buf); err != nil {
		t.Fatalf("unknown attr / multi-space line: %v", err)
	}
	if len(out.References) != 1 || out.References[0].Name() != "refs/heads/a" {
		t.Fatalf("decoded = %v", out.References)
	}
}
