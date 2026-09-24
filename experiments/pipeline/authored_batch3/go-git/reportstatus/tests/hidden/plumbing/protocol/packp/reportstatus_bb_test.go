package packp

import (
	"bytes"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing/format/pktline"
)

// rsStream encodes payload strings as pkt-lines.
func rsStream(payloads ...string) *bytes.Buffer {
	var buf bytes.Buffer
	for _, p := range payloads {
		if _, err := pktline.WriteString(&buf, p); err != nil {
			panic(err)
		}
	}
	return &buf
}

func rsFrames(t *testing.T, b []byte) []string {
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

// TestDetail01: the first packet must be `unpack <status>` — a flush or a ref
// line first is rejected.
func TestDetail01(t *testing.T) {
	var flushOnly bytes.Buffer
	if err := pktline.WriteFlush(&flushOnly); err != nil {
		t.Fatal(err)
	}
	if err := (&ReportStatus{}).Decode(&flushOnly); err == nil {
		t.Fatal("flush-first stream accepted")
	}
	for _, first := range []string{"ok refs/heads/a\n", "ng refs/heads/a bad\n"} {
		var buf bytes.Buffer
		pktline.WriteString(&buf, first)
		pktline.WriteFlush(&buf)
		if err := (&ReportStatus{}).Decode(&buf); err == nil {
			t.Fatalf("first line %q accepted", first)
		}
	}
}

// TestDetail02 (shape — Inferable: no): a non-ok unpack status surfaces
// through the error view. Whether Decode itself errors is not derivable;
// what is committed is that a decoded non-ok unpack status produces a
// non-nil error carrying the status text.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "unpack index-pack-failed\n")
	pktline.WriteFlush(&buf)
	rs := &ReportStatus{}
	if err := rs.Decode(&buf); err == nil {
		e := rs.Error()
		if e == nil || !strings.Contains(e.Error(), "index-pack-failed") {
			t.Fatalf("Error() = %v, want error carrying the unpack status", e)
		}
	}
}

// TestDetail03 (shape — Inferable: no): well-formed `ok <ref>` and
// `ng <ref> <reason>` lines populate CommandStatuses in order. Arity
// enforcement on malformed lines is not derivable and is not asserted.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "unpack ok\n")
	pktline.WriteString(&buf, "ok refs/heads/a\n")
	pktline.WriteString(&buf, "ng refs/heads/b update rejected\n")
	pktline.WriteFlush(&buf)
	rs := &ReportStatus{}
	if err := rs.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(rs.CommandStatuses) != 2 ||
		rs.CommandStatuses[0].ReferenceName != "refs/heads/a" ||
		rs.CommandStatuses[1].ReferenceName != "refs/heads/b" {
		t.Fatalf("CommandStatuses = %+v", rs.CommandStatuses)
	}
	if rs.CommandStatuses[1].Status == "" {
		t.Fatal("ng reason not captured in Status")
	}
}

// TestDetail04: the error view reports the unpack failure first, then only
// the first ref failure.
func TestDetail04(t *testing.T) {
	rs := &ReportStatus{
		UnpackStatus: "unpack-failure",
		CommandStatuses: []*CommandStatus{
			{ReferenceName: "refs/heads/first", Status: "bad1"},
			{ReferenceName: "refs/heads/second", Status: "bad2"},
		},
	}
	err := rs.Error()
	if err == nil || !strings.Contains(err.Error(), "unpack-failure") {
		t.Fatalf("Error() = %v, want unpack failure first", err)
	}

	rs.UnpackStatus = "ok"
	err = rs.Error()
	if err == nil {
		t.Fatal("Error() = nil with failing refs")
	}
	if !strings.Contains(err.Error(), "refs/heads/first") {
		t.Fatalf("Error() = %v, want first ref failure", err)
	}
	if strings.Contains(err.Error(), "refs/heads/second") {
		t.Fatalf("Error() = %v, later ref failures must be dropped", err)
	}
}

