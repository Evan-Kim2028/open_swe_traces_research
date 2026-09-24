package packp

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

var (
	urH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	urH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
	urH3, _ = plumbing.FromHex("3333333333333333333333333333333333333333")
)

func urFrames(payloads ...string) *bytes.Buffer {
	var buf bytes.Buffer
	for _, p := range payloads {
		if _, err := pktline.WriteString(&buf, p); err != nil {
			panic(err)
		}
	}
	return &buf
}

func urReadFrames(t *testing.T, b []byte) []string {
	t.Helper()
	var out []string
	s := pktline.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		out = append(out, string(s.Bytes()))
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return out
}

// TestDetail01: action classification — both zero invalid; only old zero
// create; only new zero delete; both non-zero update.
func TestDetail01(t *testing.T) {
	cases := []struct {
		old, new plumbing.Hash
		want     Action
	}{
		{plumbing.ZeroHash, plumbing.ZeroHash, Invalid},
		{plumbing.ZeroHash, urH1, Create},
		{urH1, plumbing.ZeroHash, Delete},
		{urH1, urH2, Update},
	}
	for _, c := range cases {
		if got := (&Command{Old: c.old, New: c.new}).Action(); got != c.want {
			t.Fatalf("Action(old=%v,new=%v) = %v, want %v", c.old, c.new, got, c.want)
		}
	}
}

// TestDetail02: a request with no commands is rejected by both directions
// before any wire I/O.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	err := (&UpdateRequests{}).Encode(&buf)
	if !errors.Is(err, ErrEmptyCommands) {
		t.Fatalf("Encode empty = %v, want ErrEmptyCommands", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("Encode(empty) wrote %d bytes", buf.Len())
	}
	if err := (&UpdateRequests{}).Decode(urFrames()); err == nil {
		t.Fatal("Decode of empty stream accepted")
	}
}

// TestDetail03: decode accepts shallow <id> lines before the first command;
// the id must be exactly 40 or 64 hex digits and valid hex.
func TestDetail03(t *testing.T) {
	cmd := fmt.Sprintf("%s %s %s\x00\n", urH1, urH2, "refs/heads/x")
	var buf bytes.Buffer
	if _, err := pktline.WriteString(&buf, "shallow "+urH3.String()+"\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := pktline.WriteString(&buf, cmd); err != nil {
		t.Fatal(err)
	}
	if err := pktline.WriteFlush(&buf); err != nil {
		t.Fatal(err)
	}
	r := &UpdateRequests{}
	if err := r.Decode(&buf); err != nil {
		t.Fatalf("shallow+command decode: %v", err)
	}
	if len(r.Shallows) != 1 || r.Shallows[0] != urH3 {
		t.Fatalf("Shallows = %v", r.Shallows)
	}
	if len(r.Commands) != 1 {
		t.Fatalf("Commands = %v", r.Commands)
	}

	// Bad shallow id lengths / non-hex fail the whole request.
	for _, bad := range []string{
		"shallow deadbeef\n",
		"shallow zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz\n",
	} {
		var buf bytes.Buffer
		pktline.WriteString(&buf, bad)
		pktline.WriteString(&buf, cmd)
		pktline.WriteFlush(&buf)
		if err := (&UpdateRequests{}).Decode(&buf); err == nil {
			t.Fatalf("shallow line %q accepted", bad)
		}
	}
}

// TestDetail04: shallow lines followed immediately by a flush decode as a
// valid empty request; a bare flush with no shallows is malformed.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "shallow "+urH1.String()+"\n")
	pktline.WriteFlush(&buf)
	r := &UpdateRequests{}
	if err := r.Decode(&buf); err != nil {
		t.Fatalf("shallow-only request: %v", err)
	}
	if len(r.Shallows) != 1 || len(r.Commands) != 0 {
		t.Fatalf("shallow-only request = %+v", r)
	}

	var buf2 bytes.Buffer
	pktline.WriteFlush(&buf2)
	if err := (&UpdateRequests{}).Decode(&buf2); err == nil {
		t.Fatal("bare flush accepted as a request")
	}
}

// TestDetail05: the first command line carries capabilities after a NUL;
// a first line without the NUL separator is rejected.
func TestDetail05(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\n", urH1, urH2, "refs/heads/x"))
	pktline.WriteFlush(&buf)
	if err := (&UpdateRequests{}).Decode(&buf); err == nil {
		t.Fatal("first command without NUL separator accepted")
	}

	var ok bytes.Buffer
	pktline.WriteString(&ok, fmt.Sprintf("%s %s %s\x00 multi_ack\n", urH1, urH2, "refs/heads/x"))
	pktline.WriteFlush(&ok)
	r := &UpdateRequests{}
	if err := r.Decode(&ok); err != nil {
		t.Fatalf("first command with NUL caps: %v", err)
	}
	if !r.Capabilities.Supports("multi_ack") {
		t.Fatalf("capabilities not decoded: %v", r.Capabilities.All())
	}
}

