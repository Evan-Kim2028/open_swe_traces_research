// Hidden black-box property suite for the stree unit.
// Drives only the exported API in server/stree (api.md): NewSubjectTree,
// Size, Empty, Insert, Find, Delete, Match, MatchUntil, IterOrdered,
// IterFast, LazyIntersect, IntersectGSL.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package stree_test

import (
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	gsl "example.internal/msgkit/v2/server/gsl"
	stree "example.internal/msgkit/v2/server/stree"
)

const bbStHiddenSeed = 20260919

func bbStSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbStHiddenSeed
}

func bbStRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbStSeed()))
}

func bbStRandToken(rng *rand.Rand) string {
	const alpha = "abcdefgh0123456789"
	n := 1 + rng.Intn(6)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alpha[rng.Intn(len(alpha))])
	}
	return sb.String()
}

func bbStRandSubject(rng *rand.Rand, maxToks int) string {
	n := 1 + rng.Intn(maxToks)
	var toks []string
	for i := 0; i < n; i++ {
		toks = append(toks, bbStRandToken(rng))
	}
	return strings.Join(toks, ".")
}

// bbStOracle implements the documented filter semantics: '*' covers exactly
// one token; '>' covers all remaining tokens and requires at least one.
func bbStOracle(filter, subj string) bool {
	f := strings.Split(filter, ".")
	s := strings.Split(subj, ".")
	var rec func(fi, si int) bool
	rec = func(fi, si int) bool {
		if fi == len(f) {
			return si == len(s)
		}
		switch {
		case f[fi] == "*":
			return si < len(s) && rec(fi+1, si+1)
		case f[fi] == ">" && fi == len(f)-1:
			return si < len(s) // one or more remaining tokens
		default:
			return si < len(s) && f[fi] == s[si] && rec(fi+1, si+1)
		}
	}
	return rec(0, 0)
}

func bbStMatchSet(st *stree.SubjectTree[int], filter string) map[string]int {
	got := map[string]int{}
	st.Match([]byte(filter), func(subject []byte, val *int) {
		got[string(subject)]++
	})
	return got
}

// Detail 1: Insert returns the previous value pointer and true when the
// exact subject already exists, otherwise nil+false; nil receiver -> nil+false.
func TestDetail01_InsertReturnContract(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	n := 0
	for i := 0; i < 60; i++ {
		subj := bbStRandSubject(rng, 4)
		old, upd := st.Insert([]byte(subj), i+1)
		if old != nil {
			// Already existed: size unchanged, value replaced.
			if *old >= i+1 {
				t.Fatalf("old value %d >= new %d", *old, i+1)
			}
			if !upd {
				t.Fatalf("reinsert of %q returned updated=false", subj)
			}
			if st.Size() != n {
				t.Fatalf("size=%d changed on reinsert, want %d", st.Size(), n)
			}
		} else {
			if upd {
				t.Fatalf("first insert of %q returned updated=true", subj)
			}
			n++
			if st.Size() != n {
				t.Fatalf("size=%d want %d", st.Size(), n)
			}
		}
	}
	// Same subject again returns the old value.
	old, upd := st.Insert([]byte("zz.zzz"), 100)
	if old != nil || upd {
		t.Fatalf("first zz.zzz insert=(%v,%v)", old, upd)
	}
	old, upd = st.Insert([]byte("zz.zzz"), 200)
	if old == nil || *old != 100 || !upd {
		t.Fatalf("reinsert=(%v,%v) want old=100,true", old, upd)
	}
	// Nil receiver.
	var nilSt *stree.SubjectTree[int]
	if old, upd := nilSt.Insert([]byte("x"), 1); old != nil || upd {
		t.Fatalf("nil Insert=(%v,%v) want nil,false", old, upd)
	}
}

// Detail 2: a subject containing byte 127 is refused on insert — nil+false,
// size unchanged.
func TestDetail02_NoPivotByteRefused(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	st.Insert([]byte("ok.sub"), 1)
	before := st.Size()
	for _, subj := range []string{"foo\x7fbar", "\x7f", "a.\x7f.b", "end\x7f"} {
		old, upd := st.Insert([]byte(subj), 9)
		if old != nil || upd {
			t.Fatalf("Insert(%q)=(%v,%v) want nil,false", subj, old, upd)
		}
	}
	if st.Size() != before {
		t.Fatalf("size=%d after refused inserts, want %d", st.Size(), before)
	}
	if _, found := st.Find([]byte("foo\x7fbar")); found {
		t.Fatal("refused subject findable")
	}
}

