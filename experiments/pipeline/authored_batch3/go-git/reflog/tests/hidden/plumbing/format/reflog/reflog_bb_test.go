package reflog

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
)

var (
	rlH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	rlH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
)

var rlWhen = time.Unix(1700000000, 0).In(time.FixedZone("x", 5*3600+30*60))

func rlLine(old, new plumbing.Hash, msg string) string {
	s := old.String() + " " + new.String() + " A U Thor <a@b.c> 1700000000 +0530"
	if msg != "" {
		s += "\t" + msg
	}
	return s + "\n"
}

// TestDetail01: line = <old> <new> <name> <<email>> <secs> <±HHMM> with an
// optional tab+message; a line with no tab parses with empty message.
func TestDetail01(t *testing.T) {
	entries, err := Decode(strings.NewReader(
		rlLine(rlH1, rlH2, "commit: x") + rlLine(rlH2, rlH1, "")))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d", len(entries))
	}
	e := entries[0]
	if e.OldHash != rlH1 || e.NewHash != rlH2 ||
		e.Committer.Name != "A U Thor" || e.Committer.Email != "a@b.c" {
		t.Fatalf("entry = %+v", e)
	}
	if e.Message != "commit: x" {
		t.Fatalf("message = %q", e.Message)
	}
	if entries[1].Message != "" {
		t.Fatalf("no-tab line message = %q", entries[1].Message)
	}
}

// TestDetail02 (shape — Inferable: no): blank lines never produce entries;
// a final line without trailing newline is still parsed.
func TestDetail02(t *testing.T) {
	entries, err := Decode(strings.NewReader(
		"\n" + rlLine(rlH1, rlH2, "a") + "\n\n" + rlLine(rlH2, rlH1, "b") +
			strings.TrimSuffix(rlLine(rlH1, rlH2, "c"), "\n")))
	if err != nil {
		t.Fatal(err)
	}
	// Blank lines never produce entries; the unterminated final line may or
	// may not parse (its recovery is not derivable) — but no phantom entries.
	if len(entries) < 2 || len(entries) > 3 {
		t.Fatalf("entries = %d, want 2-3", len(entries))
	}
	for _, e := range entries {
		if e.Message != "a" && e.Message != "b" && e.Message != "c" {
			t.Fatalf("phantom entry from blank line: %+v", e)
		}
	}
	if len(entries) == 3 && entries[2].Message != "c" {
		t.Fatalf("unterminated final line = %+v", entries[2])
	}
}

// TestDetail03 (shape — Inferable: no): a stream with a malformed middle
// line reports an error and never returns garbage entries beyond the
// well-formed prefix.
func TestDetail03(t *testing.T) {
	entries, err := Decode(strings.NewReader(
		rlLine(rlH1, rlH2, "good") + "TOTAL-GARBAGE\n" + rlLine(rlH1, rlH2, "late")))
	if err == nil {
		t.Fatal("malformed middle line decoded without error")
	}
	for _, e := range entries {
		if e.OldHash != rlH1 && e.OldHash != rlH2 {
			t.Fatalf("garbage entry returned: %+v", e)
		}
	}
	if len(entries) > 2 {
		t.Fatalf("entries after the error exceeded the well-formed lines: %d", len(entries))
	}
}

// TestDetail04: both hashes must satisfy the hex-hash predicate; other
// fields keep their positions regardless of name contents.
func TestDetail04(t *testing.T) {
	entries, err := Decode(strings.NewReader(
		"nothex " + rlH1.String() + " N <e@x> 1 +0000\n"))
	if err == nil || len(entries) != 0 {
		t.Fatalf("non-hex old hash accepted: %v %v", entries, err)
	}
	entries, err = Decode(strings.NewReader(
		rlH1.String() + " " + rlH2.String() + " Name With Spaces <e@x> 1 +0000\tm\n"))
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Committer.Name != "Name With Spaces" {
		t.Fatalf("name with spaces = %q", entries[0].Committer.Name)
	}
}

// TestDetail05 (shape — Inferable: no): the signature field is split inside
// the line — a `>` inside the angle section does not end the email early;
// decode must not produce a bogus timestamp.
func TestDetail05(t *testing.T) {
	line := rlH1.String() + " " + rlH2.String() + " N <a>b@c> 1700000000 +0000\n"
	entries, err := Decode(strings.NewReader(line))
	if err != nil {
		return // rejection is an acceptable shape
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d", len(entries))
	}
	if !strings.Contains(entries[0].Committer.Email, "a>b@c") {
		t.Fatalf("email = %q, want the full angle contents", entries[0].Committer.Email)
	}
}

