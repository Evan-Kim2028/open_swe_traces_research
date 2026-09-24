package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	formatcfg "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
	"example.internal/gitkit/v6/plumbing/protocol/packp"
	"example.internal/gitkit/v6/storage/memory"
)

// mockWriteCloser captures written bytes and counts Close calls. Its
// constructor signature and field set mirror the harness the kept in-tree
// tests (send_pack_test.go et al.) were written against — the unit
// excision removed the file that defined it.
type mockWriteCloser struct {
	writeBuf *bytes.Buffer
	writeErr error
	closeErr error
	closed   bool
	closes   int
}

func newMockWriteCloser(_ []byte) *mockWriteCloser {
	return &mockWriteCloser{writeBuf: &bytes.Buffer{}}
}

func (w *mockWriteCloser) Write(p []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.writeBuf.Write(p)
}

func (w *mockWriteCloser) Close() error {
	w.closed = true
	w.closes++
	return w.closeErr
}

func synthHaves(n int, prefix ...plumbing.Hash) []plumbing.Hash {
	haves := append([]plumbing.Hash{}, prefix...)
	for i := len(haves); i < n; i++ {
		haves = append(haves, plumbing.NewHash(fmt.Sprintf("%040x", i+0x1000000)))
	}
	return haves
}

func nakLines(n int) *bytes.Buffer {
	buf := &bytes.Buffer{}
	for i := 0; i < n; i++ {
		pktline.WriteString(buf, "NAK\n")
	}
	return buf
}

// segments splits a captured writer stream on flush packets; each element
// is the payload text of one client round.
func segments(raw string) []string {
	var out []string
	var cur strings.Builder
	s := pktline.NewScanner(strings.NewReader(raw))
	for s.Scan() {
		b := s.Bytes()
		if len(b) == 0 { // flush
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.Write(b)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// 1. Have-batch window grows: doubling while small, then slower growth
//    past the cap — 10%-style under stateless RPC vs a fixed small step
//    under a persistent pipe. Inferable: no — assert SHAPE: monotone,
//    doubling below the cap, and stateless outgrows pipe past it.
func TestDetail01(t *testing.T) {
	if nextFlush(true, initialFlush) != 2*initialFlush {
		t.Fatalf("stateless below cap: %d", nextFlush(true, initialFlush))
	}
	if nextFlush(false, initialFlush) != 2*initialFlush {
		t.Fatalf("pipe below cap: %d", nextFlush(false, initialFlush))
	}
	for _, sl := range []bool{true, false} {
		prev := 0
		for _, in := range []int{1, 7, 100, 1000, 5000, 20000, 40000} {
			got := nextFlush(sl, in)
			if got <= in {
				t.Fatalf("stateless=%v in=%d → %d not increasing", sl, in, got)
			}
			if got <= prev {
				t.Fatalf("stateless=%v in=%d → %d not monotone", sl, in, got)
			}
			prev = got
		}
	}
	// Past the cap neither mode doubles; stateless grows more than pipe.
	if got := nextFlush(false, largeFlush); got >= 2*largeFlush {
		t.Fatalf("pipe still doubling past cap: %d", got)
	}
	if got := nextFlush(true, largeFlush); got >= 2*largeFlush {
		t.Fatalf("stateless still doubling at cap boundary: %d", got)
	}
	if nextFlush(true, largeFlush) <= nextFlush(false, largeFlush) {
		t.Fatalf("stateless growth %d not greater than pipe %d at cap",
			nextFlush(true, largeFlush), nextFlush(false, largeFlush))
	}
	// Pipe mode grows by a small fixed step, not proportionally.
	if nextFlush(false, 2*largeFlush)-nextFlush(false, largeFlush) > largeFlush {
		t.Fatal("pipe growth between two cap-scale inputs is proportional, not fixed")
	}
}

// 2. After "continue", at most maxInVein haves ship per ACK-less stretch.
//    Inferable: no — assert SHAPE: no post-continue segment exceeds the
//    bound.
func TestDetail02(t *testing.T) {
	commonHash := plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	reader := &bytes.Buffer{}
	pktline.WriteString(reader, "ACK "+commonHash.String()+" continue\n")
	for i := 0; i < 8; i++ {
		pktline.WriteString(reader, "NAK\n")
	}

	writer := newMockWriteCloser(nil)
	req := &FetchRequest{
		Wants: []plumbing.Hash{plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")},
		Haves: synthHaves(600),
	}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), capability.List{}, true, reader, writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack: %v", err)
	}
	segs := segments(writer.writeBuf.String())
	if len(segs) < 2 {
		t.Fatalf("expected multiple rounds, got %d segments", len(segs))
	}
	for i, s := range segs {
		if i == 0 {
			continue // first round is the pre-continue batch
		}
		if n := strings.Count(s, "have "); n > maxInVein {
			t.Fatalf("segment %d sent %d haves, over the %d vein budget", i, n, maxInVein)
		}
	}
}

// 3. The loop ends when haves run out and on a trailing done round after
//    the server reports ready. Inferable: partially.
func TestDetail03(t *testing.T) {
	want := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")

	// Haves run out under plain NAKs.
	reader := nakLines(6)
	writer := newMockWriteCloser(nil)
	req := &FetchRequest{Wants: []plumbing.Hash{want}, Haves: synthHaves(20)}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), capability.List{}, false, reader, writer, req)
	if err != nil {
		t.Fatalf("haves-out negotiation: %v", err)
	}
	if !strings.Contains(writer.writeBuf.String(), "done") {
		t.Fatal("no done line written when haves ran out")
	}

	// Ready triggers a terminal done round without exhausting haves.
	reader = &bytes.Buffer{}
	pktline.WriteString(reader, "ACK "+want.String()+" ready\n")
	writer = newMockWriteCloser(nil)
	req = &FetchRequest{Wants: []plumbing.Hash{want}, Haves: synthHaves(40)}
	_, err = NegotiatePack(context.TODO(), memory.NewStorage(), capability.List{}, false, reader, writer, req)
	if err != nil {
		t.Fatalf("ready negotiation: %v", err)
	}
	if !strings.Contains(writer.writeBuf.String(), "done") {
		t.Fatal("no done round after server ready")
	}
}

