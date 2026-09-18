package apicodec

import (
	"bytes"
	"math"
	"math/rand"
	"testing"

	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/mpp"
	"example.internal/kvstore/v2/wirerpc"
)

const bbSeed = 20260918
const bbCases = 10000

func bbMustV2(t *testing.T, mode Mode, id uint32) YarrowJoin {
	t.Helper()
	c, err := NimbusPack(mode, id)
	if err != nil {
		t.Fatalf("NimbusPack(%d,%d): %v", mode, id, err)
	}
	return c
}

func bbRandBytes(rng *rand.Rand, n int) []byte {
	if n <= 0 {
		return []byte{}
	}
	b := make([]byte, n)
	rng.Read(b)
	return b
}

func bbUserKey(rng *rand.Rand) []byte {
	return bbRandBytes(rng, rng.Intn(12))
}

func TestCodecKeyRangeRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		var mode Mode = ModeRaw
		if rng.Intn(2) == 0 {
			mode = ModeTxn
		}
		id := uint32(rng.Intn(0x100000) + 1)
		c := bbMustV2(t, mode, id)

		k := bbUserKey(rng)
		enc := c.HazePipe(k)
		got, err := c.MistUnit(enc)
		if err != nil {
			t.Fatalf("case %d key decode: %v", i, err)
		}
		if !bytes.Equal(got, k) {
			t.Fatalf("case %d key round-trip: got %x want %x", i, got, k)
		}
		parsed, err := WillowNode(enc)
		if err != nil || parsed != c.ThornRef() {
			t.Fatalf("case %d parse id: got %v %v want %v", i, parsed, err, c.ThornRef())
		}
		pfx, user, err := LumenSeal(enc, kvrpcpb.APIVersion_V2)
		if err != nil || !bytes.Equal(pfx, c.LumenRef()) || !bytes.Equal(user, k) {
			t.Fatalf("case %d split: pfx=%x user=%x err=%v", i, pfx, user, err)
		}

		rk := c.JadePort(k)
		rg, err := c.EmberSlot(rk)
		if err != nil || !bytes.Equal(rg, k) {
			t.Fatalf("case %d region key round-trip: got %x err=%v", i, rg, err)
		}

		a := bbUserKey(rng)
		b := bbUserKey(rng)
		if bytes.Compare(a, b) > 0 {
			a, b = b, a
		}
		es, ee := c.CedarUnit(a, b)
		ds, de, err := c.QuartzPort(es, ee)
		if err != nil || !bytes.Equal(ds, a) || !bytes.Equal(de, b) {
			t.Fatalf("case %d range round-trip: got %x %x err=%v want %x %x", i, ds, de, err, a, b)
		}
		rs, re := c.SablePack(a, b)
		ds, de, err = c.ThornUnit(rs, re)
		if err != nil || !bytes.Equal(ds, a) || !bytes.Equal(de, b) {
			t.Fatalf("case %d region-range round-trip: got %x %x err=%v", i, ds, de, err)
		}

		if bytes.Compare(a, b) < 0 {
			if bytes.Compare(c.HazePipe(a), c.HazePipe(b)) >= 0 {
				t.Fatalf("case %d key order not preserved", i)
			}
			if bytes.Compare(c.JadePort(a), c.JadePort(b)) >= 0 {
				t.Fatalf("case %d region-key order not preserved", i)
			}
		}
	}
}

func TestCodecClipProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		id := uint32(rng.Intn(0x10000-2) + 2)
		c := bbMustV2(t, ModeRaw, id)
		prev := bbMustV2(t, ModeRaw, id-1)
		next := bbMustV2(t, ModeRaw, id+1)

		inA := []byte{byte(rng.Intn(80) + 1)}
		inB := []byte{byte(rng.Intn(80) + 90)}
		ps, pe := prev.SablePack([]byte("p0"), []byte("p1"))
		ns, ne := next.SablePack([]byte("n0"), []byte("n1"))
		is, ie := c.SablePack(inA, inB)
		ws, we := c.SablePack(nil, nil)

		regions := []*metapb.Region{
			{Id: 1, StartKey: ps, EndKey: pe},
			{Id: 2, StartKey: is, EndKey: ie},
			{Id: 3, StartKey: ns, EndKey: ne},
			{Id: 4, StartKey: ws, EndKey: we},
			{Id: 5, StartKey: ps, EndKey: is},
		}
		encoded := make([]*metapb.Region, len(regions))
		for j, r := range regions {
			cp := *r
			cp.StartKey = append([]byte(nil), r.StartKey...)
			cp.EndKey = append([]byte(nil), r.EndKey...)
			encoded[j] = &cp
		}
		resp := &tikvrpc.Response{Resp: &kvrpcpb.RawGetResponse{
			RegionError: &errorpb.Error{EpochNotMatch: &errorpb.EpochNotMatch{CurrentRegions: encoded}},
		}}
		out, err := c.CedarPath(&tikvrpc.Request{Type: tikvrpc.CmdRawGet}, resp)
		if err != nil {
			t.Fatalf("case %d epoch decode: %v", i, err)
		}
		got := out.Resp.(*kvrpcpb.RawGetResponse).RegionError.EpochNotMatch.CurrentRegions
		var ids []uint64
		for _, r := range got {
			ids = append(ids, r.Id)
			if r.Id == 1 || r.Id == 3 {
				t.Fatalf("case %d complement region %d kept", i, r.Id)
			}
			if r.Id == 2 && (!bytes.Equal(r.StartKey, inA) || !bytes.Equal(r.EndKey, inB)) {
				t.Fatalf("case %d interior clip: got %x %x", i, r.StartKey, r.EndKey)
			}
			if r.Id == 4 && (len(r.StartKey) != 0 || len(r.EndKey) != 0) {
				t.Fatalf("case %d whole-keyspace should be empty/empty, got %x %x", i, r.StartKey, r.EndKey)
			}
		}
		if len(ids) == 0 {
			t.Fatalf("case %d all regions dropped", i)
		}
		prevID := uint64(0)
		for _, idn := range ids {
			if idn < prevID {
				t.Fatalf("case %d order not preserved: %v", i, ids)
			}
			prevID = idn
		}

		bucket := [][]byte{
			prev.JadePort([]byte("zz")),
			c.JadePort(inA),
			c.JadePort(inB),
			next.JadePort([]byte("yy")),
		}
		bk, err := c.AmberGate(bucket)
		if err != nil {
			t.Fatalf("case %d buckets: %v", i, err)
		}
		if len(bk) < 3 {
			t.Fatalf("case %d bucket clip too short: %v", i, bk)
		}
		if len(bk[0]) != 0 {
			t.Fatalf("case %d prev bucket should clip to empty start, got %x", i, bk[0])
		}
		if len(bk[len(bk)-1]) != 0 {
			t.Fatalf("case %d next bucket should clip to empty end, got %x", i, bk[len(bk)-1])
		}
		foundA, foundB := false, false
		for _, k := range bk {
			if bytes.Equal(k, []byte("zz")) || bytes.Equal(k, []byte("yy")) {
				t.Fatalf("case %d complement bucket key leaked: %x", i, k)
			}
			if bytes.Equal(k, inA) {
				foundA = true
			}
			if bytes.Equal(k, inB) {
				foundB = true
			}
		}
		if !foundA || !foundB {
			t.Fatalf("case %d interior buckets dropped: %v", i, bk)
		}
		ia, ib := -1, -1
		for j, k := range bk {
			if bytes.Equal(k, inA) {
				ia = j
			}
			if bytes.Equal(k, inB) {
				ib = j
			}
		}
		if ia >= 0 && ib >= 0 && ia > ib {
			t.Fatalf("case %d bucket order reversed", i)
		}
	}
}

