// Hidden black-box property suite for the gslsublist unit.
// Drives only the exported API in server/gsl (api.md): NewSublist,
// NewSimpleSublist, Insert, Remove, Match, MatchBytes, HasInterest,
// NumInterest, MatchesFullWildcard, MatchesSingleFilter,
// HasInterestStartingIn, Count, and the documented error values.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package gsl_test

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	gsl "example.internal/msgkit/v2/server/gsl"
)

const bbGslHiddenSeed = 20260919

func bbGslSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbGslHiddenSeed
}

func bbGslRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbGslSeed()))
}

func bbGslValidSubject(subj string) bool {
	if subj == "" {
		return false
	}
	toks := strings.Split(subj, ".")
	for i, tok := range toks {
		if tok == "" {
			return false
		}
		if tok == ">" && i != len(toks)-1 {
			return false
		}
	}
	return true
}

// bbGslOracle is a local reference matcher implementing the documented
// wildcard semantics: '*' covers exactly one token, '>' covers one or more
// trailing tokens; multi-char tokens containing wildcards are literals.
func bbGslOracle(filter, subj string) bool {
	f := strings.Split(filter, ".")
	s := strings.Split(subj, ".")
	var rec func(fi, si int) bool
	rec = func(fi, si int) bool {
		if fi == len(f) {
			return si == len(s)
		}
		switch f[fi] {
		case "*":
			return si < len(s) && rec(fi+1, si+1)
		case ">":
			return si < len(s) // one or more remaining tokens
		default:
			return si < len(s) && f[fi] == s[si] && rec(fi+1, si+1)
		}
	}
	return rec(0, 0)
}

func bbGslRandToken(rng *rand.Rand) string {
	const alpha = "abcdefghxyz012"
	n := 1 + rng.Intn(4)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(alpha[rng.Intn(len(alpha))])
	}
	return sb.String()
}

func bbGslRandSubject(rng *rand.Rand, maxToks int) string {
	n := 1 + rng.Intn(maxToks)
	var toks []string
	for i := 0; i < n; i++ {
		toks = append(toks, bbGslRandToken(rng))
	}
	return strings.Join(toks, ".")
}

// Detail 1: '*' and '>' are wildcards only as whole single-character
// tokens; multi-char tokens containing them are literals.
func TestDetail01_WildcardOnlyWholeTokens(t *testing.T) {
	rng := bbGslRng(t)
	s := gsl.NewSublist[int]()
	// Literal tokens embedding wildcard chars.
	lits := []string{"foo.*-", "foo.>-", "a.*x.b", "z.>tail.y"}
	for i, sub := range lits {
		if err := s.Insert(sub, i+1); err != nil {
			t.Fatalf("insert literal %q: %v", sub, err)
		}
	}
	// They match only literally.
	for i, sub := range lits {
		var got []int
		s.Match(sub, func(v int) { got = append(got, v) })
		if len(got) != 1 || got[0] != i+1 {
			t.Fatalf("Match(%q)=%v want [%d]", sub, got, i+1)
		}
	}
	// They do not act as wildcards.
	var got []int
	s.Match("foo.anything", func(v int) { got = append(got, v) })
	if len(got) != 0 {
		t.Fatalf("foo.anything matched %v — '*-' must be literal", got)
	}
	got = got[:0]
	s.Match("a.*x.b", func(v int) { got = append(got, v) })
	if len(got) != 1 || got[0] != 3 {
		t.Fatalf("a.*x.b=%v want [3]", got)
	}
	// Random literal-with-wildcard-char tokens.
	s2 := gsl.NewSublist[int]()
	toks := []string{}
	seen := map[string]bool{}
	for len(toks) < 20 {
		tok := bbGslRandToken(rng) + []string{"*-", ">-", "*x", ">x"}[rng.Intn(4)]
		if seen[tok] {
			continue
		}
		seen[tok] = true
		toks = append(toks, tok)
		if err := s2.Insert("pre."+tok, len(toks)-1); err != nil {
			t.Fatalf("insert %q: %v", tok, err)
		}
	}
	for i, tok := range toks {
		var hit []int
		s2.Match("pre."+tok, func(v int) { hit = append(hit, v) })
		if len(hit) != 1 || hit[0] != i {
			t.Fatalf("literal %q match=%v want [%d]", tok, hit, i)
		}
	}
	// A real '*' doesn't match those literals.
	var hit []int
	s2.Match("pre.*", func(v int) { hit = append(hit, v) })
	if len(hit) != 0 {
		t.Fatalf("'*' matched literal-wildcard tokens: %v", hit)
	}
}