// 4. Wants ⊆ haves with no shallow state → no request at all, flush +
//    close + ErrNoChange. Inferable: partially.
func TestDetail04(t *testing.T) {
	a := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	b := plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	c := plumbing.NewHash("cccccccccccccccccccccccccccccccccccccccc")

	if !isSubset([]plumbing.Hash{a}, []plumbing.Hash{a, b}) {
		t.Fatal("isSubset: subset reported false")
	}
	if isSubset([]plumbing.Hash{a, c}, []plumbing.Hash{a, b}) {
		t.Fatal("isSubset: superset reported true")
	}
	if !isSubset(nil, []plumbing.Hash{a}) {
		t.Fatal("isSubset: empty needle reported false")
	}

	writer := newMockWriteCloser(nil)
	req := &FetchRequest{Wants: []plumbing.Hash{a}, Haves: []plumbing.Hash{a, b}}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), capability.List{}, false, nakLines(0), writer, req)
	if !errors.Is(err, ErrNoChange) {
		t.Fatalf("err %v, want ErrNoChange", err)
	}
	if strings.Contains(writer.writeBuf.String(), "want ") {
		t.Fatal("a want line was sent for an already-satisfied fetch")
	}
	if writer.closes == 0 {
		t.Fatal("writer was not closed")
	}
}

