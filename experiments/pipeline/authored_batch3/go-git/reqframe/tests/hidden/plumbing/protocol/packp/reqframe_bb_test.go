package packp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing/format/pktline"
)

// stubArgs is a CommandArgs that encodes two arg lines and decodes by
// draining until flush.
type stubArgs struct {
	decoded []string
}

func (a *stubArgs) Encode(w io.Writer) error {
	if _, err := pktline.WriteString(w, "arg-one\n"); err != nil {
		return err
	}
	_, err := pktline.WriteString(w, "arg-two\n")
	return err
}

func (a *stubArgs) Decode(r io.Reader) error {
	s := pktline.NewScanner(r)
	for s.Scan() {
		if s.Len() == 0 { // flush
			return s.Err()
		}
		a.decoded = append(a.decoded, s.Text())
	}
	return s.Err()
}

func rfFrames(t *testing.T, b []byte) []string {
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

// rfLens returns the packet lengths (0=flush, 1=delim) alongside Text frames.
func rfLens(t *testing.T, b []byte) []int {
	t.Helper()
	var out []int
	s := pktline.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		out = append(out, s.Len())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return out
}

// TestDetail01: an empty Command encodes as a single flush packet.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	if err := (&CommandRequest{}).Encode(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "0000" {
		t.Fatalf("empty request = %q, want a lone flush", buf.String())
	}
}

// TestDetail02: non-empty request = command=<name>, capability list, delim,
// args body, flush — in that order.
func TestDetail02(t *testing.T) {
	cr := &CommandRequest{Command: "ls-refs", Args: &stubArgs{}}
	cr.Capabilities.Add("agent")
	var buf bytes.Buffer
	if err := cr.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	lens := rfLens(t, buf.Bytes())
	frames := rfFrames(t, buf.Bytes())
	if len(lens) != 6 {
		t.Fatalf("packets = %q lens %v, want command+caps+delim+2args+flush", frames, lens)
	}
	if frames[0] != "command=ls-refs\n" {
		t.Fatalf("command packet = %q", frames[0])
	}
	if !strings.Contains(frames[1], "agent") {
		t.Fatalf("capability packet = %q", frames[1])
	}
	if lens[2] != 1 {
		t.Fatalf("third packet len = %d, want delim(1)", lens[2])
	}
	if frames[3] != "arg-one\n" || frames[4] != "arg-two\n" {
		t.Fatalf("args packets = %q", frames[3:5])
	}
	if lens[5] != 0 {
		t.Fatalf("last packet len = %d, want flush(0)", lens[5])
	}
}

// TestDetail03: a flush as the first packet on decode leaves Command empty
// and succeeds.
func TestDetail03(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteFlush(&buf)
	cr := &CommandRequest{}
	if err := cr.Decode(&buf); err != nil {
		t.Fatalf("empty request decode: %v", err)
	}
	if cr.Command != "" {
		t.Fatalf("Command = %q, want empty", cr.Command)
	}
}

// TestDetail04: the first non-flush line must begin `command=`; anything
// else is malformed.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "bogus=ls-refs\n")
	pktline.WriteFlush(&buf)
	if err := (&CommandRequest{}).Decode(&buf); err == nil {
		t.Fatal("non-command first line accepted")
	}
}

// TestDetail05: capabilities are read until a delim — any other packet kind
// there is a failure.
func TestDetail05(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "command=ls-refs\n")
	pktline.WriteString(&buf, "agent=x\n")
	pktline.WriteFlush(&buf) // flush where delim is required
	if err := (&CommandRequest{}).Decode(&buf); err == nil {
		t.Fatal("flush in place of delim accepted")
	}
}

// TestDetail06: with no Args decoder installed, the packet after the delim
// must be a flush — a data packet there is a failure.
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "command=ls-refs\n")
	pktline.WriteDelim(&buf)
	pktline.WriteString(&buf, "unexpected-arg\n")
	pktline.WriteFlush(&buf)
	if err := (&CommandRequest{}).Decode(&buf); err == nil {
		t.Fatal("arg body without decoder accepted")
	}
}

// TestDetail07 (shape — Inferable: no): end-of-input before the first packet
// is an empty request — Command stays empty and decode returns promptly.
func TestDetail07(t *testing.T) {
	cr := &CommandRequest{}
	done := make(chan error, 1)
	go func() { done <- cr.Decode(bytes.NewReader(nil)) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Decode on empty input did not return")
	}
	if cr.Command != "" {
		t.Fatalf("Command = %q after empty input", cr.Command)
	}
}