// TestDetail06 (shape — Inferable: no): an absurdly long seconds field is
// rejected — it must not decode into an entry.
func TestDetail06(t *testing.T) {
	big := strings.Repeat("9", 100)
	line := rlH1.String() + " " + rlH2.String() + " N <e@x> " + big + " +0000\n"
	entries, err := Decode(strings.NewReader(line))
	if err == nil && len(entries) != 0 {
		t.Fatal("100-char seconds field decoded")
	}
}

// TestDetail07 (shape — Inferable: no): a 5-char `±HHMM` timezone parses;
// out-of-range digits are not range-checked into an error.
func TestDetail07(t *testing.T) {
	line := rlH1.String() + " " + rlH2.String() + " N <e@x> 1700000000 +2460\n"
	entries, err := Decode(strings.NewReader(line))
	if err == nil && len(entries) == 1 {
		if _, off := entries[0].Committer.When.Zone(); off == 0 {
			t.Fatalf("+2460 parsed to zero offset")
		}
	}
	// Bad shapes are rejected.
	for _, tz := range []string{"+246", "+24600", "x2460"} {
		l := rlH1.String() + " " + rlH2.String() + " N <e@x> 1700000000 " + tz + "\n"
		if es, e := Decode(strings.NewReader(l)); e == nil && len(es) != 0 {
			t.Fatalf("tz %q accepted", tz)
		}
	}
}

// TestDetail08 (shape — Inferable: no): the encoded timezone is derived from
// the timestamp's zone — the ±HHMM text reflects the offset, and it stays a
// well-formed ±HHMM field.
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer
	e := &Entry{
		OldHash: rlH1, NewHash: rlH2,
		Committer: Signature{Name: "N", Email: "e@x", When: rlWhen},
		Message:   "m",
	}
	if err := Encode(&buf, e); err != nil {
		t.Fatal(err)
	}
	line := buf.String()
	if !strings.Contains(line, "+0530") {
		t.Fatalf("encoded line = %q, want +0530 zone text", line)
	}
}

// TestDetail09: the message is normalized — newlines/CRs become spaces,
// whitespace runs collapse, ends trimmed.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	e := &Entry{
		OldHash: rlH1, NewHash: rlH2,
		Committer: Signature{Name: "N", Email: "e@x", When: rlWhen},
		Message:   "  a\n\nb  \r\n c ",
	}
	if err := Encode(&buf, e); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(buf.String(), "\ta b c\n") {
		t.Fatalf("normalized message = %q", buf.String())
	}
}

// TestDetail10: a message that normalizes to empty is encoded with NO
// trailing tab.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	e := &Entry{
		OldHash: rlH1, NewHash: rlH2,
		Committer: Signature{Name: "N", Email: "e@x", When: rlWhen},
		Message:   " \n \n",
	}
	if err := Encode(&buf, e); err != nil {
		t.Fatal(err)
	}
	line := buf.String()
	if strings.Contains(line, "\t") {
		t.Fatalf("empty message got a tab: %q", line)
	}
	if !strings.HasSuffix(line, "+0530\n") {
		t.Fatalf("line = %q, want end right after timezone", line)
	}
}

// TestDetail11: errors quote offending fields truncated at 64 bytes — a
// giant malformed line produces a bounded error.
func TestDetail11(t *testing.T) {
	big := strings.Repeat("x", 100*1024)
	_, err := Decode(strings.NewReader(big + "\n"))
	if err == nil {
		t.Fatal("giant garbage line decoded")
	}
	if len(err.Error()) > 512 {
		t.Fatalf("error message %d bytes, want bounded", len(err.Error()))
	}
}

// TestDetail12: nil arguments are explicit errors, not panics.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, nil); err == nil {
		t.Fatal("nil entry encoded")
	}
	if err := Encode(nil, &Entry{}); err == nil {
		t.Fatal("nil writer encoded")
	}
	if _, err := Decode(nil); err == nil {
		t.Fatal("nil reader decoded")
	}
}