// Detail 3: Find returns the value pointer and true only on an exact
// literal subject; prefix/shorter/longer misses return nil+false; nil safe.
func TestDetail03_ExactFind(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	subs := map[string]int{}
	for i := 0; i < 80; i++ {
		subj := bbStRandSubject(rng, 4)
		if _, ok := subs[subj]; !ok {
			subs[subj] = i + 1
			st.Insert([]byte(subj), i+1)
		}
	}
	for subj, v := range subs {
		got, found := st.Find([]byte(subj))
		if !found || got == nil || *got != v {
			t.Fatalf("Find(%q)=(%v,%v) want %d,true", subj, got, found, v)
		}
	}
	// Misses: longer/shorter/prefix variants.
	st.Insert([]byte("foo.bar.baz"), 1)
	for _, miss := range []string{"foo.bar", "foo.bar.baz.qux", "foo.ba", "foo.bar.bazz", ""} {
		if got, found := st.Find([]byte(miss)); found || got != nil {
			t.Fatalf("Find(%q)=(%v,%v) want nil,false", miss, got, found)
		}
	}
	var nilSt *stree.SubjectTree[int]
	if got, found := nilSt.Find([]byte("x")); found || got != nil {
		t.Fatalf("nil Find=(%v,%v)", got, found)
	}
}

// Detail 4: Delete returns the value and true only on an exact subject;
// absent/empty/shorter-than-prefix -> nil+false; size drops on success.
func TestDetail04_ExactDelete(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	subs := []string{}
	seen := map[string]bool{}
	for len(subs) < 60 {
		subj := bbStRandSubject(rng, 4)
		if !seen[subj] {
			seen[subj] = true
			subs = append(subs, subj)
			st.Insert([]byte(subj), len(subs))
		}
	}
	// Misses.
	for _, miss := range []string{"", "nope", subs[0] + ".x", subs[0][:len(subs[0])-1]} {
		if got, ok := st.Delete([]byte(miss)); ok || got != nil {
			t.Fatalf("Delete(%q)=(%v,%v) want nil,false", miss, got, ok)
		}
	}
	// Exact deletes return value and shrink.
	for i, subj := range subs {
		got, ok := st.Delete([]byte(subj))
		if !ok || got == nil || *got != i+1 {
			t.Fatalf("Delete(%q)=(%v,%v) want %d,true", subj, got, ok, i+1)
		}
		if st.Size() != len(subs)-i-1 {
			t.Fatalf("size=%d want %d", st.Size(), len(subs)-i-1)
		}
		if got, ok := st.Delete([]byte(subj)); ok || got != nil {
			t.Fatalf("second Delete(%q)=(%v,%v)", subj, got, ok)
		}
	}
	if st.Size() != 0 {
		t.Fatalf("size=%d after all deletes", st.Size())
	}
	var nilSt *stree.SubjectTree[int]
	if got, ok := nilSt.Delete([]byte("x")); ok || got != nil {
		t.Fatalf("nil Delete=(%v,%v)", got, ok)
	}
}

// Detail 5: Size counts distinct subjects; Empty clears; nil receiver gets
// a fresh empty tree back.
func TestDetail05_SizeAndEmpty(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	distinct := map[string]bool{}
	for i := 0; i < 100; i++ {
		subj := bbStRandSubject(rng, 3)
		st.Insert([]byte(subj), i)
		distinct[subj] = true
	}
	if st.Size() != len(distinct) {
		t.Fatalf("size=%d want %d distinct", st.Size(), len(distinct))
	}
	st = st.Empty()
	if st == nil || st.Size() != 0 {
		t.Fatalf("Empty: size=%d", st.Size())
	}
	var nilSt *stree.SubjectTree[int]
	fresh := nilSt.Empty()
	if fresh == nil || fresh.Size() != 0 {
		t.Fatal("nil Empty should return fresh empty tree")
	}
}

