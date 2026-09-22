package resourcecontrol

import (
	"testing"
	"time"

	"example.internal/kvstore/v2/wirerpc"
	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
)

// Hidden suite for unit ruinfo. One TestDetailNN per DETAILS.md line.

// TestDetail01: bypass is marked for the internal-others source; a non-write
// request is not a write; a zero-byte write still reports IsWrite.
func TestDetail01(t *testing.T) {
	read := tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{})
	ri := MakeRequestInfo(read)
	if ri.IsWrite() {
		t.Fatalf("Get request reported as write")
	}
	byp := tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{})
	byp.Context.RequestSource = "internal_others"
	if !MakeRequestInfo(byp).Bypass() {
		t.Fatalf("internal_others source not bypassed")
	}
	ext := tikvrpc.NewRequest(tikvrpc.CmdGet, &kvrpcpb.GetRequest{})
	ext.Context.RequestSource = "external_test"
	if MakeRequestInfo(ext).Bypass() {
		t.Fatalf("external source bypassed")
	}
	empty := tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{})
	ei := MakeRequestInfo(empty)
	if !ei.IsWrite() || ei.WriteBytes() != 0 {
		t.Fatalf("zero-byte prewrite: IsWrite=%v WriteBytes=%v", ei.IsWrite(), ei.WriteBytes())
	}
}

// TestDetail02: write bytes grow with mutation/key content for Prewrite and
// Commit; other write-typed requests count nothing.
func TestDetail02(t *testing.T) {
	small := tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{
		Mutations: []*kvrpcpb.Mutation{{Key: []byte("k"), Value: []byte("v")}},
	})
	big := tikvrpc.NewRequest(tikvrpc.CmdPrewrite, &kvrpcpb.PrewriteRequest{
		Mutations:  []*kvrpcpb.Mutation{{Key: []byte("kkkk"), Value: []byte("vvvv")}},
		PrimaryLock: []byte("p"),
	})
	sw, bw := MakeRequestInfo(small), MakeRequestInfo(big)
	if !sw.IsWrite() || !bw.IsWrite() {
		t.Fatalf("prewrite not classified as write")
	}
	if !(sw.WriteBytes() > 0 && bw.WriteBytes() > sw.WriteBytes()) {
		t.Fatalf("prewrite write bytes not monotone in content: small=%d big=%d", sw.WriteBytes(), bw.WriteBytes())
	}
	c1 := MakeRequestInfo(tikvrpc.NewRequest(tikvrpc.CmdCommit, &kvrpcpb.CommitRequest{Keys: [][]byte{[]byte("aa")}}))
	c2 := MakeRequestInfo(tikvrpc.NewRequest(tikvrpc.CmdCommit, &kvrpcpb.CommitRequest{Keys: [][]byte{[]byte("aa"), []byte("bbb")}}))
	if !(c1.WriteBytes() >= 2 && c2.WriteBytes() > c1.WriteBytes()) {
		t.Fatalf("commit write bytes do not track keys: c1=%d c2=%d", c1.WriteBytes(), c2.WriteBytes())
	}
	// Any other write-typed request contributes nothing.
	other := MakeRequestInfo(tikvrpc.NewRequest(tikvrpc.CmdPessimisticRollback,
		&kvrpcpb.PessimisticRollbackRequest{Keys: [][]byte{[]byte("aa")}}))
	if other.IsWrite() && other.WriteBytes() != 0 {
		t.Fatalf("unlisted write-typed request counted %d bytes", other.WriteBytes())
	}
}

// TestDetail03: coprocessor read bytes track the response payload; a present
// ProcessedVersionsSize overrides the payload-derived count.
func TestDetail03(t *testing.T) {
	cop := tikvrpc.Response{Resp: &coprocessor.Response{Data: make([]byte, 100)}}
	if got := MakeResponseInfo(&cop).ReadBytes(); got < 100 {
		t.Fatalf("cop response readBytes = %d, payload is 100", got)
	}
	scan := tikvrpc.Response{Resp: &kvrpcpb.ScanResponse{Pairs: []*kvrpcpb.KvPair{
		{Key: []byte("key"), Value: make([]byte, 50)},
	}}}
	if got := MakeResponseInfo(&scan).ReadBytes(); got < 53 {
		t.Fatalf("scan response readBytes = %d, want >= payload", got)
	}
	over := tikvrpc.Response{Resp: &coprocessor.Response{
		Data:          make([]byte, 100),
		ExecDetailsV2: &kvrpcpb.ExecDetailsV2{ScanDetailV2: &kvrpcpb.ScanDetailV2{ProcessedVersionsSize: 700}},
	}}
	if got := MakeResponseInfo(&over).ReadBytes(); got != 700 {
		t.Fatalf("ProcessedVersionsSize did not override readBytes: %d", got)
	}
}

// TestDetail04: KV CPU prefers TimeDetailV2 ns, then the V1-era ms field,
// then the legacy details time detail.
func TestDetail04(t *testing.T) {
	v2 := tikvrpc.Response{Resp: &kvrpcpb.GetResponse{ExecDetailsV2: &kvrpcpb.ExecDetailsV2{
		TimeDetailV2: &kvrpcpb.TimeDetailV2{ProcessWallTimeNs: 500},
		TimeDetail:   &kvrpcpb.TimeDetail{ProcessWallTimeMs: 9},
	}}}
	if got := MakeResponseInfo(&v2).KVCPU(); got != 500*time.Nanosecond {
		t.Fatalf("TimeDetailV2 not preferred: %v", got)
	}
	v1 := tikvrpc.Response{Resp: &kvrpcpb.GetResponse{ExecDetailsV2: &kvrpcpb.ExecDetailsV2{
		TimeDetail: &kvrpcpb.TimeDetail{ProcessWallTimeMs: 9},
	}}}
	if got := MakeResponseInfo(&v1).KVCPU(); got != 9*time.Millisecond {
		t.Fatalf("V1-era ms field not used: %v", got)
	}
	legacy := tikvrpc.Response{Resp: &coprocessor.Response{
		ExecDetails: &kvrpcpb.ExecDetails{TimeDetail: &kvrpcpb.TimeDetail{ProcessWallTimeMs: 7}},
	}}
	if got := MakeResponseInfo(&legacy).KVCPU(); got != 7*time.Millisecond {
		t.Fatalf("legacy time detail not used: %v", got)
	}
}

// TestDetail05: nil and unlisted responses yield an empty ResponseInfo;
// Succeed is unconditionally true.
func TestDetail05(t *testing.T) {
	for name, resp := range map[string]*tikvrpc.Response{
		"nil":      {Resp: nil},
		"unlisted": {Resp: &kvrpcpb.PrewriteResponse{}},
	} {
		ri := MakeResponseInfo(resp)
		if ri == nil {
			t.Fatalf("%s: nil ResponseInfo", name)
		}
		if ri.ReadBytes() != 0 || ri.KVCPU() != 0 {
			t.Fatalf("%s: non-empty ResponseInfo: read=%d cpu=%v", name, ri.ReadBytes(), ri.KVCPU())
		}
		if !ri.Succeed() {
			t.Fatalf("%s: Succeed false", name)
		}
	}
}
