package packp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol"
)

func arHash(t *testing.T, n int) plumbing.Hash {
	t.Helper()
	h, ok := plumbing.FromHex(fmt.Sprintf("%040x", n))
	if !ok {
		t.Fatalf("bad hash %d", n)
	}
	return h
}

// arFrames returns (payloads, lens) for every pkt-line in b.
// Len 0 marks a flush packet, whose payload is empty.
func arFrames(t *testing.T, b []byte) ([]string, []int) {
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

func arEncode(t *testing.T, a *AdvRefs) ([]string, []int) {
	t.Helper()
	var buf bytes.Buffer
	if err := a.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return arFrames(t, buf.Bytes())
}

var arZeroHex = strings.Repeat("0", 40)

// TestDetail01 (partially): protocol V1 emits a leading `version 1` pkt-line;
// V0 emits none; any other version is an error.
func TestDetail01(t *testing.T) {
	h1 := arHash(t, 1)

	frames, _ := arEncode(t, &AdvRefs{
		Version:    protocol.V1,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h1)},
	})
	if len(frames) == 0 || frames[0] != "version 1\n" {
		t.Fatalf("v1 first frame = %q, want %q", frames, "version 1\n")
	}

	frames, _ = arEncode(t, &AdvRefs{
		Version:    protocol.V0,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h1)},
	})
	for _, f := range frames {
		if strings.HasPrefix(f, "version") {
			t.Fatalf("v0 emitted a version frame: %q", frames)
		}
	}

	for _, v := range []protocol.Version{protocol.V2, protocol.Undefined, protocol.Version(42)} {
		var buf bytes.Buffer
		a := &AdvRefs{
			Version:    v,
			References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h1)},
		}
		if err := a.Encode(&buf); err == nil {
			t.Fatalf("version %d: Encode succeeded, want error", v)
		}
	}
}

// TestDetail02 (shape — Inferable: no): the first ref line is HEAD when a
// non-peeled HEAD exists, otherwise the first non-peeled ref in References
// order (not sorted order).
func TestDetail02(t *testing.T) {
	h1, h2, h3 := arHash(t, 1), arHash(t, 2), arHash(t, 3)

	frames, _ := arEncode(t, &AdvRefs{
		Version: protocol.V0,
		References: []*plumbing.Reference{
			plumbing.NewHashReference("refs/heads/b", h2),
			plumbing.NewHashReference(plumbing.HEAD, h1),
			plumbing.NewHashReference("refs/heads/a", h3),
		},
	})
	if len(frames) == 0 || !strings.HasPrefix(frames[0], h1.String()+" "+plumbing.HEAD.String()) {
		t.Fatalf("first frame = %q, want HEAD line with its hash", frames)
	}

	// No HEAD: the first non-peeled ref in input order wins, and input order
	// here is deliberately not sorted.
	frames, _ = arEncode(t, &AdvRefs{
		Version: protocol.V0,
		References: []*plumbing.Reference{
			plumbing.NewHashReference("refs/tags/zzz^{}", h3),
			plumbing.NewHashReference("refs/heads/z", h1),
			plumbing.NewHashReference("refs/heads/a", h2),
		},
	})
	if len(frames) == 0 || !strings.HasPrefix(frames[0], h1.String()+" refs/heads/z") {
		t.Fatalf("first frame = %q, want first non-peeled ref in input order", frames)
	}
}

// TestDetail03 (shape — Inferable: no): with no non-peeled refs the first
// line is a dummy carrying the zero hash and a synthetic name that is not
// one of the advertised references.
func TestDetail03(t *testing.T) {
	for _, refs := range [][]*plumbing.Reference{
		nil,
		{plumbing.NewHashReference("refs/tags/v1^{}", arHash(t, 1))},
	} {
		frames, _ := arEncode(t, &AdvRefs{Version: protocol.V0, References: refs})
		if len(frames) == 0 {
			t.Fatalf("no output at all for refs %v", refs)
		}
		payload := frames[0]
		rest, ok := strings.CutPrefix(payload, arZeroHex+" ")
		if !ok {
			t.Fatalf("dummy first line %q does not start with the zero hash", payload)
		}
		nul := strings.IndexByte(rest, 0)
		if nul <= 0 {
			t.Fatalf("dummy first line %q has no non-empty name before NUL", payload)
		}
		for _, r := range refs {
			if rest[:nul] == r.Name().String() {
				t.Fatalf("dummy name %q collides with an advertised ref", rest[:nul])
			}
		}
	}
}

// TestDetail04 (shape — Inferable: no): first line is
// `<hash> SP <name> NUL <caps> LF`, with the NUL present even when the
// capability list is empty.
func TestDetail04(t *testing.T) {
	h1 := arHash(t, 1)

	// Empty capabilities: the NUL is still there, immediately before LF.
	frames, _ := arEncode(t, &AdvRefs{
		Version:    protocol.V0,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h1)},
	})
	want := h1.String() + " HEAD\x00\n"
	if len(frames) == 0 || frames[0] != want {
		t.Fatalf("empty-caps first frame = %q, want %q", frames[0], want)
	}

	// Non-empty capabilities: same frame shape, caps between NUL and LF.
	a := &AdvRefs{
		Version:    protocol.V0,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h1)},
	}
	a.Capabilities.Set("symref", "HEAD:refs/heads/main")
	a.Capabilities.Add("ofs-delta")
	frames, _ = arEncode(t, a)
	if len(frames) == 0 {
		t.Fatal("no output")
	}
	p := frames[0]
	if !strings.HasPrefix(p, h1.String()+" HEAD\x00") || !strings.HasSuffix(p, "\n") {
		t.Fatalf("first frame = %q, want <hash> SP HEAD NUL <caps> LF", p)
	}
	caps := p[len(h1.String()+" HEAD\x00") : len(p)-1]
	for _, tok := range []string{"symref=HEAD:refs/heads/main", "ofs-delta"} {
		if !strings.Contains(caps, tok) {
			t.Fatalf("caps region %q missing %q", caps, tok)
		}
	}
}