// 5. Stateless RPC: the writer closes after each round and every round
//    re-sends the accumulated common haves plus the request preamble.
//    Inferable: partially.
func TestDetail05(t *testing.T) {
	commonHash := plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	reader := &bytes.Buffer{}
	pktline.WriteString(reader, "ACK "+commonHash.String()+" common\n")
	for i := 0; i < 6; i++ {
		pktline.WriteString(reader, "NAK\n")
	}
	writer := newMockWriteCloser(nil)
	haves := append(synthHaves(15), commonHash)
	haves = append(haves, synthHaves(2, plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa01"))...)
	req := &FetchRequest{
		Wants: []plumbing.Hash{plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")},
		Haves: haves,
	}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), capability.List{}, true, reader, writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack: %v", err)
	}
	out := writer.writeBuf.String()
	if writer.closes < 2 {
		t.Fatalf("writer closed %d times in a multi-round stateless negotiation", writer.closes)
	}
	if n := strings.Count(out, "have "+commonHash.String()); n != 2 {
		t.Fatalf("common hash sent %d times, want re-send on the second round", n)
	}
	segs := segments(out)
	wantSegs := 0
	for _, seg := range segs {
		if strings.Contains(seg, "want ") {
			wantSegs++
		}
	}
	if wantSegs != 2 {
		t.Fatalf("request preamble sent in %d rounds, want 2", wantSegs)
	}
}

// 6. ACK handling: each status resets the vein counter; only ACKCommon
//    accumulates the common set, and under stateless RPC only a NEW
//    common hash resets the vein. Inferable: no — assert the observable
//    state transitions.
func TestDetail06(t *testing.T) {
	h := plumbing.NewHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	run := func(stateless bool, acks []packp.ACK, common map[plumbing.Hash]struct{}, slCommon []plumbing.Hash, inVein int) (map[plumbing.Hash]struct{}, []plumbing.Hash, bool, bool, int) {
		gotContinue, gotReady := false, false
		applyServerACKs(stateless, acks, common, &slCommon, &gotContinue, &gotReady, &inVein)
		return common, slCommon, gotContinue, gotReady, inVein
	}

	// Continue resets vein, sets continue flag, touches nothing else.
	common, sl, cont, ready, vein := run(false, []packp.ACK{{Status: packp.ACKContinue}}, map[plumbing.Hash]struct{}{}, nil, 123)
	if !cont || ready || vein != 0 || len(common) != 0 || len(sl) != 0 {
		t.Fatalf("continue: cont=%v ready=%v vein=%d common=%v sl=%v", cont, ready, vein, common, sl)
	}
	// Ready resets vein and sets both flags.
	common, sl, cont, ready, vein = run(false, []packp.ACK{{Status: packp.ACKReady}}, map[plumbing.Hash]struct{}{}, nil, 42)
	if !cont || !ready || vein != 0 || len(common) != 0 {
		t.Fatalf("ready: cont=%v ready=%v vein=%d common=%v", cont, ready, vein, common)
	}
	// Common (non-stateless) accumulates the common set and implies
	// continue, but does not itself touch the vein or the requeue list —
	// a common-driven vein reset is a stateless-only behaviour.
	common, sl, cont, ready, vein = run(false, []packp.ACK{{Hash: h, Status: packp.ACKCommon}}, map[plumbing.Hash]struct{}{}, nil, 7)
	if _, ok := common[h]; !ok || vein != 7 || !cont || len(sl) != 0 {
		t.Fatalf("stateful common: common=%v vein=%d cont=%v sl=%v", common, vein, cont, sl)
	}
	// Stateless common: accumulates BOTH sets and resets vein.
	common, sl, cont, ready, vein = run(true, []packp.ACK{{Hash: h, Status: packp.ACKCommon}}, map[plumbing.Hash]struct{}{}, nil, 9)
	if _, ok := common[h]; !ok || vein != 0 || len(sl) != 1 || sl[0] != h {
		t.Fatalf("stateless common: common=%v vein=%d sl=%v", common, vein, sl)
	}
	// Duplicate stateless common: no requeue, no vein reset.
	common, sl, cont, ready, vein = run(true,
		[]packp.ACK{{Hash: h, Status: packp.ACKCommon}},
		map[plumbing.Hash]struct{}{h: {}}, []plumbing.Hash{h}, 55)
	if len(sl) != 1 || vein != 55 {
		t.Fatalf("duplicate stateless common: sl=%v vein=%d", sl, vein)
	}
}

// 7. multi_ack_detailed is requested in preference to multi_ack — never
//    both. Inferable: partially.
func TestDetail07(t *testing.T) {
	var caps capability.List
	caps.Set(capability.MultiACK)
	caps.Set(capability.MultiACKDetailed)

	writer := newMockWriteCloser(nil)
	req := &FetchRequest{
		Wants: []plumbing.Hash{plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")},
		Haves: synthHaves(3),
	}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), caps, false, nakLines(3), writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack: %v", err)
	}
	out := writer.writeBuf.String()
	if !strings.Contains(out, "multi_ack_detailed") {
		t.Fatal("multi_ack_detailed not requested")
	}
	for _, tok := range strings.Fields(out) {
		if tok == "multi_ack" {
			t.Fatal("bare multi_ack requested alongside multi_ack_detailed")
		}
	}
}