// Detail 6: '*' and '>' wildcards only when token-aligned — at start or
// immediately after '.', AND at end or immediately before '.'; '>' must be
// the last character; otherwise literal.
func TestDetail06_TokenAlignedWildcards(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	// Store subjects including literal-wildcard tokens.
	st.Insert([]byte("foo.123"), 1)
	st.Insert([]byte("'*.123"), 2)
	st.Insert([]byte("bar"), 3)
	st.Insert([]byte("a>b.c"), 4)
	st.Insert([]byte("x*y.z"), 5)
	// `'*.*`: first '*' is mid-token (after ') -> literal; second '*' aligned -> wildcard.
	got := bbStMatchSet(st, "'*.*")
	if len(got) != 1 || got["'*.123"] != 1 {
		t.Fatalf("'*.* matched %v want {'*.123:1}", got)
	}
	if got["bar"] != 0 {
		t.Fatal("'*.* must not match bar")
	}
	// Mid-token '>' is literal.
	got = bbStMatchSet(st, "a>b.c")
	if len(got) != 1 || got["a>b.c"] != 1 {
		t.Fatalf("a>b.c matched %v want {a>b.c:1}", got)
	}
	// '>' not at end is literal — "a.>b" matches literal subject "a.>b" only.
	st.Insert([]byte("a.>b"), 6)
	got = bbStMatchSet(st, "a.>b")
	if len(got) != 1 || got["a.>b"] != 1 {
		t.Fatalf("a.>b matched %v want {a.>b:1}", got)
	}
	// Mid-token '*' literal both ways.
	got = bbStMatchSet(st, "x*y.z")
	if len(got) != 1 || got["x*y.z"] != 1 {
		t.Fatalf("x*y.z matched %v want {x*y.z:1}", got)
	}
	// Random misaligned wildcards never act as wildcards.
	rng := bbStRng(t)
	st2 := stree.NewSubjectTree[int]()
	stored := map[string]int{}
	for i := 0; i < 40; i++ {
		subj := bbStRandSubject(rng, 3)
		stored[subj] = i
		st2.Insert([]byte(subj), i)
	}
	for i := 0; i < 60; i++ {
		// Build a filter with a wildcard char embedded in a token.
		tok := bbStRandToken(rng)
		filter := tok + []string{"*", ">", "*x", ">x", "x*", "x>"}[rng.Intn(6)] + "." + bbStRandToken(rng)
		got := bbStMatchSet(st2, filter)
		// Filter is fully literal (no aligned wildcards): only exact match allowed.
		for subj := range got {
			if subj != filter {
				t.Fatalf("literal filter %q matched %q", filter, subj)
			}
		}
	}
}

// Detail 7: '*' covers exactly one token; '>' covers all remaining tokens
// and requires at least one — foo.> does not match foo.
func TestDetail07_WildcardArity(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	st.Insert([]byte("foo"), 1)
	st.Insert([]byte("foo.bar"), 2)
	st.Insert([]byte("foo.bar.baz"), 3)
	st.Insert([]byte("one.two"), 4)
	check := func(filter string, want ...string) {
		got := bbStMatchSet(st, filter)
		if len(got) != len(want) {
			t.Fatalf("Match(%q)=%v want %v", filter, got, want)
		}
		for _, w := range want {
			if got[w] == 0 {
				t.Fatalf("Match(%q)=%v missing %q", filter, got, w)
			}
		}
	}
	check("foo.*", "foo.bar")
	check("foo.>", "foo.bar", "foo.bar.baz")
	check("*", "foo")
	check("*.two", "one.two")
	check("*.>", "foo.bar", "foo.bar.baz", "one.two")
	check(">", "foo", "foo.bar", "foo.bar.baz", "one.two")
	// 'foo.>' must not match 'foo' (needs >=1 remaining token).
	if got := bbStMatchSet(st, "foo.>"); got["foo"] != 0 {
		t.Fatal("foo.> matched foo")
	}
	// Oracle cross-check.
	rng := bbStRng(t)
	st2 := stree.NewSubjectTree[int]()
	subs := []string{}
	for i := 0; i < 60; i++ {
		subj := bbStRandSubject(rng, 4)
		if _, dup := st2.Find([]byte(subj)); !dup {
			st2.Insert([]byte(subj), i)
			subs = append(subs, subj)
		}
	}
	filters := []string{"a.>", "*.*", "a.*.c", ">", "a.>.*"}
	for _, f := range filters {
		got := bbStMatchSet(st2, f)
		want := map[string]int{}
		for _, s := range subs {
			if bbStOracle(f, s) {
				want[s] = 1
			}
		}
		if len(got) != len(want) {
			t.Fatalf("Match(%q)=%v want %v", f, got, want)
		}
		for s := range want {
			if got[s] == 0 {
				t.Fatalf("Match(%q) missing %q", f, s)
			}
		}
	}
}