// TestDetail05: a ref status of exactly `ok` produces no error; any other
// status — including empty — is an error carrying ref name and status.
func TestDetail05(t *testing.T) {
	if err := (&CommandStatus{ReferenceName: "refs/heads/a", Status: "ok"}).Error(); err != nil {
		t.Fatalf("ok status produced error %v", err)
	}
	for _, st := range []string{"", "ng", "failure text"} {
		err := (&CommandStatus{ReferenceName: "refs/heads/x", Status: st}).Error()
		if err == nil {
			t.Fatalf("status %q produced no error", st)
		}
		if !strings.Contains(err.Error(), "refs/heads/x") {
			t.Fatalf("status %q error %v lacks ref name", st, err)
		}
	}
}

// TestDetail06: decode stops at the first flush and requires it — running
// out of input first is a missing-terminator error.
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "unpack ok\n")
	pktline.WriteString(&buf, "ok refs/heads/a\n")
	if err := (&ReportStatus{}).Decode(&buf); err == nil {
		t.Fatal("missing flush accepted")
	}
}

// TestDetail07: encode emits `unpack <status>`, one line per ref, then a
// flush — nothing else.
func TestDetail07(t *testing.T) {
	rs := &ReportStatus{
		UnpackStatus: "ok",
		CommandStatuses: []*CommandStatus{
			{ReferenceName: "refs/heads/a", Status: "ok"},
			{ReferenceName: "refs/heads/b", Status: "forced-update"},
		},
	}
	var buf bytes.Buffer
	if err := rs.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := rsFrames(t, buf.Bytes())
	if len(frames) != 4 { // unpack + ok + ng + flush (flush shows as "")
		t.Fatalf("frames = %q, want unpack+2 refs+flush", frames)
	}
	if frames[0] != "unpack ok\n" {
		t.Fatalf("first frame = %q", frames[0])
	}
	if frames[1] != "ok refs/heads/a\n" {
		t.Fatalf("ok frame = %q", frames[1])
	}
	if !strings.HasPrefix(frames[2], "ng refs/heads/b ") ||
		!strings.Contains(frames[2], "forced-update") {
		t.Fatalf("ng frame = %q", frames[2])
	}
}

// TestDetail08 (shape — Inferable: no): on encode a ref's line is `ok` when
// it has no error and `ng` carrying its status text otherwise.
func TestDetail08(t *testing.T) {
	rs := &ReportStatus{
		UnpackStatus: "ok",
		CommandStatuses: []*CommandStatus{
			{ReferenceName: "refs/heads/a", Status: "not-ok"},
		},
	}
	var buf bytes.Buffer
	if err := rs.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := rsFrames(t, buf.Bytes())
	if len(frames) < 2 {
		t.Fatalf("frames = %q", frames)
	}
	refLine := frames[1]
	if strings.HasPrefix(refLine, "ok ") {
		t.Fatalf("failing ref encoded as ok: %q", refLine)
	}
	if !strings.Contains(refLine, "refs/heads/a") || !strings.Contains(refLine, "not-ok") {
		t.Fatalf("failing ref line = %q, want name+status", refLine)
	}
}

// TestDetail09: the two failure kinds are distinguishable error values, each
// carrying its offending text (kept Error() methods).
func TestDetail09(t *testing.T) {
	u := UnpackStatusErr{Status: "the-unpack-status"}
	c := CommandStatusErr{ReferenceName: "refs/heads/r", Status: "the-ref-status"}
	if !strings.Contains(u.Error(), "the-unpack-status") {
		t.Fatalf("UnpackStatusErr = %q", u.Error())
	}
	if !strings.Contains(c.Error(), "the-ref-status") ||
		!strings.Contains(c.Error(), "refs/heads/r") {
		t.Fatalf("CommandStatusErr = %q", c.Error())
	}
	if u.Error() == c.Error() {
		t.Fatal("the two error kinds render identically")
	}
}

// TestDetail10: a stream starting with a ref line, or an empty stream, fails
// before any ref-status decode.
func TestDetail10(t *testing.T) {
	if err := (&ReportStatus{}).Decode(strings.NewReader("")); err == nil {
		t.Fatal("empty stream accepted")
	}
	var buf bytes.Buffer
	pktline.WriteString(&buf, "ok refs/heads/a\n")
	pktline.WriteFlush(&buf)
	if err := (&ReportStatus{}).Decode(&buf); err == nil {
		t.Fatal("ref-line-first stream accepted")
	}
}