// 8. sideband-64k chosen over sideband when both offered; no progress
//    channel selects no-progress. Inferable: partially.
func TestDetail08(t *testing.T) {
	var caps capability.List
	caps.Set(capability.Sideband)
	caps.Set(capability.Sideband64k)

	writer := newMockWriteCloser(nil)
	req := &FetchRequest{
		Wants:    []plumbing.Hash{plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")},
		Haves:    synthHaves(3),
		Progress: io.Discard,
	}
	_, err := NegotiatePack(context.TODO(), memory.NewStorage(), caps, false, nakLines(3), writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack: %v", err)
	}
	out := writer.writeBuf.String()
	if !strings.Contains(out, "side-band-64k") {
		t.Fatal("side-band-64k not requested")
	}
	for _, tok := range strings.Fields(out) {
		if tok == "side-band" || tok == "no-progress" {
			t.Fatalf("unexpected token %q", tok)
		}
	}

	// No progress channel → the request advertises no progress capability.
	writer = newMockWriteCloser(nil)
	req.Progress = nil
	_, err = NegotiatePack(context.TODO(), memory.NewStorage(), caps, false, nakLines(3), writer, req)
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range strings.Fields(writer.writeBuf.String()) {
		if tok == "side-band" || tok == "side-band-64k" || tok == "no-progress" {
			t.Fatalf("unexpected progress token %q", tok)
		}
	}
}

// 9. Depth>0 requires the shallow capability or hard-fails; the request
//    carries the repo's current shallow list. Inferable: partially.
func TestDetail09(t *testing.T) {
	want := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	shallowH := plumbing.NewHash("dddddddddddddddddddddddddddddddddddddddd")

	st := memory.NewStorage()
	if err := st.SetShallow([]plumbing.Hash{shallowH}); err != nil {
		t.Fatal(err)
	}

	// No shallow capability → hard fail.
	writer := newMockWriteCloser(nil)
	req := &FetchRequest{Wants: []plumbing.Hash{want}, Haves: synthHaves(3), Depth: 2}
	_, err := NegotiatePack(context.TODO(), st, capability.List{}, false, nakLines(3), writer, req)
	if err == nil {
		t.Fatal("depth without shallow capability did not error")
	}

	// With the capability, the shallow list is announced.
	var caps capability.List
	caps.Set(capability.Shallow)
	// A depth request makes each response open with a shallow-update
	// section — empty here — terminated by a flush.
	depthReader := &bytes.Buffer{}
	pktline.WriteFlush(depthReader)
	for i := 0; i < 4; i++ {
		pktline.WriteString(depthReader, "NAK\n")
	}
	writer = newMockWriteCloser(nil)
	_, err = NegotiatePack(context.TODO(), st, caps, false, depthReader, writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack with shallow cap: %v", err)
	}
	out := writer.writeBuf.String()
	if !strings.Contains(out, "shallow "+shallowH.String()) {
		t.Fatalf("shallow list not announced; out=%q", out)
	}
}

// 10. Object-format reconciliation: mismatch is a hard error except the
//     fresh-clone case (unset client format + placeholder HEAD) which
//     adopts the server's algorithm. Inferable: partially.
func TestDetail10(t *testing.T) {
	withFormat := func(v string) capability.List {
		var caps capability.List
		caps.Set(capability.ObjectFormat, v)
		return caps
	}

	// Unknown server algorithm → error.
	if err := ReconcileObjectFormatV2(memory.NewStorage(), withFormat("sha999")); err == nil {
		t.Fatal("unknown algorithm accepted")
	}

	// Matching sha1 → fine.
	if err := ReconcileObjectFormatV2(memory.NewStorage(), withFormat("sha1")); err != nil {
		t.Fatalf("matching sha1: %v", err)
	}

	// Fresh clone: unset client format + placeholder HEAD + sha256 server
	// → adopts sha256.
	st := memory.NewStorage()
	if err := st.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.Invalid)); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileObjectFormatV2(st, withFormat("sha256")); err != nil {
		t.Fatalf("fresh-clone reconcile: %v", err)
	}
	cfg, err := st.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Extensions.ObjectFormat != formatcfg.SHA256 {
		t.Fatalf("client did not adopt server format: %q", cfg.Extensions.ObjectFormat)
	}

	// Mismatch on an initialised repo → error.
	st2 := memory.NewStorage(memory.WithObjectFormat(formatcfg.SHA1))
	if err := st2.SetReference(plumbing.NewHashReference("refs/heads/main", plumbing.NewHash("1111111111111111111111111111111111111111"))); err != nil {
		t.Fatal(err)
	}
	if err := st2.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, "refs/heads/main")); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileObjectFormatV2(st2, withFormat("sha256")); err == nil {
		t.Fatal("mismatched format on initialised repo accepted")
	}
}