// Detail 8: Match delivers each covered stored subject once with the full
// literal subject bytes and a pointer to its value; a purely literal
// filter delivers only the exact subject.
func TestDetail08_MatchDelivery(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	subs := map[string]int{}
	for len(subs) < 50 {
		subj := bbStRandSubject(rng, 4)
		if _, ok := subs[subj]; !ok {
			subs[subj] = len(subs) + 1
			st.Insert([]byte(subj), len(subs))
		}
	}
	for subj, v := range subs {
		var hits []string
		var vals []int
		st.Match([]byte(subj), func(subject []byte, val *int) {
			hits = append(hits, string(subject))
			if val != nil {
				vals = append(vals, *val)
			}
		})
		if len(hits) != 1 || hits[0] != subj || len(vals) != 1 || vals[0] != v {
			t.Fatalf("literal Match(%q) hits=%v vals=%v want [%q]=%d", subj, hits, vals, subj, v)
		}
	}
	// Each covered subject delivered exactly once under wildcards.
	st2 := stree.NewSubjectTree[int]()
	st2.Insert([]byte("a.b.c"), 1)
	st2.Insert([]byte("a.b.d"), 2)
	st2.Insert([]byte("a.b"), 3)
	got := bbStMatchSet(st2, "a.>")
	for s, c := range got {
		if c != 1 {
			t.Fatalf("subject %q delivered %d times", s, c)
		}
	}
	if len(got) != 3 {
		t.Fatalf("a.> delivered %v want 3 subjects", got)
	}
}

// Detail 9: Match with empty filter or nil callback delivers nothing;
// MatchUntil in those cases reports true (ran to completion).
func TestDetail09_EmptyFilterAndNilCallback(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	st.Insert([]byte("foo.bar"), 1)
	st.Insert([]byte("baz"), 2)
	n := 0
	st.Match(nil, func(subject []byte, val *int) { n++ })
	st.Match([]byte(""), func(subject []byte, val *int) { n++ })
	st.Match([]byte(">"), nil)
	if n != 0 {
		t.Fatalf("empty-filter/nil-cb delivered %d", n)
	}
	if !st.MatchUntil(nil, func(subject []byte, val *int) bool { return false }) {
		t.Fatal("MatchUntil empty filter should complete (true)")
	}
	if !st.MatchUntil([]byte(""), func(subject []byte, val *int) bool { return false }) {
		t.Fatal("MatchUntil '' should complete (true)")
	}
	if !st.MatchUntil([]byte(">"), nil) {
		t.Fatal("MatchUntil nil cb should complete (true)")
	}
	// Empty tree, normal filter -> completes too.
	empty := stree.NewSubjectTree[int]()
	if !empty.MatchUntil([]byte("foo.>"), func(subject []byte, val *int) bool { return true }) {
		t.Fatal("MatchUntil on empty tree should complete")
	}
}

// Detail 10: MatchUntil stops the instant a callback returns false and
// reports false; a full traversal reports true.
func TestDetail10_MatchUntilStop(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	for i := 0; i < 20; i++ {
		st.Insert([]byte("s."+strconv.Itoa(i)), i)
	}
	count := 0
	completed := st.MatchUntil([]byte("s.>"), func(subject []byte, val *int) bool {
		count++
		return count < 5
	})
	if completed {
		t.Fatal("MatchUntil should report false on early stop")
	}
	if count != 5 {
		t.Fatalf("MatchUntil delivered %d want 5", count)
	}
	count = 0
	completed = st.MatchUntil([]byte("s.>"), func(subject []byte, val *int) bool {
		count++
		return true
	})
	if !completed || count != 20 {
		t.Fatalf("full traversal: completed=%v count=%d", completed, count)
	}
	// Stop on first.
	count = 0
	completed = st.MatchUntil([]byte(">"), func(subject []byte, val *int) bool {
		count++
		return false
	})
	if completed || count != 1 {
		t.Fatalf("stop-first: completed=%v count=%d", completed, count)
	}
}

