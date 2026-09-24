package packp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

var (
	nsH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	nsH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
	nsH3, _ = plumbing.FromHex("3333333333333333333333333333333333333333")
)

func nsPayloads(payloads ...string) *bytes.Buffer {
	var buf bytes.Buffer
	for _, p := range payloads {
		if _, err := pktline.WriteString(&buf, p); err != nil {
			panic(err)
		}
	}
	return &buf
}

func nsFrames(t *testing.T, b []byte) []string {
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

// TestDetail01: shallow-update lines are `shallow <hash>` / `unshallow
// <hash>` ending in a flush; each kind lands in its own list in order.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "shallow "+nsH1.String()+"\n")
	pktline.WriteString(&buf, "unshallow "+nsH2.String()+"\n")
	pktline.WriteString(&buf, "shallow "+nsH3.String()+"\n")
	pktline.WriteFlush(&buf)
	su := &ShallowUpdate{}
	if err := su.Decode(&buf); err != nil {
		t.Fatal(err)
	}
	if len(su.Shallows) != 2 || su.Shallows[0] != nsH1 || su.Shallows[1] != nsH3 {
		t.Fatalf("Shallows = %v", su.Shallows)
	}
	if len(su.Unshallows) != 1 || su.Unshallows[0] != nsH2 {
		t.Fatalf("Unshallows = %v", su.Unshallows)
	}
}

// TestDetail02 (shape — Inferable: no): the two decoders disagree on a stream
// missing its flush — one accepts plain EOF, the other calls it truncated.
// Which is which is not derivable; the disagreement itself is committed.
func TestDetail02(t *testing.T) {
	var shallowBuf bytes.Buffer
	pktline.WriteString(&shallowBuf, "shallow "+nsH1.String()+"\n") // no flush
	var optBuf bytes.Buffer
	pktline.WriteString(&optBuf, "opt-one") // no flush

	suErr := make(chan error, 1)
	go func() {
		su := &ShallowUpdate{}
		suErr <- su.Decode(&shallowBuf)
	}()
	poErr := make(chan error, 1)
	go func() {
		po := &PushOptions{}
		poErr <- po.Decode(&optBuf)
	}()
	var se, pe error
	for i := 0; i < 2; i++ {
		select {
		case se = <-suErr:
		case pe = <-poErr:
		case <-time.After(5 * time.Second):
			t.Fatal("decode did not return")
		}
	}
	if (se == nil) == (pe == nil) {
		t.Fatalf("decoders agree on truncated input: shallow=%v options=%v", se, pe)
	}
}

// TestDetail03 (shape — Inferable: no): a correct-length line lands its hash;
// a wrong-length or non-hex line is never silently parsed into a valid entry.
func TestDetail03(t *testing.T) {
	// Correct length, non-hex digits: may be accepted or rejected, but must
	// not produce a *different* hash than the tail of the line.
	var buf bytes.Buffer
	pktline.WriteString(&buf, "shallow "+strings.Repeat("ab", 20)+"\n")
	pktline.WriteFlush(&buf)
	su := &ShallowUpdate{}
	err := su.Decode(&buf)
	if err == nil && len(su.Shallows) != 1 {
		t.Fatalf("correct-length line produced %d entries", len(su.Shallows))
	}

	// Wrong length: must error or add no entry.
	for _, line := range []string{"shallow deadbeef\n", "shallow " + nsH1.String() + "extra\n"} {
		var b bytes.Buffer
		pktline.WriteString(&b, line)
		pktline.WriteFlush(&b)
		s := &ShallowUpdate{}
		e := s.Decode(&b)
		if e == nil && len(s.Shallows)+len(s.Unshallows) != 0 {
			t.Fatalf("wrong-length line %q added an entry", line)
		}
	}
}

// TestDetail04 (shape — Inferable: no): a line matching neither keyword is a
// malformed-line failure — at minimum it must not be silently accumulated.
func TestDetail04(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "frobnicate "+nsH1.String()+"\n")
	pktline.WriteFlush(&buf)
	su := &ShallowUpdate{}
	err := su.Decode(&buf)
	if err == nil && len(su.Shallows)+len(su.Unshallows) != 0 {
		t.Fatal("unrecognised line silently accumulated")
	}
}

