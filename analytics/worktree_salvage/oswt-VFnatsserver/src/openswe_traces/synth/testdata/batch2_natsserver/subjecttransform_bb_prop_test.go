// Hidden black-box property suite for the subjecttransform unit.
// Drives only the API in api.md: NewSubjectTransform{,Strict,WithStrict},
// Match/TransformSubject/TransformTokenizedSubject, the same-package
// helpers, transform-kind enum consts, and the documented error values.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

const bbStxHiddenSeed = 20260919

func bbStxSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbStxHiddenSeed
}

func bbStxRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbStxSeed()))
}

// The package wraps destination errors in a struct that does not implement
// Unwrap, so errors.Is cannot reach the sentinels; the sentinel's message
// text is part of the produced error string, so match on that.
func bbStErrIs(err, sentinel error) bool {
	return err != nil && sentinel != nil && strings.Contains(err.Error(), sentinel.Error())
}

// Detail 1: empty dest -> (nil,nil); empty src -> treated as '>'.
func TestDetail01_EmptyEndpoints(t *testing.T) {
	tr, err := NewSubjectTransform("foo.*", "")
	if tr != nil || err != nil {
		t.Fatalf("empty dest: tr=%v err=%v want nil,nil", tr, err)
	}
	tr, err = NewSubjectTransform("", "x.>")
	if err != nil || tr == nil {
		t.Fatalf("empty src: tr=%v err=%v", tr, err)
	}
	// Empty src acts as '>' — covers any subject.
	got, err := tr.Match("any.subject.here")
	if err != nil {
		t.Fatalf("empty-src Match: %v", err)
	}
	if got == "" {
		t.Fatal("empty-src transform produced empty")
	}
	tr, err = NewSubjectTransform("", "")
	if tr != nil || err != nil {
		t.Fatalf("both empty: tr=%v err=%v want nil,nil", tr, err)
	}
}

// Detail 2: both ends must be valid subjects; dest must not contain '*';
// src/dest must agree on terminal '>'.
func TestDetail02_SubjectValidityRules(t *testing.T) {
	for _, sd := range [][2]string{
		{"a..b", "x"},
		{"a.", "x"},
		{".a", "x"},
		{"a.>.b", "x"},
		{"a", "x..y"},
		{"a.*", "x.*"},   // dest may not contain '*'
		{"a.>", "x.y"},   // src '>' but dest not
		{"a.b", "x.>"},   // dest '>' but src not
		{"a.>.>", "x.>"}, // non-terminal '>'
	} {
		tr, err := NewSubjectTransform(sd[0], sd[1])
		if err == nil {
			t.Fatalf("NewSubjectTransform(%q,%q)=%v want error", sd[0], sd[1], tr)
		}
	}
	// Agreeing '>' endpoints are fine.
	tr, err := NewSubjectTransform("a.>", "x.>")
	if err != nil || tr == nil {
		t.Fatalf("a.> -> x.>: %v", err)
	}
	tr, err = NewSubjectTransform("a.*", "x.$1")
	if err != nil || tr == nil {
		t.Fatalf("a.* -> x.$1: %v", err)
	}
}

// Detail 3: '$'+digits -> positional placeholder (1-based over source '*'s);
// '$'+non-digit -> literal NoTransform token, not an error.
func TestDetail03_DollarPlaceholders(t *testing.T) {
	tr, err := NewSubjectTransform("a.*.*", "x.$2.$1")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("a.P.Q")
	if err != nil || got != "x.Q.P" {
		t.Fatalf("Match=%q err=%v want x.Q.P", got, err)
	}
	// $foo is a literal token.
	tr, err = NewSubjectTransform("a.*", "x.$foo.$1")
	if err != nil {
		t.Fatalf("$foo literal should not error: %v", err)
	}
	got, err = tr.Match("a.Z")
	if err != nil || got != "x.$foo.Z" {
		t.Fatalf("Match=%q err=%v want x.$foo.Z", got, err)
	}
	// $0 is not a valid placeholder index (1-based); it is literal-ish or an
	// error — assert behavior is deterministic per API: indexPlaceHolders
	// classifies it.
	tt, idxs, intarg, strarg, err := indexPlaceHolders("$1")
	if err != nil || tt != Wildcard || len(idxs) != 1 || idxs[0] != 1 {
		t.Fatalf("indexPlaceHolders($1)=(%d,%v,%d,%q,%v)", tt, idxs, intarg, strarg, err)
	}
	tt, _, _, _, _ = indexPlaceHolders("$foo")
	if tt != NoTransform {
		t.Fatalf("indexPlaceHolders($foo) type=%d want NoTransform", tt)
	}
}