// Detail 11: ordered iterator yields strict lexicographic order and stops
// on false; fast iterator covers the same set in arbitrary order.
func TestDetail11_Iterators(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	subs := map[string]int{}
	for len(subs) < 80 {
		subj := bbStRandSubject(rng, 4)
		if _, ok := subs[subj]; !ok {
			v := len(subs)
			subs[subj] = v
			st.Insert([]byte(subj), v)
		}
	}
	var want []string
	for s := range subs {
		want = append(want, s)
	}
	sort.Strings(want)
	var got []string
	st.IterOrdered(func(subject []byte, val *int) bool {
		got = append(got, string(subject))
		if subs[string(subject)] != *val {
			t.Fatalf("IterOrdered value for %q = %d want %d", subject, *val, subs[string(subject)])
		}
		return true
	})
	if len(got) != len(want) {
		t.Fatalf("IterOrdered delivered %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("IterOrdered[%d]=%q want %q", i, got[i], want[i])
		}
	}
	// Early stop.
	got = got[:0]
	st.IterOrdered(func(subject []byte, val *int) bool {
		got = append(got, string(subject))
		return len(got) < 3
	})
	if len(got) != 3 {
		t.Fatalf("IterOrdered early stop delivered %d", len(got))
	}
	// Fast iterator: same set, arbitrary order.
	fast := map[string]int{}
	st.IterFast(func(subject []byte, val *int) bool {
		fast[string(subject)] = *val
		return true
	})
	if len(fast) != len(subs) {
		t.Fatalf("IterFast delivered %d want %d", len(fast), len(subs))
	}
	for s, v := range subs {
		if fast[s] != v {
			t.Fatalf("IterFast %q=%d want %d", s, fast[s], v)
		}
	}
	// Fast early stop.
	n := 0
	st.IterFast(func(subject []byte, val *int) bool {
		n++
		return n < 7
	})
	if n != 7 {
		t.Fatalf("IterFast early stop delivered %d", n)
	}
}

// Detail 12: LazyIntersect calls back once per subject present in both trees.
func TestDetail12_LazyIntersect(t *testing.T) {
	rng := bbStRng(t)
	st1 := stree.NewSubjectTree[int]()
	st2 := stree.NewSubjectTree[int]()
	common := map[string]bool{}
	for i := 0; i < 80; i++ {
		subj := bbStRandSubject(rng, 3)
		st1.Insert([]byte(subj), i)
		if rng.Intn(2) == 0 {
			st2.Insert([]byte(subj), i+1000)
			common[subj] = true
		}
	}
	for i := 0; i < 20; i++ {
		st2.Insert([]byte("only2."+strconv.Itoa(i)), i)
	}
	got := map[string]int{}
	stree.LazyIntersect(st1, st2, func(key []byte, v1, v2 *int) {
		got[string(key)]++
	})
	if len(got) != len(common) {
		t.Fatalf("LazyIntersect delivered %d want %d", len(got), len(common))
	}
	for s := range common {
		if got[s] != 1 {
			t.Fatalf("LazyIntersect %q delivered %d times", s, got[s])
		}
	}
	// Empty intersections.
	st3 := stree.NewSubjectTree[int]()
	got = map[string]int{}
	stree.LazyIntersect(st1, st3, func(key []byte, v1, v2 *int) { got[string(key)]++ })
	if len(got) != 0 {
		t.Fatalf("empty-tree intersect delivered %v", got)
	}
}