// Detail 2: '>' is legal only as the terminal token — any token after it
// fails ErrInvalidSubject on insert and remove.
func TestDetail02_FullWildcardTerminalOnly(t *testing.T) {
	s := gsl.NewSublist[int]()
	for _, bad := range []string{"a.>.b", ">.x", "a.>.>", "a.>.b.c"} {
		if err := s.Insert(bad, 1); !errors.Is(err, gsl.ErrInvalidSubject) {
			t.Fatalf("Insert(%q): %v want ErrInvalidSubject", bad, err)
		}
		if err := s.Remove(bad, 1); !errors.Is(err, gsl.ErrInvalidSubject) {
			t.Fatalf("Remove(%q): %v want ErrInvalidSubject", bad, err)
		}
	}
	// Terminal '>' is legal.
	if err := s.Insert("a.>", 1); err != nil {
		t.Fatalf("Insert(a.>): %v", err)
	}
	if err := s.Remove("a.>", 1); err != nil {
		t.Fatalf("Remove(a.>): %v", err)
	}
}

// Detail 3: empty tokens are invalid on insert and remove — leading or
// trailing separator, or '..' anywhere.
func TestDetail03_EmptyTokensInvalid(t *testing.T) {
	s := gsl.NewSublist[int]()
	for _, bad := range []string{".a.b", "a.b.", "a..b", "a.b..c", ".", "..", "a..", "..a"} {
		if err := s.Insert(bad, 1); !errors.Is(err, gsl.ErrInvalidSubject) {
			t.Fatalf("Insert(%q): %v want ErrInvalidSubject", bad, err)
		}
		if err := s.Remove(bad, 1); !errors.Is(err, gsl.ErrInvalidSubject) {
			t.Fatalf("Remove(%q): %v want ErrInvalidSubject", bad, err)
		}
	}
	if s.Count() != 0 {
		t.Fatalf("count=%d after invalid inserts", s.Count())
	}
}

// Detail 4: matching a malformed literal subject yields zero callbacks —
// no error, no panic.
func TestDetail04_MatchMalformedSubject(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("a.b", 1)
	_ = s.Insert("a.*", 2)
	_ = s.Insert("a.>", 3)
	_ = s.Insert(">", 4)
	for _, bad := range []string{".a.b", "a.b.", "a..b", ".", ".."} {
		n := 0
		s.Match(bad, func(v int) { n++ })
		if n != 0 {
			t.Fatalf("Match(%q) delivered %d callbacks, want 0", bad, n)
		}
		s.MatchBytes([]byte(bad), func(v int) { n++ })
		if n != 0 {
			t.Fatalf("MatchBytes(%q) delivered %d callbacks, want 0", bad, n)
		}
	}
}

// Detail 5: '*' covers exactly one token.
func TestDetail05_PartialWildcardArity(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("foo.*", 1)
	_ = s.Insert("*", 2)
	check := func(subj string, want []int) {
		var got []int
		s.Match(subj, func(v int) { got = append(got, v) })
		if len(got) != len(want) {
			t.Fatalf("Match(%q)=%v want %v", subj, got, want)
		}
		for _, w := range want {
			found := false
			for _, g := range got {
				if g == w {
					found = true
				}
			}
			if !found {
				t.Fatalf("Match(%q)=%v missing %d", subj, got, w)
			}
		}
	}
	check("foo.bar", []int{1})
	check("foo", []int{2})
	check("foo.bar.baz", nil)
	check("one", []int{2})
	check("one.two", nil)
	// Randomized against oracle.
	rng := bbGslRng(t)
	s2 := gsl.NewSublist[int]()
	filters := []string{}
	for i := 0; i < 40; i++ {
		var f string
		switch rng.Intn(4) {
		case 0:
			f = bbGslRandToken(rng) + ".*"
		case 1:
			f = "*." + bbGslRandToken(rng)
		case 2:
			f = bbGslRandToken(rng) + "." + bbGslRandToken(rng) + ".*"
		default:
			f = bbGslRandSubject(rng, 3)
		}
		filters = append(filters, f)
		_ = s2.Insert(f, i)
	}
	for i := 0; i < 100; i++ {
		subj := bbGslRandSubject(rng, 4)
		var got []int
		s2.Match(subj, func(v int) { got = append(got, v) })
		want := map[int]bool{}
		for j, f := range filters {
			if bbGslOracle(f, subj) {
				want[j] = true
			}
		}
		if len(got) != len(want) {
			t.Fatalf("Match(%q)=%v want %d filters %v", subj, got, len(want), want)
		}
		for _, g := range got {
			if !want[g] {
				t.Fatalf("Match(%q) delivered filter %d which oracle rejects", subj, g)
			}
		}
	}
}