// TestDetail08: the git-transport request is one packet line carrying
// command, pathname, and — only when set — host and params.
func TestDetail08(t *testing.T) {
	req := &GitProtoRequest{
		RequestCommand: "git-upload-pack",
		Pathname:       "/repo.git",
		Host:           "example.com",
		ExtraParams:    []string{"p1", "p2"},
	}
	var buf bytes.Buffer
	if err := req.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := rfFrames(t, buf.Bytes())
	if len(frames) != 1 {
		t.Fatalf("git-proto request = %d frames, want 1", len(frames))
	}
	line := frames[0]
	if !strings.Contains(line, "git-upload-pack /repo.git") {
		t.Fatalf("request line = %q", line)
	}
	if !strings.Contains(line, "example.com") ||
		!strings.Contains(line, "p1") || !strings.Contains(line, "p2") {
		t.Fatalf("host/params missing: %q", line)
	}

	// Host omitted when empty.
	req.Host = ""
	buf.Reset()
	if err := req.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames = rfFrames(t, buf.Bytes())
	if len(frames) != 1 || strings.Contains(frames[0], "host=") {
		t.Fatalf("empty host still emitted: %q", frames)
	}
}

// TestDetail09: both directions reject empty command or any ASCII control
// byte in fields — encode produces no wire bytes on failure.
func TestDetail09(t *testing.T) {
	for _, bad := range []*GitProtoRequest{
		{RequestCommand: "", Pathname: "/x"},
		{RequestCommand: "git-upload-pack", Pathname: "/x\ny"},
		{RequestCommand: "git-upload-pack", Pathname: "/x", Host: "h\x1b"},
		{RequestCommand: "git-upload-pack", Pathname: "/x", ExtraParams: []string{"p\x7f"}},
	} {
		var buf bytes.Buffer
		err := bad.Encode(&buf)
		if !errors.Is(err, ErrInvalidGitProtoRequest) {
			t.Fatalf("encode %+v = %v, want ErrInvalidGitProtoRequest", bad, err)
		}
		if buf.Len() != 0 {
			t.Fatalf("invalid request wrote %d bytes", buf.Len())
		}
	}
}

// TestDetail10 (shape — Inferable: no): a request line with no terminating
// NUL (or an empty/degenerate line) exits early — no hang, and any parsed
// result is empty when the parse failed.
func TestDetail10(t *testing.T) {
	for _, payload := range []string{"git-upload-pack /repo.git\n", "", "nospaceshere"} {
		var buf bytes.Buffer
		pktline.WriteString(&buf, payload)
		pktline.WriteFlush(&buf)
		g := &GitProtoRequest{}
		done := make(chan error, 1)
		go func() { done <- g.Decode(&buf) }()
		select {
		case err := <-done:
			if err == nil && g.RequestCommand == "" && payload != "" {
				// successful decode of a non-empty line must have a command
				t.Fatalf("payload %q decoded to empty command", payload)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Decode(%q) did not return", payload)
		}
	}
}

// TestDetail11 (shape — Inferable: no): the field after the first NUL
// reaches the request — with or without a `host=` prefix, its value (minus
// the prefix) must land in Host or be preserved.
func TestDetail11(t *testing.T) {
	for _, payload := range []string{
		"git-upload-pack /repo.git\x00host=example.com\x00",
		"git-upload-pack /repo.git\x00example.com\x00",
	} {
		var buf bytes.Buffer
		pktline.WriteString(&buf, payload)
		pktline.WriteFlush(&buf)
		g := &GitProtoRequest{}
		if err := g.Decode(&buf); err != nil {
			t.Fatalf("decode %q: %v", payload, err)
		}
		got := strings.Contains(g.Host, "example.com")
		for _, p := range g.ExtraParams {
			got = got || strings.Contains(p, "example.com")
		}
		if !got {
			t.Fatalf("payload %q: host value lost (Host=%q Params=%v)",
				payload, g.Host, g.ExtraParams)
		}
	}
}

// TestDetail12: empty parameter slots (doubled terminators) are dropped on
// decode.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "git-upload-pack /repo.git\x00\x00p1\x00\x00")
	pktline.WriteFlush(&buf)
	g := &GitProtoRequest{}
	if err := g.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	for _, p := range g.ExtraParams {
		if p == "" {
			t.Fatalf("empty param slot kept: %v", g.ExtraParams)
		}
	}
	if len(g.ExtraParams) != 1 || g.ExtraParams[0] != "p1" {
		t.Fatalf("ExtraParams = %v, want [p1]", g.ExtraParams)
	}
}

// TestDetail13: decode validates parsed fields with the same control-byte
// rule as encode.
func TestDetail13(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "git-upload-pack /re\x01po.git\x00")
	pktline.WriteFlush(&buf)
	err := (&GitProtoRequest{}).Decode(&buf)
	if !errors.Is(err, ErrInvalidGitProtoRequest) {
		t.Fatalf("control-byte pathname decode = %v, want ErrInvalidGitProtoRequest", err)
	}
}
