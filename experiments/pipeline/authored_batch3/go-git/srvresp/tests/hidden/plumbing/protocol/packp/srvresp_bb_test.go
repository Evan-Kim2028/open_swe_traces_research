package packp

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

var (
	ackH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	ackH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
)

// pktFrames encodes payload strings as pkt-lines.
func pktFrames(payloads ...string) *bytes.Buffer {
	var buf bytes.Buffer
	for _, p := range payloads {
		if _, err := pktline.WriteString(&buf, p); err != nil {
			panic(err)
		}
	}
	return &buf
}

// readFrames decodes a buffer into pkt-line payloads. Special packets
// (flush/delim/response-end) appear as empty strings.
func readFrames(t *testing.T, b []byte) []string {
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

func decodeWithin(t *testing.T, r *ServerResponse, buf *bytes.Buffer) (err error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- r.Decode(buf) }()
	select {
	case err = <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Decode did not return within 5s")
		return nil
	}
}

// TestDetail01: valid lines are ACK <hash>, ACK <hash> <status>, or NAK —
// anything else is an unexpected-content failure.
func TestDetail01(t *testing.T) {
	for _, line := range []string{"GARBAGE\n", "ERR something\n", "NOK\n"} {
		r := &ServerResponse{}
		if err := r.Decode(pktFrames(line)); err == nil {
			t.Fatalf("line %q decoded without error", line)
		}
	}
}

// TestDetail02 (shape — Inferable: no): a NAK line ends the response. Whether
// the decode reports success or an error is not derivable; what is committed
// is that NAK terminates decoding and is never itself recorded as an ACK.
func TestDetail02(t *testing.T) {
	r := &ServerResponse{}
	_ = decodeWithin(t, r, pktFrames(
		"ACK "+ackH1.String()+" continue\n",
		"NAK\n",
		"ACK "+ackH2.String()+" ready\n",
	))
	for _, a := range r.ACKs {
		if a.Hash == plumbing.ZeroHash {
			t.Fatalf("NAK line recorded as an ACK entry: %+v", r.ACKs)
		}
	}
	if len(r.ACKs) > 1 {
		t.Fatalf("lines after NAK were consumed: %+v", r.ACKs)
	}
}

// TestDetail03 (shape — Inferable: no): a bare `ACK <hash>` line is a valid
// line that appends an ACK. Whether decoding stops after it is not derivable,
// so only the append (and prompt return) is asserted.
func TestDetail03(t *testing.T) {
	r := &ServerResponse{}
	err := decodeWithin(t, r, pktFrames(
		"ACK "+ackH1.String()+" continue\n",
		"ACK "+ackH2.String()+"\n",
		"NAK\n",
	))
	if err != nil {
		t.Fatalf("well-formed ack sequence: %v", err)
	}
	if len(r.ACKs) != 2 || r.ACKs[0].Hash != ackH1 || r.ACKs[1].Hash != ackH2 {
		t.Fatalf("ACKs = %+v, want the two acks appended in order", r.ACKs)
	}
}

// TestDetail04 (shape — Inferable: no): the three named status constants map
// to their words; an unrecognised status word, when the decode succeeds,
// lands as a status that is none of the three named constants.
func TestDetail04(t *testing.T) {
	for word, want := range map[string]ACKStatus{
		"continue": ACKContinue, "common": ACKCommon, "ready": ACKReady,
	} {
		r := &ServerResponse{}
		if err := r.Decode(pktFrames("ACK "+ackH1.String()+" "+word+"\n", "NAK\n")); err != nil {
			t.Fatalf("status %q: %v", word, err)
		}
		if len(r.ACKs) != 1 || r.ACKs[0].Status != want {
			t.Fatalf("status %q decoded as %+v", word, r.ACKs)
		}
	}
	r := &ServerResponse{}
	err := r.Decode(pktFrames("ACK "+ackH1.String()+" weirdstatus\n", "NAK\n"))
	if err == nil {
		if len(r.ACKs) != 1 {
			t.Fatalf("unrecognised status ack = %+v", r.ACKs)
		}
		st := r.ACKs[0].Status
		if st == ACKContinue || st == ACKCommon || st == ACKReady {
			t.Fatalf("unrecognised word %q mapped to named status %v", "weirdstatus", st)
		}
	}
}