// Detail 6: '>' covers one or more trailing tokens; '*.>' requires >=2.
func TestDetail06_FullWildcardArity(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("foo.>", 1)
	_ = s.Insert("*.>", 2)
	_ = s.Insert(">", 3)
	matchN := func(subj string) map[int]bool {
		got := map[int]bool{}
		s.Match(subj, func(v int) { got[v] = true })
		return got
	}
	if m := matchN("foo.bar"); !m[1] || !m[3] {
		t.Fatalf("foo.bar matched %v, want 1 and 3", m)
	}
	if m := matchN("foo.bar.baz"); !m[1] || !m[3] {
		t.Fatalf("foo.bar.baz matched %v, want 1 and 3", m)
	}
	if m := matchN("foo"); m[1] {
		t.Fatalf("foo matched foo.> — '>' needs at least one token")
	}
	// '*.>' needs at least two tokens.
	if m := matchN("foo"); m[2] {
		t.Fatalf("foo matched *.> — needs two tokens")
	}
	if m := matchN("foo.bar"); !m[2] {
		t.Fatalf("foo.bar should match *.>")
	}
	// '>' alone covers everything non-empty.
	if m := matchN("x.y.z.w"); !m[3] {
		t.Fatalf("x.y.z.w should match >")
	}
	// Oracle cross-check on randoms.
	rng := bbGslRng(t)
	s2 := gsl.NewSublist[int]()
	filters := []string{"a.>", ">", "x.y.>", "*.>"}
	for i, f := range filters {
		_ = s2.Insert(f, i)
	}
	for i := 0; i < 100; i++ {
		subj := bbGslRandSubject(rng, 4)
		got := map[int]bool{}
		s2.Match(subj, func(v int) { got[v] = true })
		for j, f := range filters {
			if got[j] != bbGslOracle(f, subj) {
				t.Fatalf("Match(%q) filter %q: got=%v want %v", subj, f, got[j], bbGslOracle(f, subj))
			}
		}
	}
}

// Detail 7: all matching subscriptions at a level fire — a literal, a '*',
// and a '>' each deliver on a matching subject.
func TestDetail07_OverlappingDelivery(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("foo.bar", 1)
	_ = s.Insert("foo.*", 2)
	_ = s.Insert("foo.>", 3)
	got := map[int]bool{}
	s.Match("foo.bar", func(v int) { got[v] = true })
	if len(got) != 3 || !got[1] || !got[2] || !got[3] {
		t.Fatalf("foo.bar delivered %v want {1,2,3}", got)
	}
	got = map[int]bool{}
	s.Match("foo.baz", func(v int) { got[v] = true })
	if len(got) != 2 || !got[2] || !got[3] {
		t.Fatalf("foo.baz delivered %v want {2,3}", got)
	}
}