// Detail 4: {{wildcard(n)}} arity — ()->NotEnoughArgs, >1->TooManyArgs,
// non-int->InvalidArg, index>npwcs->IndexOutOfRange.
func TestDetail04_WildcardArity(t *testing.T) {
	cases := []struct {
		dest string
		want error
	}{
		{"x.{{wildcard()}}", ErrMappingDestinationNotEnoughArgs},
		{"x.{{wildcard(1,2)}}", ErrMappingDestinationTooManyArgs},
		{"x.{{wildcard(abc)}}", ErrMappingDestinationInvalidArg},
		{"x.{{wildcard(3)}}", ErrMappingDestinationIndexOutOfRange}, // src has 2 wildcards
	}
	for _, c := range cases {
		_, err := NewSubjectTransform("a.*.*", c.dest)
		if !bbStErrIs(err, c.want) {
			t.Fatalf("dest %q: err=%v want %v", c.dest, err, c.want)
		}
	}
	// Valid arities work.
	tr, err := NewSubjectTransform("a.*.*", "x.{{wildcard(2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("a.P.Q")
	if err != nil || got != "x.Q" {
		t.Fatalf("Match=%q want x.Q", got)
	}
}

// Detail 5: {{partition(n)}} hashes the whole subject;
// {{partition(n,i,...)}} hashes concatenated chosen tokens through the
// wildcard-index map (index 0 -> source token 0).
func TestDetail05_PartitionKeying(t *testing.T) {
	rng := bbStxRng(t)
	n := 1 + rng.Intn(10)
	// Whole-subject hash.
	tr, err := NewSubjectTransform("a.*", "p.{{partition("+strconv.Itoa(n)+")}}")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		subj := "a.tok" + strconv.Itoa(rng.Intn(1000))
		got, err := tr.Match(subj)
		if err != nil {
			t.Fatal(err)
		}
		h := fnv.New32a()
		h.Write([]byte(subj))
		want := "p." + strconv.FormatUint(uint64(h.Sum32()%uint32(n)), 10)
		if got != want {
			t.Fatalf("partition(%s)=%q want %q", subj, got, want)
		}
	}
	// Token-index partition: partition(n,1) hashes source token index 1.
	tr, err = NewSubjectTransform("a.*.b", "p.{{partition("+strconv.Itoa(n)+",1)}}")
	if err != nil {
		t.Fatal(err)
	}
	subj := "a.WILD.b"
	got, err := tr.Match(subj)
	if err != nil {
		t.Fatal(err)
	}
	h := fnv.New32a()
	h.Write([]byte("WILD"))
	want := "p." + strconv.FormatUint(uint64(h.Sum32()%uint32(n)), 10)
	if got != want {
		t.Fatalf("token partition=%q want %q", got, want)
	}
	// Index 0 -> source token 0 (literal 'a'), so all subjects hash identically.
	tr, err = NewSubjectTransform("a.*.b", "p.{{partition("+strconv.Itoa(n)+",0)}}")
	if err != nil {
		t.Fatal(err)
	}
	g1, err := tr.Match("a.X.b")
	if err != nil {
		t.Fatal(err)
	}
	g2, err := tr.Match("a.Y.b")
	if err != nil {
		t.Fatal(err)
	}
	if g1 != g2 {
		t.Fatalf("index-0 partition should be constant: %q vs %q", g1, g2)
	}
	h = fnv.New32a()
	h.Write([]byte("a"))
	want = "p." + strconv.FormatUint(uint64(h.Sum32()%uint32(n)), 10)
	if g1 != want {
		t.Fatalf("index-0 partition=%q want %q", g1, want)
	}
	// Concatenation of chosen tokens (wildcard indexes are '*' positions).
	tr, err = NewSubjectTransform("a.*.b.*", "p.{{partition("+strconv.Itoa(n)+",1,2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.X.b.Y")
	if err != nil {
		t.Fatal(err)
	}
	h = fnv.New32a()
	h.Write([]byte("XY"))
	want = "p." + strconv.FormatUint(uint64(h.Sum32()%uint32(n)), 10)
	if got != want {
		t.Fatalf("concat partition=%q want %q", got, want)
	}
}

