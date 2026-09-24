package mocktikv

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/pingcap/kvproto/pkg/kvrpcpb"
)

// Hidden suite for unit mvccread. One TestDetailNN per DETAILS.md line.

// TestDetail01: check passes through when the lock is newer than the read
// or the lock op is Op_Lock / Op_PessimisticLock; a blocking op errors.
func TestDetail01(t *testing.T) {
	key := []byte("k")
	// Lock newer than the read: passthrough regardless of op.
	l := &mvccLock{startTS: 100, primary: []byte("p"), op: kvrpcpb.Op_Put}
	ts, err := l.check(50, key, nil)
	if err != nil || ts != 50 {
		t.Fatalf("newer lock blocked read: ts=%v err=%v", ts, err)
	}
	// Pessimistic and Lock ops never block reads.
	for _, op := range []kvrpcpb.Op{kvrpcpb.Op_Lock, kvrpcpb.Op_PessimisticLock} {
		l := &mvccLock{startTS: 10, primary: []byte("p"), op: op}
		ts, err := l.check(50, key, nil)
		if err != nil || ts != 50 {
			t.Fatalf("op %v blocked read: ts=%v err=%v", op, ts, err)
		}
	}
	// A write op at an older startTS blocks.
	l = &mvccLock{startTS: 10, primary: []byte("p"), op: kvrpcpb.Op_Put}
	if _, err := l.check(50, key, nil); err == nil {
		t.Fatalf("blocking op did not error")
	}
}

// TestDetail02: at ts==MaxUint64 with primary matching the raw key, check
// returns a read timestamp just below the lock's startTS — the lock's
// writer reading its own key. Shape: below startTS, no error, and the
// short-circuit requires both conditions.
func TestDetail02(t *testing.T) {
	key := []byte("k")
	l := &mvccLock{startTS: 42, primary: key, op: kvrpcpb.Op_Put}
	ts, err := l.check(math.MaxUint64, key, nil)
	if err != nil {
		t.Fatalf("own-key max-ts read errored: %v", err)
	}
	if ts >= l.startTS {
		t.Fatalf("short-circuit ts %v not below lock startTS %v", ts, l.startTS)
	}
	// Primary mismatch: no short-circuit, still a lock error.
	l2 := &mvccLock{startTS: 42, primary: []byte("other"), op: kvrpcpb.Op_Put}
	if _, err := l2.check(math.MaxUint64, key, nil); err == nil {
		t.Fatalf("primary mismatch took the short-circuit")
	}
	// Finite ts: no short-circuit even for the primary key.
	if _, err := l.check(100, key, nil); err == nil {
		t.Fatalf("finite ts took the short-circuit")
	}
}

// TestDetail03: a resolved startTS skips the lock; otherwise check returns
// (0, ErrLocked) whose Key is the raw key mvcc-encoded at lockVer.
func TestDetail03(t *testing.T) {
	key := []byte("k")
	l := &mvccLock{startTS: 10, primary: []byte("p"), op: kvrpcpb.Op_Put, ttl: 7, txnSize: 9, forUpdateTS: 11}
	ts, err := l.check(50, key, []uint64{10})
	if err != nil || ts != 50 {
		t.Fatalf("resolved lock blocked: ts=%v err=%v", ts, err)
	}
	ts, err = l.check(50, key, []uint64{11, 12})
	if ts != 0 {
		t.Fatalf("unresolved check returned ts=%v", ts)
	}
	var el *ErrLocked
	if !errors.As(err, &el) {
		t.Fatalf("error is %T, want *ErrLocked", err)
	}
	if !bytes.Equal(el.Key, mvccEncode(key, lockVer)) {
		t.Fatalf("ErrLocked.Key=%x, want mvccEncode(key, lockVer)", []byte(el.Key))
	}
	if !bytes.Equal(el.Primary, []byte("p")) || el.StartTS != 10 || el.TTL != 7 ||
		el.TxnSize != 9 || el.ForUpdateTS != 11 || el.LockType != kvrpcpb.Op_Put {
		t.Fatalf("ErrLocked fields: %+v", el)
	}
}