// Detail 8: Insert stores value->subject and the count increments per call,
// including a repeat insert of an identical (subject, value) pair.
func TestDetail08_InsertCounting(t *testing.T) {
	rng := bbGslRng(t)
	s := gsl.NewSublist[int]()
	n := 0
	for i := 0; i < 50; i++ {
		sub := bbGslRandSubject(rng, 4)
		if err := s.Insert(sub, i); err != nil {
			t.Fatalf("insert %q: %v", sub, err)
		}
		n++
		if s.Count() != uint32(n) {
			t.Fatalf("count=%d want %d", s.Count(), n)
		}
	}
	// Repeat identical pair still increments.
	s2 := gsl.NewSublist[int]()
	_ = s2.Insert("a.b", 7)
	if s2.Count() != 1 {
		t.Fatalf("count=%d want 1", s2.Count())
	}
	_ = s2.Insert("a.b", 7)
	if s2.Count() != 2 {
		t.Fatalf("count after duplicate pair=%d want 2", s2.Count())
	}
	// Count tracks duplicates, but delivery dedups identical (subject,
	// value) pairs to a single callback.
	got := 0
	s2.Match("a.b", func(v int) { got++ })
	if got != 1 {
		t.Fatalf("duplicate pair delivered %d want 1", got)
	}
	// Same subject different value also counts.
	_ = s2.Insert("a.b", 9)
	if s2.Count() != 3 {
		t.Fatalf("count=%d want 3", s2.Count())
	}
}

// Detail 9: Remove validates the subject exactly like insert and reports
// not-found for an absent path or an absent value under an existing path.
func TestDetail09_RemoveValidationAndNotFound(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("a.b.c", 1)
	_ = s.Insert("a.b.d", 2)
	// Absent path.
	if err := s.Remove("x.y.z", 1); !errors.Is(err, gsl.ErrNotFound) {
		t.Fatalf("Remove absent path: %v want ErrNotFound", err)
	}
	// Absent value under existing path.
	if err := s.Remove("a.b.c", 99); !errors.Is(err, gsl.ErrNotFound) {
		t.Fatalf("Remove absent value: %v want ErrNotFound", err)
	}
	// Wildcard path never registered.
	if err := s.Remove("a.*", 1); !errors.Is(err, gsl.ErrNotFound) {
		t.Fatalf("Remove absent wildcard: %v want ErrNotFound", err)
	}
	// Invalid subject still wins over not-found ordering.
	if err := s.Remove("a..b", 1); !errors.Is(err, gsl.ErrInvalidSubject) {
		t.Fatalf("Remove invalid: %v want ErrInvalidSubject", err)
	}
	if err := s.Remove("a.>.b", 1); !errors.Is(err, gsl.ErrInvalidSubject) {
		t.Fatalf("Remove bad fwc: %v want ErrInvalidSubject", err)
	}
	// Successful remove.
	if err := s.Remove("a.b.c", 1); err != nil {
		t.Fatalf("Remove present: %v", err)
	}
}

// Detail 10: a successful remove decrements the count and prunes empty
// nodes bottom-up — observable via Count and interest predicates.
func TestDetail10_RemovePruning(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("a.b.c.d.e", 1)
	_ = s.Insert("x.y.z", 2)
	if s.Count() != 2 {
		t.Fatalf("count=%d want 2", s.Count())
	}
	if err := s.Remove("a.b.c.d.e", 1); err != nil {
		t.Fatal(err)
	}
	if s.Count() != 1 {
		t.Fatalf("count=%d want 1", s.Count())
	}
	// Pruned: no interest anywhere along the removed path.
	for _, p := range []string{"a.b.c.d.e", "a.b.c.d", "a.b.c", "a.b", "a"} {
		if s.HasInterestStartingIn(p) {
			t.Fatalf("interest remains under %q after remove", p)
		}
	}
	// Other branch intact.
	if !s.HasInterestStartingIn("x.y") {
		t.Fatal("unrelated branch pruned incorrectly")
	}
	// Removing the last entry empties the sublist.
	if err := s.Remove("x.y.z", 2); err != nil {
		t.Fatal(err)
	}
	if s.Count() != 0 || s.HasInterestStartingIn("x") {
		t.Fatalf("sublist not empty after final remove: count=%d", s.Count())
	}
	if subj, ok := s.MatchesSingleFilter(); ok || subj != "" {
		t.Fatalf("empty sublist single filter=(%q,%v)", subj, ok)
	}
}