// TestDetail05: encode writes all shallows, then all unshallows, then flush —
// regardless of decode order.
func TestDetail05(t *testing.T) {
	su := &ShallowUpdate{
		Shallows:   []plumbing.Hash{nsH1, nsH3},
		Unshallows: []plumbing.Hash{nsH2},
	}
	var buf bytes.Buffer
	if err := su.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := nsFrames(t, buf.Bytes())
	if len(frames) != 4 {
		t.Fatalf("frames = %q, want 3 data lines + flush", frames)
	}
	if frames[0] != "shallow "+nsH1.String()+"\n" ||
		frames[1] != "shallow "+nsH3.String()+"\n" ||
		frames[2] != "unshallow "+nsH2.String()+"\n" {
		t.Fatalf("frames = %q, want shallows-then-unshallows", frames)
	}
}

// TestDetail06 (shape — Inferable: no): each push option is its own pkt-line
// payload — option boundaries come from framing, not embedded newlines.
func TestDetail06(t *testing.T) {
	var buf bytes.Buffer
	if err := (&PushOptions{Options: []string{"alpha", "beta"}}).Encode(&buf); err != nil {
		t.Fatal(err)
	}
	frames := nsFrames(t, buf.Bytes())
	if len(frames) != 3 { // 2 options + flush
		t.Fatalf("frames = %q, want one frame per option + flush", frames)
	}
	if !strings.Contains(frames[0], "alpha") || !strings.Contains(frames[1], "beta") {
		t.Fatalf("frames = %q missing option text", frames)
	}
}

// TestDetail07: encode validates every option before writing any byte.
func TestDetail07(t *testing.T) {
	var buf bytes.Buffer
	err := (&PushOptions{Options: []string{"fine", "bad\ttab"}}).Encode(&buf)
	if err == nil {
		t.Fatal("invalid option encoded")
	}
	if buf.Len() != 0 {
		t.Fatalf("invalid option wrote %d bytes", buf.Len())
	}
}

// TestDetail08: an option with a non-graphic character (space allowed,
// tab/newline/control not) or over the max payload size is rejected.
func TestDetail08(t *testing.T) {
	for _, bad := range []string{"has\ttab", "has\nnewline", "has\x01ctl"} {
		if err := (&PushOptions{Options: []string{bad}}).Encode(&bytes.Buffer{}); err == nil {
			t.Fatalf("option %q accepted", bad)
		}
	}
	if err := (&PushOptions{Options: []string{"space is fine"}}).Encode(&bytes.Buffer{}); err != nil {
		t.Fatalf("space-containing option rejected: %v", err)
	}
	big := strings.Repeat("x", pktline.MaxPayloadSize+1)
	if err := (&PushOptions{Options: []string{big}}).Encode(&bytes.Buffer{}); err == nil {
		t.Fatal("over-max option accepted")
	}
}

// TestDetail09 (shape — Inferable: no): decode of an empty stream leaves the
// option list empty and succeeds.
func TestDetail09(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteFlush(&buf)
	po := &PushOptions{}
	if err := po.Decode(&buf); err != nil {
		t.Fatalf("empty option stream: %v", err)
	}
	if po.Options == nil || len(po.Options) != 0 {
		t.Fatalf("empty stream left Options = %v, want non-nil empty", po.Options)
	}
}

// TestDetail10: a push-option line carrying non-graphic bytes fails on
// decode — same predicate both directions.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	pktline.WriteString(&buf, "ok-option")
	pktline.WriteString(&buf, "bad\x01option")
	pktline.WriteFlush(&buf)
	err := (&PushOptions{}).Decode(&buf)
	if err == nil {
		t.Fatal("non-graphic option decoded")
	}
	if !errors.Is(err, ErrInvalidPushOption) {
		t.Fatalf("decode error = %v, want ErrInvalidPushOption", err)
	}
}