// Detail 6: partition/random int arg must be <= math.MaxInt32 (== allowed,
// > rejected as InvalidArg).
func TestDetail06_Int32ArgBounds(t *testing.T) {
	max := strconv.FormatInt(math.MaxInt32, 10)
	over := strconv.FormatInt(math.MaxInt32+1, 10)
	if _, err := NewSubjectTransform("a.*", "p.{{partition("+max+")}}"); err != nil {
		t.Fatalf("partition(MaxInt32): %v", err)
	}
	_, err := NewSubjectTransform("a.*", "p.{{partition("+over+")}}")
	if !bbStErrIs(err, ErrMappingDestinationInvalidArg) {
		t.Fatalf("partition(MaxInt32+1): %v want InvalidArg", err)
	}
	if _, err := NewSubjectTransform("a.*", "p.{{random("+max+")}}"); err != nil {
		t.Fatalf("random(MaxInt32): %v", err)
	}
	_, err = NewSubjectTransform("a.*", "p.{{random("+over+")}}")
	if !bbStErrIs(err, ErrMappingDestinationInvalidArg) {
		t.Fatalf("random(MaxInt32+1): %v want InvalidArg", err)
	}
}

// Detail 7: the six two-arg functions require exactly 2 int args via the
// shared helper.
func TestDetail07_TwoArgFunctionArity(t *testing.T) {
	fns := []string{"splitfromleft", "splitfromright", "slicefromleft", "slicefromright", "left", "right"}
	for _, f := range fns {
		_, err := NewSubjectTransform("a.*", "x.{{"+f+"(1)}}")
		if !bbStErrIs(err, ErrMappingDestinationNotEnoughArgs) {
			t.Fatalf("%s(1): %v want NotEnoughArgs", f, err)
		}
		_, err = NewSubjectTransform("a.*", "x.{{"+f+"(1,2,3)}}")
		if !bbStErrIs(err, ErrMappingDestinationTooManyArgs) {
			t.Fatalf("%s(1,2,3): %v want TooManyArgs", f, err)
		}
		_, err = NewSubjectTransform("a.*", "x.{{"+f+"(1,zz)}}")
		if !bbStErrIs(err, ErrMappingDestinationInvalidArg) {
			t.Fatalf("%s(1,zz): %v want InvalidArg", f, err)
		}
		_, err = NewSubjectTransform("a.*", "x.{{"+f+"(zz,2)}}")
		if !bbStErrIs(err, ErrMappingDestinationInvalidArg) {
			t.Fatalf("%s(zz,2): %v want InvalidArg", f, err)
		}
		// Valid construction works.
		if _, err := NewSubjectTransform("a.*", "x.{{"+f+"(1,2)}}"); err != nil {
			t.Fatalf("%s(1,2): %v", f, err)
		}
	}
}

