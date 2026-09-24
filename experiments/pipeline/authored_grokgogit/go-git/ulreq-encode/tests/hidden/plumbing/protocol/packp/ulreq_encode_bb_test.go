package packp

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

func urHash(t *testing.T, n int) plumbing.Hash {
	t.Helper()
	h, ok := plumbing.FromHex(fmt.Sprintf("%040x", n))
	if !ok {
		t.Fatalf("bad hash %d", n)
	}
	return h
}

// urFrames returns (payloads, lens) for every pkt-line in b.
func urFrames(t *testing.T, b []byte) ([]string, []int) {
	t.Helper()
	s := pktline.NewScanner(bytes.NewReader(b))
	var frames []string
	var lens []int
	for s.Scan() {
		frames = append(frames, s.Text())
		lens = append(lens, s.Len())
	}
	if err := s.Err(); err != nil {
		t.Fatalf("pktline scan: %v", err)
	}
	return frames, lens
}

func urEncode(t *testing.T, req *UploadRequest) ([]string, []int) {
	t.Helper()
	var buf bytes.Buffer
	if err := req.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return urFrames(t, buf.Bytes())
}

// TestDetail01 (yes): zero wants returns an error.
func TestDetail01(t *testing.T) {
	var buf bytes.Buffer
	if err := (&UploadRequest{}).Encode(&buf); err == nil {
		t.Fatal("Encode with zero wants succeeded, want error")
	}
}

// TestDetail02 (partially): wants are sorted ascending before writing.
func TestDetail02(t *testing.T) {
	w1, w2, w3 := urHash(t, 41), urHash(t, 42), urHash(t, 43)

	frames, _ := urEncode(t, &UploadRequest{Wants: []plumbing.Hash{w3, w1, w2}})
	var wants []string
	for _, f := range frames {
		if strings.HasPrefix(f, "want ") {
			wants = append(wants, f)
		}
	}
	want := []string{
		"want " + w1.String() + "\n",
		"want " + w2.String() + "\n",
		"want " + w3.String() + "\n",
	}
	if len(wants) != len(want) {
		t.Fatalf("want lines = %q", wants)
	}
	for i := range want {
		if wants[i] != want[i] {
			t.Fatalf("want[%d] = %q, want %q (all %q)", i, wants[i], want[i], wants)
		}
	}
}

// TestDetail03 (partially): consecutive duplicate wants (after sorting) are
// skipped — each wanted hash appears exactly once.
func TestDetail03(t *testing.T) {
	w1, w2 := urHash(t, 51), urHash(t, 52)

	frames, _ := urEncode(t, &UploadRequest{Wants: []plumbing.Hash{w2, w1, w2, w1, w2}})
	var wants []string
	for _, f := range frames {
		if strings.HasPrefix(f, "want ") {
			wants = append(wants, f)
		}
	}
	if len(wants) != 2 {
		t.Fatalf("want lines = %q, want exactly 2", wants)
	}
	if !strings.HasPrefix(wants[0], "want "+w1.String()) ||
		!strings.HasPrefix(wants[1], "want "+w2.String()) {
		t.Fatalf("want lines = %q, want w1 then w2", wants)
	}
}

// TestDetail04 (yes): the first want line is `want <hash>\n`, or
// `want <hash> <caps>\n` when capabilities are non-empty.
func TestDetail04(t *testing.T) {
	w1 := urHash(t, 61)

	frames, _ := urEncode(t, &UploadRequest{Wants: []plumbing.Hash{w1}})
	if len(frames) == 0 || frames[0] != "want "+w1.String()+"\n" {
		t.Fatalf("first frame = %q, want bare want line", frames)
	}

	req := &UploadRequest{Wants: []plumbing.Hash{w1}}
	req.Capabilities.Set("multi_ack_detailed")
	req.Capabilities.Set("filter", "tree:0")
	frames, _ = urEncode(t, req)
	if len(frames) == 0 {
		t.Fatal("no output")
	}
	p := frames[0]
	if !strings.HasPrefix(p, "want "+w1.String()+" ") || !strings.HasSuffix(p, "\n") {
		t.Fatalf("first frame = %q, want `want <hash> <caps>`", p)
	}
	caps := p[len("want "+w1.String()+" ") : len(p)-1]
	for _, tok := range []string{"multi_ack_detailed", "filter=tree:0"} {
		if !strings.Contains(caps, tok) {
			t.Fatalf("caps region %q missing %q", caps, tok)
		}
	}
}