// Detail 11: byte-slice matching is identical to string matching.
func TestDetail11_MatchBytesEquivalence(t *testing.T) {
	rng := bbGslRng(t)
	s := gsl.NewSublist[int]()
	filters := []string{"a.b", "a.*", "a.>", "x.*.z", ">"}
	for i, f := range filters {
		_ = s.Insert(f, i+1)
	}
	for i := 0; i < 120; i++ {
		subj := bbGslRandSubject(rng, 4)
		var a, b []int
		s.Match(subj, func(v int) { a = append(a, v) })
		s.MatchBytes([]byte(subj), func(v int) { b = append(b, v) })
		if len(a) != len(b) {
			t.Fatalf("Match(%q)=%v MatchBytes=%v", subj, a, b)
		}
		setA, setB := map[int]bool{}, map[int]bool{}
		for _, v := range a {
			setA[v] = true
		}
		for _, v := range b {
			setB[v] = true
		}
		for v := range setA {
			if !setB[v] {
				t.Fatalf("Match(%q)=%v MatchBytes=%v", subj, a, b)
			}
		}
	}
}

// Detail 12: HasInterest reports at least one match (malformed -> false);
// NumInterest is a lower bound that short-circuits on first confirmed match.
func TestDetail12_InterestPredicates(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("a.b", 1)
	_ = s.Insert("a.*", 2)
	_ = s.Insert("a.>", 3)
	if !s.HasInterest("a.b") || !s.HasInterest("a.zzz") {
		t.Fatal("HasInterest false on covered subjects")
	}
	if s.HasInterest("zzz.top") {
		t.Fatal("HasInterest true on uncovered subject")
	}
	// Malformed subject -> false.
	for _, bad := range []string{".a", "a..b", "a."} {
		if s.HasInterest(bad) {
			t.Fatalf("HasInterest(%q)=true on malformed", bad)
		}
	}
	// NumInterest: >=1 when a match exists, <= actual number of matching
	// subscriptions (it may stop early).
	if n := s.NumInterest("a.b"); n < 1 || n > 3 {
		t.Fatalf("NumInterest(a.b)=%d want 1..3", n)
	}
	if n := s.NumInterest("zzz.top"); n != 0 {
		t.Fatalf("NumInterest(uncovered)=%d want 0", n)
	}
	if n := s.NumInterest("a..b"); n != 0 {
		t.Fatalf("NumInterest(malformed)=%d want 0", n)
	}
	// Single matching sub -> exact.
	s2 := gsl.NewSublist[int]()
	_ = s2.Insert("p.q", 1)
	if n := s2.NumInterest("p.q"); n != 1 {
		t.Fatalf("NumInterest single=%d want 1", n)
	}
}

// Detail 13: MatchesFullWildcard true iff a '>' subscription sits at the
// top level; nil receiver -> false.
func TestDetail13_FullWildcardPredicate(t *testing.T) {
	s := gsl.NewSublist[int]()
	if s.MatchesFullWildcard() {
		t.Fatal("empty sublist MatchesFullWildcard")
	}
	_ = s.Insert("a.>", 1)
	if s.MatchesFullWildcard() {
		t.Fatal("nested '>' is not top-level")
	}
	_ = s.Insert("*", 2)
	if s.MatchesFullWildcard() {
		t.Fatal("'*' at top is not full wildcard")
	}
	_ = s.Insert(">", 3)
	if !s.MatchesFullWildcard() {
		t.Fatal("top-level '>' should report full wildcard")
	}
	// Remove it -> false again.
	if err := s.Remove(">", 3); err != nil {
		t.Fatal(err)
	}
	if s.MatchesFullWildcard() {
		t.Fatal("after removing '>' should be false")
	}
	// Nil receiver.
	var nilS *gsl.GenericSublist[int]
	if nilS.MatchesFullWildcard() {
		t.Fatal("nil receiver MatchesFullWildcard should be false")
	}
}