// Detail 8: {{split(i,delim)}} — exactly 2 args; delim may not contain
// space or the subject separator.
func TestDetail08_SplitDelimRules(t *testing.T) {
	// 2 args required.
	_, err := NewSubjectTransform("a.*", "x.{{split(1)}}")
	if !bbStErrIs(err, ErrMappingDestinationNotEnoughArgs) {
		t.Fatalf("split(1): %v want NotEnoughArgs", err)
	}
	_, err = NewSubjectTransform("a.*", "x.{{split(1,_,x)}}")
	if !bbStErrIs(err, ErrMappingDestinationTooManyArgs) {
		t.Fatalf("split(1,_,x): %v want TooManyArgs", err)
	}
	// Delim containing a space rejected (args are space-trimmed, so an
	// embedded space is the only way to land one inside the delimiter).
	if _, err := NewSubjectTransform("a.*", "x.{{split(1,a b)}}"); !bbStErrIs(err, ErrMappingDestinationInvalidArg) {
		t.Fatalf("split delim with space: %v want InvalidArg", err)
	}
	// '.' as delim doesn't register as a function — the mustache token is
	// emitted verbatim like any unrecognized literal.
	trDot, err := NewSubjectTransform("a.*", "x.{{split(1,.)}}")
	if err != nil {
		t.Fatalf("split delim '.': %v", err)
	}
	if got, err := trDot.Match("a.pq"); err != nil || got != "x.{{split(1,.)}}" {
		t.Fatalf("split delim '.' match=%q err=%v want verbatim x.{{split(1,.)}}", got, err)
	}
	// Valid split: drops empty pieces.
	tr, err := NewSubjectTransform("a.*", "x.{{split(1,_)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("a.p_q_r")
	if err != nil || got != "x.p.q.r" {
		t.Fatalf("split=%q err=%v want x.p.q.r", got, err)
	}
}

// Detail 9: {{…}}-shaped tokens matching no known function ->
// ErrUnknownMappingDestinationFunction; tokens not ending '}}' are literals.
func TestDetail09_UnknownFunctionVsLiteral(t *testing.T) {
	for _, dest := range []string{
		"x.{{unimplemented(1)}}",
		"x.{{wildcard5)}}}",
		"x.{{nosuch()}}",
		"x.{{PARTITIONX(1)}}",
	} {
		_, err := NewSubjectTransform("a.*", dest)
		if !bbStErrIs(err, ErrUnknownMappingDestinationFunction) {
			t.Fatalf("dest %q: %v want ErrUnknownMappingDestinationFunction", dest, err)
		}
	}
	// Tokens not ending in '}}' are literals — no error, verbatim emission.
	for _, dest := range []string{"x.{{splitLeft(2,2}", "x.{{wildcard(1)", "x.{{", "x.}"} {
		tr, err := NewSubjectTransform("a.*", dest)
		if err != nil {
			t.Fatalf("literal dest %q should not error: %v", dest, err)
		}
		got, err := tr.Match("a.z")
		if err != nil {
			t.Fatalf("Match on literal dest %q: %v", dest, err)
		}
		if !strings.Contains(got, strings.TrimPrefix(dest, "x.")) {
			t.Fatalf("literal dest %q produced %q — token not verbatim", dest, got)
		}
	}
}

// Detail 10: no-'*' source — dest may use only literals/partition/random;
// other fns -> IndexOutOfRange.
func TestDetail10_NoWildcardSource(t *testing.T) {
	for _, dest := range []string{
		"x.{{wildcard(1)}}",
		"x.{{splitfromleft(1,2)}}",
		"x.{{left(1,2)}}",
		"x.{{slicefromright(1,2)}}",
		"x.$1",
	} {
		_, err := NewSubjectTransform("foo", dest)
		if !bbStErrIs(err, ErrMappingDestinationIndexOutOfRange) {
			t.Fatalf("no-* src dest %q: %v want IndexOutOfRange", dest, err)
		}
	}
	// partition/random/literals are fine.
	if _, err := NewSubjectTransform("foo", "x.{{partition(4)}}"); err != nil {
		t.Fatalf("partition no-*: %v", err)
	}
	if _, err := NewSubjectTransform("foo", "x.{{random(4)}}"); err != nil {
		t.Fatalf("random no-*: %v", err)
	}
	if _, err := NewSubjectTransform("foo", "x.literal"); err != nil {
		t.Fatalf("literal no-*: %v", err)
	}
}

// Detail 11: strict mode — only Wildcard allowed (incl $N); all source '*'
// must be placed -> NotSupportedForImport / NotUsingAllWildcards.
func TestDetail11_StrictMode(t *testing.T) {
	// Non-wildcard functions rejected in strict.
	for _, dest := range []string{
		"x.{{partition(4)}}",
		"x.{{random(4)}}",
		"x.{{left(1,2)}}",
		"x.{{splitfromleft(1,2)}}",
	} {
		_, err := NewSubjectTransformStrict("a.*", dest)
		if !bbStErrIs(err, ErrMappingDestinationNotSupportedForImport) {
			t.Fatalf("strict dest %q: %v want NotSupportedForImport", dest, err)
		}
	}
	// Wildcard functions OK.
	if _, err := NewSubjectTransformStrict("a.*", "x.{{wildcard(1)}}"); err != nil {
		t.Fatalf("strict wildcard: %v", err)
	}
	if _, err := NewSubjectTransformStrict("a.*", "x.$1"); err != nil {
		t.Fatalf("strict $1: %v", err)
	}
	// Unused source '*' -> NotUsingAllWildcards.
	_, err := NewSubjectTransformStrict("a.*.*", "x.$1")
	if !bbStErrIs(err, ErrMappingDestinationNotUsingAllWildcards) {
		t.Fatalf("strict unused wildcard: %v want NotUsingAllWildcards", err)
	}
	// All placed -> OK.
	if _, err := NewSubjectTransformStrict("a.*.*", "x.$2.$1"); err != nil {
		t.Fatalf("strict all-placed: %v", err)
	}
	// NewSubjectTransformWithStrict(false) == non-strict.
	if _, err := NewSubjectTransformWithStrict("a.*", "x.{{partition(4)}}", false); err != nil {
		t.Fatalf("WithStrict(false) partition: %v", err)
	}
}

// Detail 12: Match — ('>'|empty src)&('>'|empty dest) -> passthrough;
// invalid literal -> ErrBadSubject; non-covered -> ErrNoTransforms.
func TestDetail12_MatchContract(t *testing.T) {
	tr, err := NewSubjectTransform(">", ">")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("any.thing.at.all")
	if err != nil || got != "any.thing.at.all" {
		t.Fatalf("passthrough=%q err=%v", got, err)
	}
	tr, err = NewSubjectTransform("", ">")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.b.c")
	if err != nil || got != "a.b.c" {
		t.Fatalf("empty-src passthrough=%q err=%v", got, err)
	}
	// Invalid literal -> ErrBadSubject.
	tr, _ = NewSubjectTransform("a.*", "x.$1")
	if _, err := tr.Match("a..b"); !bbStErrIs(err, ErrBadSubject) {
		t.Fatalf("invalid literal: %v want ErrBadSubject", err)
	}
	// Non-covered -> ErrNoTransforms.
	if _, err := tr.Match("zzz.top"); !bbStErrIs(err, ErrNoTransforms) {
		t.Fatalf("non-covered: %v want ErrNoTransforms", err)
	}
}

// Detail 13: '>' dest token ends the pattern then appends all remaining
// source tokens.
func TestDetail13_DestFullWildcardTail(t *testing.T) {
	tr, err := NewSubjectTransform("baz.>", "my.pre.>")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("baz.a.b")
	if err != nil || got != "my.pre.a.b" {
		t.Fatalf("Match=%q err=%v want my.pre.a.b", got, err)
	}
	got, err = tr.Match("baz.x")
	if err != nil || got != "my.pre.x" {
		t.Fatalf("Match=%q err=%v want my.pre.x", got, err)
	}
	// '>' after wildcard placeholders appends unmapped remainder.
	tr, err = NewSubjectTransform("s.*.>", "d.$1.>")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("s.P.q.r")
	if err != nil || got != "d.P.q.r" {
		t.Fatalf("Match=%q err=%v want d.P.q.r", got, err)
	}
}

// Detail 14: missing source token for a placeholder -> empty emission for
// the slot but separators still emitted. ('$N' beyond the '*' count is
// rejected at construction and a short subject fails Match, so the empty
// slot is exercised through TransformTokenizedSubject with a short list.)
func TestDetail14_MissingTokenSlot(t *testing.T) {
	tr, err := NewSubjectTransform("one.*.*", "one.$1.$2")
	if err != nil {
		t.Fatal(err)
	}
	got := tr.TransformTokenizedSubject([]string{"one", "two"})
	if got != "one.two." {
		t.Fatalf("TransformTokenizedSubject=%q want \"one.two.\" (empty slot, separator emitted)", got)
	}
}

// Detail 15: per-function guards — split positions / slice sizes outside
// (0,len) -> token verbatim; slicefromleft flushes short tail as own
// piece; slicefromright emits remainder first; split drops empty pieces
// incl. leading/trailing.
func TestDetail15_FunctionGuards(t *testing.T) {
	// splitfromleft position out of range -> verbatim token.
	tr, err := NewSubjectTransform("a.*", "x.{{splitfromleft(1,9)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.Match("a.p_q_r")
	if err != nil || got != "x.p_q_r" {
		t.Fatalf("splitfromleft(1,9)=%q want verbatim x.p_q_r", got)
	}
	// split emits two tokens: the piece before and after the position.
	tr, err = NewSubjectTransform("a.*", "x.{{splitfromleft(1,2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.p_q_r")
	if err != nil || got != "x.p_.q_r" {
		t.Fatalf("splitfromleft(1,2)=%q want x.p_.q_r", got)
	}
	tr, err = NewSubjectTransform("a.*", "x.{{splitfromright(1,1)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.p_q_r")
	if err != nil || got != "x.p_q_.r" {
		t.Fatalf("splitfromright(1,1)=%q want x.p_q_.r", got)
	}
	// slicefromleft: short tail is its own piece.
	tr, err = NewSubjectTransform("a.*", "x.{{slicefromleft(1,2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.aabbc")
	if err != nil || got != "x.aa.bb.c" {
		t.Fatalf("slicefromleft=%q want x.aa.bb.c", got)
	}
	// slicefromright: remainder emitted first.
	tr, err = NewSubjectTransform("a.*", "x.{{slicefromright(1,2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.aabbc")
	if err != nil || got != "x.a.ab.bc" {
		t.Fatalf("slicefromright=%q want x.a.ab.bc", got)
	}
	// slice size out of (0,len) -> verbatim.
	tr, err = NewSubjectTransform("a.*", "x.{{slicefromleft(1,9)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.abc")
	if err != nil || got != "x.abc" {
		t.Fatalf("slicefromleft(1,9)=%q want verbatim x.abc", got)
	}
	// split drops empty pieces incl. leading/trailing.
	tr, err = NewSubjectTransform("a.*", "x.{{split(1,_)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a._p__q_")
	if err != nil || got != "x.p.q" {
		t.Fatalf("split empties=%q want x.p.q", got)
	}
	// left/right take n chars.
	tr, err = NewSubjectTransform("a.*", "x.{{left(1,3)}}.{{right(1,2)}}")
	if err != nil {
		t.Fatal(err)
	}
	got, err = tr.Match("a.abcdef")
	if err != nil || got != "x.abc.ef" {
		t.Fatalf("left/right=%q want x.abc.ef", got)
	}
}

// Detail 16: partition = FNV-1a-32(key) mod n; random = rand mod n; n=0 ->
// "0" for both.
func TestDetail16_PartitionAndRandomMath(t *testing.T) {
	tr, err := NewSubjectTransform("a.*", "x.{{partition(10)}}")
	if err != nil {
		t.Fatal(err)
	}
	hashPart := tr.getHashPartition
	randPart := tr.getRandomPartition
	// Verify fnv math against stdlib.
	h := fnv.New32a()
	h.Write([]byte("some key"))
	want := strconv.FormatUint(uint64(h.Sum32()%7), 10)
	if got := hashPart([]byte("some key"), 7); got != want {
		t.Fatalf("getHashPartition=%q want %q", got, want)
	}
	// n=0 -> "0" for both.
	if got := hashPart([]byte("anything"), 0); got != "0" {
		t.Fatalf("partition(0)=%q want \"0\"", got)
	}
	if got := randPart(0); got != "0" {
		t.Fatalf("random(0)=%q want \"0\"", got)
	}
	// random(n) in [0,n).
	for i := 0; i < 50; i++ {
		got := randPart(6)
		v, err := strconv.Atoi(got)
		if err != nil || v < 0 || v >= 6 {
			t.Fatalf("random(6)=%q out of range", got)
		}
	}
}

// Detail 17: transformTokenize numbers '*' as $1..; transformUntokenize
// maps '$'+digit-leading or wildcard(n) tokens to '*' and collects
// placeholders in order; reverse() builds the strict inverse.
func TestDetail17_TokenizeHelpers(t *testing.T) {
	if got := transformTokenize("foo.*.*"); got != "foo.$1.$2" {
		t.Fatalf("transformTokenize=%q want foo.$1.$2", got)
	}
	if got := transformTokenize("*.a.*"); got != "$1.a.$2" {
		t.Fatalf("transformTokenize=%q want $1.a.$2", got)
	}
	if got := transformTokenize("foo.bar"); got != "foo.bar" {
		t.Fatalf("transformTokenize literal=%q", got)
	}
	sub, phs := transformUntokenize("foo.$2.$1")
	if sub != "foo.*.*" || len(phs) != 2 || phs[0] != "$2" || phs[1] != "$1" {
		t.Fatalf("transformUntokenize(foo.$2.$1)=(%q,%v)", sub, phs)
	}
	sub, phs = transformUntokenize("bar")
	if sub != "bar" || len(phs) != 0 {
		t.Fatalf("transformUntokenize(bar)=(%q,%v)", sub, phs)
	}
	sub, phs = transformUntokenize("foo.{{wildcard(2)}}.{{wildcard(1)}}")
	if sub != "foo.*.*" || len(phs) != 2 {
		t.Fatalf("transformUntokenize wildcards=(%q,%v)", sub, phs)
	}
	// reverse() builds the strict inverse.
	tr, err := NewSubjectTransformStrict("a.*", "x.$1")
	if err != nil {
		t.Fatal(err)
	}
	revOf := tr.reverse
	rev := revOf()
	got, err := rev.Match("x.Q")
	if err != nil || got != "a.Q" {
		t.Fatalf("reverse Match=%q err=%v want a.Q", got, err)
	}
	// Double reverse ~ identity on matched subjects.
	revOf2 := rev.reverse
	fwd, err := revOf2().Match("a.Z")
	if err != nil || fwd != "x.Z" {
		t.Fatalf("double-reverse=%q err=%v want x.Z", fwd, err)
	}
}