// TestDetail05 (yes): later wants are `want <hash>\n` with no capabilities.
func TestDetail05(t *testing.T) {
	w1, w2, w3 := urHash(t, 71), urHash(t, 72), urHash(t, 73)

	req := &UploadRequest{Wants: []plumbing.Hash{w1, w2, w3}}
	req.Capabilities.Set("ofs-delta")
	frames, _ := urEncode(t, req)
	var wants []string
	for _, f := range frames {
		if strings.HasPrefix(f, "want ") {
			wants = append(wants, f)
		}
	}
	if len(wants) != 3 {
		t.Fatalf("want lines = %q", wants)
	}
	if wants[1] != "want "+w2.String()+"\n" || wants[2] != "want "+w3.String()+"\n" {
		t.Fatalf("later wants = %q, want bare `want <hash>` lines", wants[1:])
	}
}

// TestDetail06 (partially): shallows are sorted, consecutive duplicates
// skipped, and written as `shallow <hash>\n` after the wants.
func TestDetail06(t *testing.T) {
	s1, s2 := urHash(t, 81), urHash(t, 82)

	frames, _ := urEncode(t, &UploadRequest{
		Wants:    []plumbing.Hash{urHash(t, 80)},
		Shallows: []plumbing.Hash{s2, s1, s2, s1},
	})
	var shallowIdx []int
	var shallows []string
	for i, f := range frames {
		if strings.HasPrefix(f, "shallow ") {
			shallowIdx = append(shallowIdx, i)
			shallows = append(shallows, f)
		}
	}
	want := []string{
		"shallow " + s1.String() + "\n",
		"shallow " + s2.String() + "\n",
	}
	if len(shallows) != len(want) {
		t.Fatalf("shallow lines = %q, want %q (all %q)", shallows, want, frames)
	}
	for i := range want {
		if shallows[i] != want[i] {
			t.Fatalf("shallow[%d] = %q, want %q", i, shallows[i], want[i])
		}
	}
	// all shallows come after every want line
	lastWant := -1
	for i, f := range frames {
		if strings.HasPrefix(f, "want ") {
			lastWant = i
		}
	}
	for _, i := range shallowIdx {
		if i < lastWant {
			t.Fatalf("shallow at %d precedes last want at %d: %q", i, lastWant, frames)
		}
	}
}

// TestDetail07 (doc): Deepen > 0 together with a non-zero DeepenSince or a
// non-empty DeepenNot returns ErrDeepenMutuallyExclusive.
func TestDetail07(t *testing.T) {
	for _, d := range []DepthRequest{
		{Deepen: 3, DeepenSince: time.Unix(1700000000, 0)},
		{Deepen: 3, DeepenNot: []string{"refs/heads/old"}},
		{Deepen: 3, DeepenSince: time.Unix(1700000000, 0), DeepenNot: []string{"x"}},
	} {
		var buf bytes.Buffer
		err := (&UploadRequest{Wants: []plumbing.Hash{urHash(t, 1)}, Depth: d}).Encode(&buf)
		if !errors.Is(err, ErrDeepenMutuallyExclusive) {
			t.Fatalf("depth %+v: err = %v, want ErrDeepenMutuallyExclusive", d, err)
		}
	}

	// Deepen alone is fine.
	var buf bytes.Buffer
	err := (&UploadRequest{
		Wants: []plumbing.Hash{urHash(t, 1)},
		Depth: DepthRequest{Deepen: 3},
	}).Encode(&buf)
	if err != nil {
		t.Fatalf("deepen alone: %v", err)
	}
}

// TestDetail08 (yes): `deepen <n>` is omitted when Deepen is 0.
func TestDetail08(t *testing.T) {
	frames, _ := urEncode(t, &UploadRequest{Wants: []plumbing.Hash{urHash(t, 1)}})
	for _, f := range frames {
		if strings.HasPrefix(f, "deepen ") {
			t.Fatalf("deepen emitted with Deepen=0: %q", frames)
		}
	}

	frames, _ = urEncode(t, &UploadRequest{
		Wants: []plumbing.Hash{urHash(t, 1)},
		Depth: DepthRequest{Deepen: 7},
	})
	found := false
	for _, f := range frames {
		if f == "deepen 7\n" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no `deepen 7` line: %q", frames)
	}
}