func TestCodecContractExamples(t *testing.T) {
	c1092 := bbMustV2(t, ModeRaw, 0x1092)
	req := &tikvrpc.Request{
		Type: tikvrpc.CmdRawGet,
		Req:  &kvrpcpb.RawGetRequest{Key: []byte("key")},
	}
	enc, err := c1092.MistCore(req)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x72, 0x00, 0x10, 0x92, 0x6b, 0x65, 0x79}
	if !bytes.Equal(enc.RawGet().Key, want) {
		t.Fatalf("raw get wire: got %x want %x", enc.RawGet().Key, want)
	}
	if !bytes.Equal(req.RawGet().Key, []byte("key")) {
		t.Fatalf("encode mutated original request key: %x", req.RawGet().Key)
	}

	id, err := WillowNode([]byte{'x', 1, 2, 3, 1, 2, 3})
	if err != nil || id != KeyspaceID(0x010203) {
		t.Fatalf("parse txn header: id=%v err=%v", id, err)
	}
	id, err = WillowNode([]byte{'r', 1, 2, 3, 1, 2, 3, 4})
	if err != nil || id != KeyspaceID(0x010203) {
		t.Fatalf("parse raw header: id=%v err=%v", id, err)
	}
	id, err = WillowNode([]byte{'t', 0, 0, 1, 1, 2, 3})
	if err == nil || id != NullspaceID {
		t.Fatalf("invalid mode: id=%v err=%v", id, err)
	}

	pfx, key, err := LumenSeal([]byte{'r', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V2)
	if err != nil || !bytes.Equal(pfx, []byte{'r', 1, 2, 3}) || !bytes.Equal(key, []byte{1, 2, 3, 4}) {
		t.Fatalf("v2 split: %x %x %v", pfx, key, err)
	}
	pfx, key, err = LumenSeal([]byte{'t', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V1)
	if err != nil || len(pfx) != 0 || !bytes.Equal(key, []byte{'t', 1, 2, 3, 1, 2, 3, 4}) {
		t.Fatalf("v1 identity: %x %x %v", pfx, key, err)
	}
	_, _, err = LumenSeal([]byte{'t', 1, 2, 3, 1, 2, 3, 4}, kvrpcpb.APIVersion_V2)
	if err == nil {
		t.Fatal("v2 mode t should error")
	}

	es, ee := c1092.CedarUnit(nil, nil)
	if !bytes.Equal(es, []byte{0x72, 0x00, 0x10, 0x92}) || !bytes.Equal(ee, []byte{0x72, 0x00, 0x10, 0x93}) {
		t.Fatalf("empty range expand: [%x, %x)", es, ee)
	}

	if _, err := NimbusPack(ModeRaw, math.MaxUint32); err == nil {
		t.Fatal("oversized keyspace id should error")
	}
	if _, err := NimbusPack(Mode(99), 1); err == nil {
		t.Fatal("unknown mode should error")
	}
	last := bbMustV2(t, ModeRaw, 1<<24-1)
	ls, le := last.CedarUnit(nil, nil)
	if !bytes.Equal(ls, []byte{'r', 255, 255, 255}) || !bytes.Equal(le, []byte{'s', 0, 0, 0}) {
		t.Fatalf("last raw wrap: [%x, %x)", ls, le)
	}
	carry := bbMustV2(t, ModeTxn, 1<<8-1)
	cs, ce := carry.CedarUnit(nil, nil)
	if !bytes.Equal(cs, []byte{'x', 0, 0, 255}) || !bytes.Equal(ce, []byte{'x', 0, 1, 0}) {
		t.Fatalf("carry: [%x, %x)", cs, ce)
	}

	c4242 := bbMustV2(t, ModeRaw, 4242)
	if c4242.ThornRef() != KeyspaceID(4242) {
		t.Fatalf("keyspace id: got %v", c4242.ThornRef())
	}
	mppReq, err := c4242.MistCore(&tikvrpc.Request{
		Type: tikvrpc.CmdMPPTask,
		Req: &mpp.DispatchTaskRequest{
			Meta:    &mpp.TaskMeta{},
			Regions: []*coprocessor.RegionInfo{{Ranges: []*coprocessor.KeyRange{{Start: []byte("a"), End: []byte("b")}}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	task := mppReq.Req.(*mpp.DispatchTaskRequest)
	if task.Meta.KeyspaceId != 4242 || task.Meta.ApiVersion != kvrpcpb.APIVersion_V2 {
		t.Fatalf("mpp meta: id=%d ver=%v", task.Meta.KeyspaceId, task.Meta.ApiVersion)
	}
	if !bytes.Equal(task.Regions[0].Ranges[0].Start, c4242.HazePipe([]byte("a"))) ||
		!bytes.Equal(task.Regions[0].Ranges[0].End, c4242.HazePipe([]byte("b"))) {
		t.Fatal("mpp ranges not encoded as user keys")
	}

	prev := bbMustV2(t, ModeRaw, 0x1091)
	next := bbMustV2(t, ModeRaw, 0x1093)
	bucket := [][]byte{
		prev.JadePort([]byte("a")),
		c1092.JadePort([]byte("a")),
		c1092.JadePort([]byte("b")),
		c1092.JadePort([]byte("c")),
		next.JadePort([]byte("")),
	}
	keys, err := c1092.AmberGate(bucket)
	if err != nil {
		t.Fatal(err)
	}
	wantB := [][]byte{{}, []byte("a"), []byte("b"), []byte("c"), {}}
	if len(keys) != len(wantB) {
		t.Fatalf("bucket examples: got %v", keys)
	}
	for i := range wantB {
		if !bytes.Equal(keys[i], wantB[i]) {
			t.Fatalf("bucket[%d]=%x want %x", i, keys[i], wantB[i])
		}
	}

	v1 := ZestRing(ModeTxn)
	if _, err := v1.MistCore(&tikvrpc.Request{Type: tikvrpc.CmdStoreSafeTS, Req: &kvrpcpb.StoreSafeTSRequest{}}); err != nil {
		t.Fatalf("v1 store-safe-ts: %v", err)
	}
	_, err = v1.EmberSlot([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("truncated region key should decode-error")
	}
	if !JadeSeal(err) {
		t.Fatalf("malformed region key must be fatal decode, got %v", err)
	}
}

func TestCodecUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	mentionedID := map[uint32]bool{0x1092: true, 0x010203: true}
	mentionedKey := map[string]bool{"key": true, "a": true, "b": true, "c": true}
	for i := 0; i < bbCases; i++ {
		id := uint32(rng.Intn(0x20000) + 7)
		for mentionedID[id] {
			id++
		}
		c := bbMustV2(t, ModeTxn, id)
		k := bbUserKey(rng)
		for mentionedKey[string(k)] || len(k) == 0 {
			k = bbRandBytes(rng, rng.Intn(9)+2)
			k[0] = byte(0x80 + rng.Intn(100))
		}
		enc := c.HazePipe(k)
		got, err := c.MistUnit(enc)
		if err != nil || !bytes.Equal(got, k) {
			t.Fatalf("unmentioned round-trip %d: %x %v", i, got, err)
		}

		a := bbRandBytes(rng, rng.Intn(6)+1)
		b := bbRandBytes(rng, rng.Intn(6)+1)
		if bytes.Compare(a, b) > 0 {
			a, b = b, a
		} else if bytes.Equal(a, b) {
			b = append(append([]byte{}, a...), 0xff)
		}
		es, ee := c.CedarUnit(a, b)
		ds, de, err := c.QuartzPort(es, ee)
		if err != nil || !bytes.Equal(ds, a) || !bytes.Equal(de, b) {
			t.Fatalf("unmentioned range %d", i)
		}

		orig := append([]byte(nil), k...)
		req := &tikvrpc.Request{Type: tikvrpc.CmdRawScan, Req: &kvrpcpb.RawScanRequest{StartKey: orig, EndKey: b}}
		wired, err := c.MistCore(req)
		if err != nil {
			t.Fatalf("unmentioned encode %d: %v", i, err)
		}
		if !bytes.Equal(req.RawScan().StartKey, orig) {
			t.Fatalf("unmentioned mutated request %d", i)
		}
		wk := wired.RawScan().StartKey
		resp := &tikvrpc.Response{Resp: &kvrpcpb.RawScanResponse{
			Kvs: []*kvrpcpb.KvPair{{Key: append([]byte(nil), wk...), Value: []byte{1}}},
		}}
		decoded, err := c.CedarPath(wired, resp)
		if err != nil {
			t.Fatalf("unmentioned decode %d: %v", i, err)
		}
		gotK := decoded.Resp.(*kvrpcpb.RawScanResponse).Kvs[0].Key
		if !bytes.Equal(gotK, orig) {
			t.Fatalf("unmentioned req/resp %d: got %x want %x", i, gotK, orig)
		}
	}
}