// 11. A server advertising no object-format speaks sha1 only — a client
//     on another algorithm fails rather than corrupting the pack.
//     Inferable: no — assert SHAPE: error for sha256 client, nil for
//     sha1/unset client.
func TestDetail11(t *testing.T) {
	sha256st := memory.NewStorage(memory.WithObjectFormat(formatcfg.SHA256))
	if err := ReconcileObjectFormatV2(sha256st, capability.List{}); err == nil {
		t.Fatal("sha256 client accepted a server with no object-format")
	}
	if err := ReconcileObjectFormatV2(memory.NewStorage(), capability.List{}); err != nil {
		t.Fatalf("unset/sha1 client rejected: %v", err)
	}
}

// 12. The shallow-update response is decoded when depth was requested —
//     the returned ShallowUpdate carries the server's shallow lines.
//     Inferable: partially.
func TestDetail12(t *testing.T) {
	want := plumbing.NewHash("6ecf0ef2c2dffb796033e5a02219af86ec6584e5")
	shallowH := plumbing.NewHash("dddddddddddddddddddddddddddddddddddddddd")

	st := memory.NewStorage()
	var caps capability.List
	caps.Set(capability.Shallow)

	reader := &bytes.Buffer{}
	pktline.WriteString(reader, "shallow "+shallowH.String()+"\n")
	pktline.WriteFlush(reader)
	for i := 0; i < 4; i++ {
		pktline.WriteString(reader, "NAK\n")
	}

	writer := newMockWriteCloser(nil)
	req := &FetchRequest{Wants: []plumbing.Hash{want}, Haves: synthHaves(3), Depth: 1}
	su, err := NegotiatePack(context.TODO(), st, caps, false, reader, writer, req)
	if err != nil {
		t.Fatalf("NegotiatePack: %v", err)
	}
	if su == nil {
		t.Fatal("shallow update not decoded when depth was requested")
	} else {
		found := false
		for _, h := range su.Shallows {
			if h == shallowH {
				found = true
			}
		}
		if !found {
			t.Fatalf("shallow %v absent from update %+v", shallowH, su.Shallows)
		}
	}

	// No depth requested → no shallow update surfaced.
	reader2 := nakLines(4)
	writer2 := newMockWriteCloser(nil)
	req2 := &FetchRequest{Wants: []plumbing.Hash{want}, Haves: synthHaves(3)}
	su2, err := NegotiatePack(context.TODO(), memory.NewStorage(), caps, false, reader2, writer2, req2)
	if err != nil {
		t.Fatalf("NegotiatePack no-depth: %v", err)
	}
	if su2 != nil {
		t.Fatalf("shallow update surfaced without a depth request: %+v", su2)
	}
}