// TestDetail05 (doc): remaining non-peeled refs, excluding the first-line
// one, are sorted by name and written `<hash> SP <name> LF`.
func TestDetail05(t *testing.T) {
	h0, h1, h2, h3 := arHash(t, 10), arHash(t, 11), arHash(t, 12), arHash(t, 13)

	frames, _ := arEncode(t, &AdvRefs{
		Version: protocol.V0,
		References: []*plumbing.Reference{
			plumbing.NewHashReference(plumbing.HEAD, h0),
			plumbing.NewHashReference("refs/heads/c", h1),
			plumbing.NewHashReference("refs/heads/a", h2),
			plumbing.NewHashReference("refs/heads/b", h3),
		},
	})
	want := []string{
		h0.String() + " HEAD\x00\n",
		h2.String() + " refs/heads/a\n",
		h3.String() + " refs/heads/b\n",
		h1.String() + " refs/heads/c\n",
	}
	if len(frames) < len(want) {
		t.Fatalf("frames = %q", frames)
	}
	for i, w := range want {
		if frames[i] != w {
			t.Fatalf("frame %d = %q, want %q (all frames %q)", i, frames[i], w, frames)
		}
	}
}

// TestDetail06 (doc): a peeled ref `name^{}` is written immediately after
// `name`, not in the sorted position its own name would take.
func TestDetail06(t *testing.T) {
	h0, h1, h2, h3 := arHash(t, 20), arHash(t, 21), arHash(t, 22), arHash(t, 23)

	// HEAD occupies the first line; among the rest, sorted order would be
	// v1, v1-b, v1^{} — the committed order splices the peeled line directly
	// behind its base.
	frames, _ := arEncode(t, &AdvRefs{
		Version: protocol.V0,
		References: []*plumbing.Reference{
			plumbing.NewHashReference(plumbing.HEAD, h0),
			plumbing.NewHashReference("refs/tags/v1-b", h3),
			plumbing.NewHashReference("refs/tags/v1^{}", h2),
			plumbing.NewHashReference("refs/tags/v1", h1),
		},
	})
	want := []string{
		h0.String() + " HEAD\x00\n",
		h1.String() + " refs/tags/v1\n",
		h2.String() + " refs/tags/v1^{}\n",
		h3.String() + " refs/tags/v1-b\n",
	}
	if len(frames) < len(want) {
		t.Fatalf("frames = %q", frames)
	}
	for i, w := range want {
		if frames[i] != w {
			t.Fatalf("frame %d = %q, want %q (all frames %q)", i, frames[i], w, frames)
		}
	}
}

// TestDetail07 (yes): shallows are written after refs as
// `shallow <hash>\n`, sorted by hex string.
func TestDetail07(t *testing.T) {
	h0 := arHash(t, 30)
	s1, s2, s3 := arHash(t, 39), arHash(t, 32), arHash(t, 35)

	frames, _ := arEncode(t, &AdvRefs{
		Version:    protocol.V0,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, h0)},
		Shallows:   []plumbing.Hash{s1, s2, s3},
	})
	want := []string{
		h0.String() + " HEAD\x00\n",
		"shallow " + s2.String() + "\n",
		"shallow " + s3.String() + "\n",
		"shallow " + s1.String() + "\n",
	}
	if len(frames) < len(want) {
		t.Fatalf("frames = %q", frames)
	}
	for i, w := range want {
		if frames[i] != w {
			t.Fatalf("frame %d = %q, want %q (all frames %q)", i, frames[i], w, frames)
		}
	}
}

// TestDetail08 (yes): the advertisement ends with a flush packet.
func TestDetail08(t *testing.T) {
	var buf bytes.Buffer
	a := &AdvRefs{
		Version:    protocol.V0,
		References: []*plumbing.Reference{plumbing.NewHashReference(plumbing.HEAD, arHash(t, 1))},
	}
	if err := a.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("0000")) {
		t.Fatalf("output does not end with a flush packet: %q", buf.Bytes())
	}
	frames, lens := arFrames(t, buf.Bytes())
	if len(lens) == 0 || lens[len(lens)-1] != 0 {
		t.Fatalf("last packet len = %v, want flush (0); frames %q", lens, frames)
	}
}

// TestDetail09 (doc): every data payload ends with a newline.
func TestDetail09(t *testing.T) {
	a := &AdvRefs{
		Version: protocol.V1,
		References: []*plumbing.Reference{
			plumbing.NewHashReference(plumbing.HEAD, arHash(t, 1)),
			plumbing.NewHashReference("refs/heads/x", arHash(t, 2)),
		},
		Shallows: []plumbing.Hash{arHash(t, 3)},
	}
	frames, lens := arEncode(t, a)
	for i, f := range frames {
		if lens[i] == 0 {
			continue
		}
		if !strings.HasSuffix(f, "\n") {
			t.Fatalf("payload %q does not end with newline (frames %q)", f, frames)
		}
	}
}