// TestDetail06: command grammar is <old> SP <new> SP <name>; the name is the
// complete remainder, including interior spaces.
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\x00\n", urH1, urH2, "refs/heads/with space"))
	pktline.WriteFlush(&buf)
	r := &UpdateRequests{}
	if err := r.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(r.Commands) != 1 || r.Commands[0].Name != "refs/heads/with space" {
		t.Fatalf("name with interior space = %v", r.Commands)
	}
}

// TestDetail07: object ids accept 40 or 64 hex; other lengths or non-hex are
// rejected.
func TestDetail07(t *testing.T) {
	h64 := strings.Repeat("cd", 32)
	var buf bytes.Buffer
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\x00\n", h64, h64, "refs/heads/x"))
	pktline.WriteFlush(&buf)
	if err := (&UpdateRequests{}).Decode(&buf); err != nil {
		t.Fatalf("64-hex command rejected: %v", err)
	}
	for _, bad := range []string{
		fmt.Sprintf("%s %s %s\x00\n", "abc", urH2, "refs/heads/x"),
		fmt.Sprintf("%s %s %s\x00\n", urH1, "zz", "refs/heads/x"),
	} {
		var b bytes.Buffer
		pktline.WriteString(&b, bad)
		pktline.WriteFlush(&b)
		if err := (&UpdateRequests{}).Decode(&b); err == nil {
			t.Fatalf("bad id line %q accepted", bad)
		}
	}
}

// TestDetail08: decode requires a terminating flush; EOF before it fails.
// (Whether trailing payload after the flush is rejected is the non-inferable
// half of this detail — not pinned.)
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\x00\n", urH1, urH2, "refs/heads/x"))
	if err := (&UpdateRequests{}).Decode(&buf); err == nil {
		t.Fatal("missing flush accepted")
	}
}

// TestDetail09: later command lines strip only the trailing newline — name
// whitespace is preserved.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\x00\n", urH1, urH2, "refs/heads/a"))
	pktline.WriteString(&buf, fmt.Sprintf("%s %s %s\n", urH2, urH3, "refs/heads/  padded  name"))
	pktline.WriteFlush(&buf)
	r := &UpdateRequests{}
	if err := r.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(r.Commands) != 2 {
		t.Fatalf("Commands = %v", r.Commands)
	}
	if r.Commands[1].Name != "refs/heads/  padded  name" {
		t.Fatalf("second command name = %q, whitespace not preserved", r.Commands[1].Name)
	}
}

// TestDetail10: encode order — shallows, first command with NUL+caps, rest
// bare, one flush.
func TestDetail10(t *testing.T) {
	req := &UpdateRequests{
		Shallows: []plumbing.Hash{urH3},
		Commands: []*Command{
			{Name: "refs/heads/a", Old: urH1, New: urH2},
			{Name: "refs/heads/b", Old: urH2, New: urH3},
		},
	}
	var buf bytes.Buffer
	if err := req.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := urReadFrames(t, buf.Bytes())
	if len(frames) != 4 {
		t.Fatalf("frames = %q, want shallow+2 commands+flush", frames)
	}
	if !strings.HasPrefix(frames[0], "shallow "+urH3.String()) {
		t.Fatalf("first frame = %q, want shallow", frames[0])
	}
	if !strings.Contains(frames[1], urH1.String()+" "+urH2.String()+" refs/heads/a") {
		t.Fatalf("first command = %q", frames[1])
	}
	if !strings.Contains(frames[1], "\x00") {
		t.Fatalf("first command lacks caps NUL: %q", frames[1])
	}
	if strings.TrimRight(frames[2], "\n") != urH2.String()+" "+urH3.String()+" refs/heads/b" {
		t.Fatalf("second command = %q, want bare command line", frames[2])
	}
}

// TestDetail11: with a non-empty capability list the first command carries
// the caps after the NUL.
func TestDetail11(t *testing.T) {
	req := &UpdateRequests{
		Commands: []*Command{{Name: "refs/heads/a", Old: urH1, New: urH2}},
	}
	req.Capabilities.Add("multi_ack")
	var buf bytes.Buffer
	if err := req.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := urReadFrames(t, buf.Bytes())
	if len(frames) < 2 {
		t.Fatalf("frames = %q", frames)
	}
	cmd := frames[0]
	nul := strings.IndexByte(cmd, 0)
	if nul < 0 {
		t.Fatalf("first command lacks NUL: %q", cmd)
	}
	if !strings.Contains(cmd[nul:], "multi_ack") {
		t.Fatalf("capabilities missing after NUL: %q", cmd)
	}
}

// TestDetail12: validation runs before any bytes are emitted.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	err := (&UpdateRequests{Commands: []*Command{
		{Name: "refs/heads/x", Old: plumbing.ZeroHash, New: plumbing.ZeroHash},
	}}).Encode(&buf)
	if err == nil {
		t.Fatal("both-zero command encoded")
	}
	if buf.Len() != 0 {
		t.Fatalf("invalid request produced %d bytes", buf.Len())
	}
}
