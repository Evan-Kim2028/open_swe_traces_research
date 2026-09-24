package logutil

import (
	"strings"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
)

// Hidden suite for unit hexdump. One TestDetailNN per DETAILS.md line.

// TestDetail01: a proto renders as a brace-enclosed Field:value listing, one
// space between fields, no XXX_* internals, nested messages recurse.
func TestDetail01(t *testing.T) {
	out := Hex(&kvrpcpb.Context{RegionId: 7, IsolationLevel: kvrpcpb.IsolationLevel_SI}).String()
	if !strings.HasPrefix(out, "{") || !strings.HasSuffix(out, "}") {
		t.Fatalf("output not brace-enclosed: %q", out)
	}
	if !strings.Contains(out, "RegionId:7") {
		t.Fatalf("missing scalar field rendering: %q", out)
	}
	if !strings.Contains(out, "IsolationLevel:") {
		t.Fatalf("missing enum field rendering: %q", out)
	}
	if strings.Contains(out, "XXX") {
		t.Fatalf("XXX_* internals leaked: %q", out)
	}
	if strings.Contains(out, ", ") {
		t.Fatalf("fields separated by comma, want single space: %q", out)
	}
	// Nested message recurses into the same brace form.
	out = Hex(&kvrpcpb.Context{Peer: &metapb.Peer{Id: 5}}).String()
	if !strings.Contains(out, "Peer:{") || !strings.Contains(out, "Id:5") {
		t.Fatalf("nested message not in brace form: %q", out)
	}
}

// TestDetail02: []byte renders lowercase hex; hex applies only when the slice
// element kind is uint8.
func TestDetail02(t *testing.T) {
	out := Hex(&kvrpcpb.Context{ResourceGroupTag: []byte{0xde, 0xad, 0xbe, 0xef}}).String()
	if !strings.Contains(out, "deadbeef") {
		t.Fatalf("[]byte not rendered as lowercase hex: %q", out)
	}
	// [][]byte: inner bytes render as text, not hex.
	out = Hex(&kvrpcpb.PrewriteRequest{Secondaries: [][]byte{[]byte("a"), []byte("bc")}}).String()
	if !strings.Contains(out, "bc") {
		t.Fatalf("[][]byte inner content missing: %q", out)
	}
	if strings.Contains(out, "6263") {
		t.Fatalf("[][]byte inner bytes were hex-encoded: %q", out)
	}
	// []uint64: rendered but not as hex.
	out = Hex(&kvrpcpb.Context{ResolvedLocks: []uint64{1, 22}}).String()
	if !strings.Contains(out, "1") || !strings.Contains(out, "22") {
		t.Fatalf("[]uint64 field not rendered: %q", out)
	}
	if strings.Contains(out, "0116") {
		t.Fatalf("[]uint64 rendered as hex: %q", out)
	}
}

// TestDetail03: nil pointer fields and a nil message both print <nil>.
func TestDetail03(t *testing.T) {
	if out := Hex((*kvrpcpb.Context)(nil)).String(); out != "<nil>" {
		t.Fatalf("Hex(nil) = %q, want <nil>", out)
	}
	out := Hex(&kvrpcpb.Context{RegionId: 1, RegionEpoch: nil}).String()
	if !strings.Contains(out, "<nil>") {
		t.Fatalf("nil pointer field not rendered as <nil>: %q", out)
	}
}

// TestDetail04: scalar/enum fields print via %v, so enums render by name.
func TestDetail04(t *testing.T) {
	out := Hex(&kvrpcpb.Context{
		IsolationLevel: kvrpcpb.IsolationLevel_SI,
		DiskFullOpt:    kvrpcpb.DiskFullOpt_NotAllowedOnFull,
		Peer:           &metapb.Peer{Role: metapb.PeerRole_Voter},
	}).String()
	for _, name := range []string{"SI", "NotAllowedOnFull", "Voter"} {
		if !strings.Contains(out, name) {
			t.Fatalf("enum name %q missing from %q", name, out)
		}
	}
}