// Detail 13: IntersectGSL calls back once per stored subject that has
// interest in the sublist — deduplicated across overlapping subscriptions.
func TestDetail13_GSLIntersect(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	st.Insert([]byte("one.two.three"), 1)
	st.Insert([]byte("one.two.four"), 2)
	st.Insert([]byte("one.five"), 3)
	st.Insert([]byte("six"), 4)
	sl := gsl.NewSublist[int]()
	// Overlapping subs covering the same stored subjects.
	_ = sl.Insert("one.two.three", 11)
	_ = sl.Insert("one.two.*", 22)
	_ = sl.Insert("one.>", 33)
	got := map[string]int{}
	stree.IntersectGSL(st, sl, func(subj []byte, entry *int) bool {
		got[string(subj)]++
		return true
	})
	want := map[string]bool{"one.two.three": true, "one.two.four": true, "one.five": true}
	if len(got) != len(want) {
		t.Fatalf("IntersectGSL delivered %v want %v", got, want)
	}
	for s := range want {
		if got[s] != 1 {
			t.Fatalf("IntersectGSL %q delivered %d times (dedup)", s, got[s])
		}
	}
	// Early stop.
	n := 0
	stree.IntersectGSL(st, sl, func(subj []byte, entry *int) bool {
		n++
		return n < 2
	})
	if n != 2 {
		t.Fatalf("IntersectGSL early stop delivered %d", n)
	}
}

// Detail 14: compressed-prefix storage is transparent — inserts splitting
// on common prefixes keep size/find/match/iteration correct.
func TestDetail14_PrefixCompressionTransparent(t *testing.T) {
	rng := bbStRng(t)
	st := stree.NewSubjectTree[int]()
	// Adversarial prefix families: long shared prefixes, suffix splits,
	// prefix-of relationships.
	families := [][]string{
		{"a1.aaaaaaaaaaaaaaaaaaaaaa0", "a1.aaaaaaaaaaaaaaaaaaaaaa1", "a2.0", "a2.1"},
		{"x", "x.y", "x.y.z", "x.y.z.w"},
		{"p.q", "p.qr", "p.qrs", "p.qrst"},
		{"same.same.same.same", "same.same.same.diff", "same.same", "same"},
	}
	n := 0
	for _, fam := range families {
		for _, s := range fam {
			if _, upd := st.Insert([]byte(s), n); !upd {
				n++
			}
		}
	}
	// Random additions with long common prefixes.
	for i := 0; i < 60; i++ {
		subj := "pre." + bbStRandToken(rng) + "." + bbStRandSubject(rng, 2)
		if _, upd := st.Insert([]byte(subj), n); !upd {
			n++
		}
	}
	if st.Size() != n {
		t.Fatalf("size=%d want %d", st.Size(), n)
	}
	// All findable, all iterable, delete works through the compressed paths.
	var all []string
	st.IterOrdered(func(subject []byte, val *int) bool {
		all = append(all, string(subject))
		return true
	})
	if len(all) != n {
		t.Fatalf("iterated %d want %d", len(all), n)
	}
	for _, s := range all {
		if _, ok := st.Find([]byte(s)); !ok {
			t.Fatalf("Find(%q) failed", s)
		}
	}
	// Delete all -> size 0.
	for _, s := range all {
		st.Delete([]byte(s))
	}
	if st.Size() != 0 {
		t.Fatalf("size=%d after deletes", st.Size())
	}
}

// Detail 15: a filter that merely shares a prefix with stored subjects but
// is not covered yields nothing — no prefix false-positives.
func TestDetail15_NoPrefixFalsePositives(t *testing.T) {
	st := stree.NewSubjectTree[int]()
	st.Insert([]byte("foo.bar"), 1)
	st.Insert([]byte("foo.baz"), 2)
	st.Insert([]byte("foo"), 3)
	st.Insert([]byte("one.two.three"), 4)
	for _, f := range []string{"foo.ba", "foo.b", "one.two.three.four", "on", "foo.bar.x", "f"} {
		if got := bbStMatchSet(st, f); len(got) != 0 {
			t.Fatalf("Match(%q)=%v want nothing", f, got)
		}
	}
	// Wildcard prefix misses too.
	for _, f := range []string{"foo.bar.*", "one.*.three.four", "*.nope"} {
		if got := bbStMatchSet(st, f); len(got) != 0 {
			t.Fatalf("Match(%q)=%v want nothing", f, got)
		}
	}
}