// TestDetail09 (shape — Inferable: no): `deepen-since` carries the
// DeepenSince instant expressed as UTC unix seconds.
func TestDetail09(t *testing.T) {
	// A fixed-offset zone makes local-vs-UTC decoding distinguishable.
	loc := time.FixedZone("T", 5*3600+30*60)
	since := time.Date(2024, 6, 1, 12, 0, 0, 0, loc)

	frames, _ := urEncode(t, &UploadRequest{
		Wants: []plumbing.Hash{urHash(t, 1)},
		Depth: DepthRequest{DeepenSince: since},
	})
	want := fmt.Sprintf("deepen-since %d\n", since.UTC().Unix())
	for _, f := range frames {
		if strings.HasPrefix(f, "deepen-since ") {
			if f != want {
				t.Fatalf("deepen-since = %q, want %q (UTC unix)", f, want)
			}
			return
		}
	}
	t.Fatalf("no deepen-since line: %q", frames)
}

// TestDetail10 (yes): each DeepenNot ref is a `deepen-not <ref>\n` line.
func TestDetail10(t *testing.T) {
	frames, _ := urEncode(t, &UploadRequest{
		Wants: []plumbing.Hash{urHash(t, 1)},
		Depth: DepthRequest{DeepenNot: []string{"refs/heads/a", "refs/tags/b"}},
	})
	var nots []string
	for _, f := range frames {
		if strings.HasPrefix(f, "deepen-not ") {
			nots = append(nots, f)
		}
	}
	if len(nots) != 2 {
		t.Fatalf("deepen-not lines = %q (all %q)", nots, frames)
	}
	for _, ref := range []string{"refs/heads/a", "refs/tags/b"} {
		found := false
		for _, f := range nots {
			if f == "deepen-not "+ref+"\n" {
				found = true
			}
		}
		if !found {
			t.Fatalf("no deepen-not for %q in %q", ref, nots)
		}
	}
}

// TestDetail11 (yes): a non-empty Filter is a `filter <filter>\n` line.
func TestDetail11(t *testing.T) {
	frames, _ := urEncode(t, &UploadRequest{
		Wants:  []plumbing.Hash{urHash(t, 1)},
		Filter: "blob:none",
	})
	for _, f := range frames {
		if f == "filter blob:none\n" {
			return
		}
	}
	t.Fatalf("no `filter blob:none` line: %q", frames)
}

// TestDetail12 (yes): the message ends with a flush packet.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	if err := (&UploadRequest{Wants: []plumbing.Hash{urHash(t, 1)}}).Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("0000")) {
		t.Fatalf("output does not end with flush: %q", buf.Bytes())
	}
	_, lens := urFrames(t, buf.Bytes())
	if len(lens) == 0 || lens[len(lens)-1] != 0 {
		t.Fatalf("last packet is not a flush: %v", lens)
	}
}

// TestDetail13 (doc): every data payload ends with a newline.
func TestDetail13(t *testing.T) {
	mk := func() *UploadRequest {
		r := &UploadRequest{
			Wants:    []plumbing.Hash{urHash(t, 1), urHash(t, 2)},
			Shallows: []plumbing.Hash{urHash(t, 3)},
			Filter:   "f",
		}
		r.Capabilities.Set("thin-pack")
		return r
	}
	reqs := []*UploadRequest{
		mk(),
		func() *UploadRequest { r := mk(); r.Depth = DepthRequest{Deepen: 2}; return r }(),
		func() *UploadRequest { r := mk(); r.Depth = DepthRequest{DeepenNot: []string{"r"}}; return r }(),
	}
	for _, req := range reqs {
		frames, lens := urEncode(t, req)
		for i, f := range frames {
			if lens[i] == 0 {
				continue
			}
			if !strings.HasSuffix(f, "\n") {
				t.Fatalf("payload %q lacks trailing newline (frames %q)", f, frames)
			}
		}
	}
}