// TestDetail04: Get runs the lock check only at SI; returns the first
// version with commitTS<=ts skipping rollback/lock values; (nil,nil) when
// nothing matches.
func TestDetail04(t *testing.T) {
	mk := NewMvccKey([]byte("k"))
	e := &mvccEntry{
		key: mk,
		values: []mvccValue{
			{valueType: typePut, startTS: 1, commitTS: 5, value: []byte("v5")},
			{valueType: typePut, startTS: 1, commitTS: 3, value: []byte("v3")},
		},
	}
	v, err := e.Get(4, kvrpcpb.IsolationLevel_SI, nil)
	if err != nil || string(v) != "v3" {
		t.Fatalf("Get(4)=%q,%v want v3", v, err)
	}
	v, err = e.Get(6, kvrpcpb.IsolationLevel_SI, nil)
	if err != nil || string(v) != "v5" {
		t.Fatalf("Get(6)=%q,%v want v5", v, err)
	}
	// No match -> (nil, nil).
	v, err = e.Get(2, kvrpcpb.IsolationLevel_SI, nil)
	if err != nil || v != nil {
		t.Fatalf("Get(2)=%q,%v want nil,nil", v, err)
	}
	// Rollback and lock values are skipped.
	e2 := &mvccEntry{
		key: mk,
		values: []mvccValue{
			{valueType: typeRollback, startTS: 1, commitTS: 9},
			{valueType: typeLock, startTS: 1, commitTS: 8},
			{valueType: typePut, startTS: 1, commitTS: 3, value: []byte("v3")},
		},
	}
	v, err = e2.Get(10, kvrpcpb.IsolationLevel_SI, nil)
	if err != nil || string(v) != "v3" {
		t.Fatalf("rollback/lock skip: %q,%v", v, err)
	}
	// Lock check only under SI.
	e3 := &mvccEntry{
		key:    mk,
		values: []mvccValue{{valueType: typePut, startTS: 1, commitTS: 5, value: []byte("v5")}},
		lock:   &mvccLock{startTS: 4, primary: []byte("p"), op: kvrpcpb.Op_Put},
	}
	if _, err := e3.Get(10, kvrpcpb.IsolationLevel_SI, nil); err == nil {
		t.Fatalf("SI read ignored lock")
	}
	v, err = e3.Get(10, kvrpcpb.IsolationLevel_RC, nil)
	if err != nil || string(v) != "v5" {
		t.Fatalf("RC read blocked by lock: %q,%v", v, err)
	}
	// Resolved lock under SI passes through.
	v, err = e3.Get(10, kvrpcpb.IsolationLevel_SI, []uint64{4})
	if err != nil || string(v) != "v5" {
		t.Fatalf("resolved SI read: %q,%v", v, err)
	}
}

// TestDetail05: regionContains is [startKey, endKey) with empty endKey
// unbounded.
func TestDetail05(t *testing.T) {
	cases := []struct {
		start, end, key []byte
		want            bool
	}{
		{[]byte("a"), []byte("c"), []byte("b"), true},
		{[]byte("a"), []byte("c"), []byte("a"), true},  // inclusive start
		{[]byte("a"), []byte("c"), []byte("c"), false}, // exclusive end
		{[]byte("a"), []byte("c"), []byte("0"), false},
		{[]byte("a"), nil, []byte("z"), true},          // unbounded end
		{[]byte("a"), nil, []byte("0"), false},
		{nil, []byte("z"), []byte("a"), true},          // empty start
	}
	for i, c := range cases {
		if got := regionContains(c.start, c.end, c.key); got != c.want {
			t.Fatalf("case %d: regionContains(%q,%q,%q)=%v want %v", i, c.start, c.end, c.key, got, c.want)
		}
	}
}

// TestDetail06: mvccEntry.Less is bytes.Compare on the ENCODED keys.
func TestDetail06(t *testing.T) {
	a := &mvccEntry{key: NewMvccKey([]byte("a"))}
	b := &mvccEntry{key: NewMvccKey([]byte("b"))}
	if !a.Less(b) || b.Less(a) || a.Less(a) {
		t.Fatalf("Less ordering wrong")
	}
	// Ordering is on encoded bytes: Less must agree with bytes.Compare of
	// the encoded forms, whatever the encoding's order is.
	for _, pair := range [][2]string{{"x", "y"}, {"y", "x"}, {"k\x00", "k"}} {
		x := NewMvccKey([]byte(pair[0]))
		y := NewMvccKey([]byte(pair[1]))
		e1 := &mvccEntry{key: x}
		e2 := &mvccEntry{key: y}
		want := bytes.Compare(x, y) < 0
		if e1.Less(e2) != want {
			t.Fatalf("Less(%q,%q)=%v want %v", pair[0], pair[1], e1.Less(e2), want)
		}
	}
}