// TestDetail05: the zero status prints as the empty string; the three named
// constants print non-empty.
func TestDetail05(t *testing.T) {
	if s := ACKStatus(0).String(); s != "" {
		t.Fatalf("ACKStatus(0).String() = %q, want empty", s)
	}
	for _, s := range []ACKStatus{ACKContinue, ACKCommon, ACKReady} {
		if s.String() == "" {
			t.Fatalf("named status %d printed empty", s)
		}
	}
}

// TestDetail06: an ACK line shorter than the minimum for `ACK <40-hex>`, or
// one missing the hash field, is malformed.
func TestDetail06(t *testing.T) {
	for _, line := range []string{"ACK 1234\n", "ACK \n", "ACK not-hex-at-all-padding-padding-pad\n"} {
		r := &ServerResponse{}
		if err := r.Decode(pktFrames(line)); err == nil {
			t.Fatalf("malformed ACK line %q decoded without error", line)
		}
	}
}

// TestDetail07 (shape — Inferable: no): encoding an empty response emits
// exactly one pkt-line frame and it is a data line, not a flush/delim.
func TestDetail07(t *testing.T) {
	var buf bytes.Buffer
	if err := (&ServerResponse{}).Encode(&buf); err != nil {
		t.Fatal(err)
	}
	s := pktline.NewScanner(bytes.NewReader(buf.Bytes()))
	if !s.Scan() {
		t.Fatal("empty response encoded to zero packets")
	}
	if s.Len() < pktline.LenSize {
		t.Fatalf("empty response emitted a special packet (len %d), want a data line", s.Len())
	}
	if s.Scan() {
		t.Fatalf("empty response emitted more than one packet")
	}
}

// TestDetail08 (shape — Inferable: no): encoded acks appear in order, first
// frame first; each emitted frame starts with the ACK keyword and carries its
// hash. Whether all acks or only the first are emitted is not derivable.
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer
	err := (&ServerResponse{ACKs: []ACK{
		{Hash: ackH1, Status: ACKContinue},
		{Hash: ackH2, Status: ACKReady},
	}}).Encode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	frames := readFrames(t, buf.Bytes())
	if len(frames) == 0 {
		t.Fatal("no frames emitted")
	}
	if !strings.HasPrefix(frames[0], "ACK ") || !strings.Contains(frames[0], ackH1.String()) {
		t.Fatalf("first frame = %q, want ACK carrying first hash", frames[0])
	}

	buf.Reset()
	err = (&ServerResponse{ACKs: []ACK{{Hash: ackH1}, {Hash: ackH2}}}).Encode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	frames = readFrames(t, buf.Bytes())
	if len(frames) == 0 || !strings.HasPrefix(frames[0], "ACK ") ||
		!strings.Contains(frames[0], ackH1.String()) {
		t.Fatalf("statusless encode first frame = %q", frames)
	}
}

// TestDetail09: every valid ACK line appends to the list — including ones
// whose status word was unrecognised (they are not dropped).
func TestDetail09(t *testing.T) {
	r := &ServerResponse{}
	err := r.Decode(pktFrames(
		"ACK "+ackH1.String()+" continue\n",
		"ACK "+ackH2.String()+" bogus\n",
		"NAK\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.ACKs) != 2 || r.ACKs[0].Hash != ackH1 || r.ACKs[1].Hash != ackH2 {
		t.Fatalf("unrecognised-status ack was dropped: %+v", r.ACKs)
	}
}

// TestDetail10: a blank/flush line inside the response is an error, not a
// terminator.
func TestDetail10(t *testing.T) {
	var buf bytes.Buffer
	if _, err := pktline.WriteString(&buf, "ACK "+ackH1.String()+" continue\n"); err != nil {
		t.Fatal(err)
	}
	if err := pktline.WriteFlush(&buf); err != nil {
		t.Fatal(err)
	}
	if err := (&ServerResponse{}).Decode(&buf); err == nil {
		t.Fatal("flush inside response accepted as terminator")
	}
}