// Detail 14: MatchesSingleFilter returns the subject iff exactly one
// distinct subject is held; several values on one subject still count as
// one; nil/empty -> ("", false).
func TestDetail14_SingleFilter(t *testing.T) {
	s := gsl.NewSublist[int]()
	if subj, ok := s.MatchesSingleFilter(); ok || subj != "" {
		t.Fatalf("empty: (%q,%v)", subj, ok)
	}
	var nilS *gsl.GenericSublist[int]
	if subj, ok := nilS.MatchesSingleFilter(); ok || subj != "" {
		t.Fatalf("nil: (%q,%v)", subj, ok)
	}
	_ = s.Insert("a.b.c", 1)
	if subj, ok := s.MatchesSingleFilter(); !ok || subj != "a.b.c" {
		t.Fatalf("single: (%q,%v) want (a.b.c,true)", subj, ok)
	}
	// Several values, same subject -> still one filter.
	_ = s.Insert("a.b.c", 2)
	_ = s.Insert("a.b.c", 3)
	if subj, ok := s.MatchesSingleFilter(); !ok || subj != "a.b.c" {
		t.Fatalf("multi-value single subject: (%q,%v)", subj, ok)
	}
	// Second branch anywhere -> false.
	_ = s.Insert("a.b.d", 4)
	if _, ok := s.MatchesSingleFilter(); ok {
		t.Fatal("two subjects should not be single filter")
	}
	_ = s.Remove("a.b.d", 4)
	if subj, ok := s.MatchesSingleFilter(); !ok || subj != "a.b.c" {
		t.Fatalf("after prune: (%q,%v) want (a.b.c,true)", subj, ok)
	}
	// Wildcard subject is a valid single filter.
	s2 := gsl.NewSublist[int]()
	_ = s2.Insert("a.*.c", 9)
	if subj, ok := s2.MatchesSingleFilter(); !ok || subj != "a.*.c" {
		t.Fatalf("wildcard single: (%q,%v)", subj, ok)
	}
}

// Detail 15: HasInterestStartingIn — true when any subscription's token
// path begins with the prefix; '>' anywhere along the path makes it true;
// a prefix strictly shorter than every entry is true.
func TestDetail15_PrefixInterest(t *testing.T) {
	s := gsl.NewSublist[int]()
	_ = s.Insert("a.b.c", 1)
	_ = s.Insert("x.y.>", 2)
	// Shorter prefixes are true.
	for _, p := range []string{"a", "a.b", "a.b.c", "x", "x.y", "x.y.z"} {
		if !s.HasInterestStartingIn(p) {
			t.Fatalf("HasInterestStartingIn(%q)=false want true", p)
		}
	}
	// '>' along the path -> deeper prefixes still true.
	if !s.HasInterestStartingIn("x.y.z.w.v") {
		t.Fatal("'>' should satisfy deeper prefixes")
	}
	// Unrelated prefixes false.
	for _, p := range []string{"b", "a.c", "a.b.c.d.e", "x.x"} {
		if s.HasInterestStartingIn(p) {
			t.Fatalf("HasInterestStartingIn(%q)=true want false", p)
		}
	}
	// a.b.c.d.e is false here? 'a.b.c' is a literal path, no '>'.
	// Recheck: prefix longer than the entry is only true via '>'.
	rng := bbGslRng(t)
	s2 := gsl.NewSublist[int]()
	var subs []string
	for i := 0; i < 30; i++ {
		sub := bbGslRandSubject(rng, 3)
		if rng.Intn(3) == 0 {
			sub += ".>"
		}
		subs = append(subs, sub)
		_ = s2.Insert(sub, i)
	}
	// Reference: prefix p is interested iff some subscription's token path
	// has p as a prefix OR a '>'-terminated path covers it.
	oracle := func(p string) bool {
		pt := strings.Split(p, ".")
		for _, sub := range subs {
			st := strings.Split(sub, ".")
			// Walk tokens; '>' in sub covers any remaining prefix.
			ok := true
			for i, tok := range pt {
				if i >= len(st) {
					ok = false // prefix deeper than this path
					break
				}
				if st[i] == ">" {
					break // covered by full wildcard
				}
				if st[i] == "*" {
					continue // '*' covers this token position
				}
				if st[i] != tok {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		return false
	}
	for i := 0; i < 150; i++ {
		p := bbGslRandSubject(rng, 5)
		if got := s2.HasInterestStartingIn(p); got != oracle(p) {
			t.Fatalf("HasInterestStartingIn(%q)=%v oracle=%v subs=%v", p, got, oracle(p), subs)
		}
	}
}
