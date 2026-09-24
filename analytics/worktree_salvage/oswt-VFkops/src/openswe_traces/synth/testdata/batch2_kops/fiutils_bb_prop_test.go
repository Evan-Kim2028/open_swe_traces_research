package utils_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	fiutils "example.internal/kops/upup/pkg/fi/utils"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func randStr(rng *rand.Rand, n int, alphabet string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rng.Intn(len(alphabet))]
	}
	return string(b)
}

// Detail 1: StringSlicesEqual is length + elementwise order.
func TestDetail01_StringSlicesEqualOrdered(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		n := rng.Intn(8)
		l := make([]string, n)
		for j := range l {
			l[j] = randStr(rng, 1+rng.Intn(6), "abc")
		}
		r := append([]string{}, l...)
		if !fiutils.StringSlicesEqual(l, r) {
			t.Fatalf("i=%d identical slices unequal: %v %v", i, l, r)
		}
		if n >= 2 {
			// Swap two elements: order must matter.
			r[0], r[1] = r[1], r[0]
			if l[0] != l[1] && fiutils.StringSlicesEqual(l, r) {
				t.Fatalf("i=%d reordered slices equal: %v %v", i, l, r)
			}
		}
		if n > 0 {
			short := l[:n-1]
			if fiutils.StringSlicesEqual(l, short) {
				t.Fatalf("i=%d different lengths equal", i)
			}
		}
	}
}

// Detail 2: StringSlicesEqualIgnoreOrder — lengths must match, then every
// right element merely has to appear in the left (NOT a multiset count):
// ["a","a","b"] equals ["a","b","b"].
func TestDetail02_IgnoreOrderIsMembershipNotMultiset(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	// The dup-multiplicity corner.
	if !fiutils.StringSlicesEqualIgnoreOrder([]string{"a", "a", "b"}, []string{"a", "b", "b"}) {
		t.Fatalf("dup-multiplicity case not equal")
	}
	if fiutils.StringSlicesEqualIgnoreOrder([]string{"a", "a"}, []string{"a", "b"}) {
		t.Fatalf("distinct-element case wrongly equal")
	}
	for i := 0; i < 400; i++ {
		n := 1 + rng.Intn(6)
		l := make([]string, n)
		for j := range l {
			l[j] = randStr(rng, 1+rng.Intn(3), "ab")
		}
		r := append([]string{}, l...)
		rng.Shuffle(n, func(a, b int) { r[a], r[b] = r[b], r[a] })
		if !fiutils.StringSlicesEqualIgnoreOrder(l, r) {
			t.Fatalf("i=%d permutation unequal: %v %v", i, l, r)
		}
		// Same length but an element not present on the left must fail —
		// unless dup-multiplicity semantics mask it. Build a right side with
		// a guaranteed-foreign element.
		r2 := append([]string{}, l...)
		r2[rng.Intn(n)] = "ZZZ" + strconv.Itoa(i)
		if fiutils.StringSlicesEqualIgnoreOrder(l, r2) {
			t.Fatalf("i=%d foreign element tolerated: %v %v", i, l, r2)
		}
		if n > 1 && fiutils.StringSlicesEqualIgnoreOrder(l, l[:n-1]) {
			t.Fatalf("i=%d different lengths equal", i)
		}
	}
}

// Detail 3: nil/empty equivalence in both compares.
func TestDetail03_NilEmptyEquivalence(t *testing.T) {
	if !fiutils.StringSlicesEqual(nil, []string{}) {
		t.Fatalf("StringSlicesEqual nil vs empty")
	}
	if !fiutils.StringSlicesEqual(nil, nil) {
		t.Fatalf("StringSlicesEqual nil vs nil")
	}
	if !fiutils.StringSlicesEqualIgnoreOrder(nil, []string{}) {
		t.Fatalf("IgnoreOrder nil vs empty")
	}
	if fiutils.StringSlicesEqual(nil, []string{""}) {
		t.Fatalf("nil vs [\"\"]")
	}
	if fiutils.StringSlicesEqualIgnoreOrder([]string{}, []string{""}) {
		t.Fatalf("empty vs [\"\"]")
	}
}

// Detail 4: SanitizeString substitutes each non-[a-zA-Z0-9_-] char with "_"
// — output length == input length (not removed).
func TestDetail04_SanitizeSubstitutesNotRemoves(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	const alphabet = "ab09_- !@#$%^&*()+=;:'\"/?.>"
	for i := 0; i < 500; i++ {
		n := 1 + rng.Intn(50)
		s := randStr(rng, n, alphabet)
		got := fiutils.SanitizeString(s)
		wantLen := n
		if n > 200 {
			wantLen = 200
		}
		if len(got) != wantLen {
			t.Fatalf("i=%d SanitizeString(%q) len %d want %d", i, s, len(got), wantLen)
		}
		if n <= 200 {
			for j, c := range got {
				orig := s[j]
				ok := (orig >= 'a' && orig <= 'z') || (orig >= 'A' && orig <= 'Z') ||
					(orig >= '0' && orig <= '9') || orig == '_' || orig == '-'
				if ok {
					if byte(c) != orig {
						t.Fatalf("i=%d SanitizeString(%q)[%d]=%q want %q", i, s, j, c, orig)
					}
				} else if c != '_' {
					t.Fatalf("i=%d SanitizeString(%q)[%d]=%q want '_'", i, s, j, c)
				}
			}
		}
	}
}

// Detail 5: truncation keeps the LAST 200 chars.
func TestDetail05_SanitizeKeepsLast200(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		head := randStr(rng, 150+rng.Intn(200), "abcXYZ019 _-")
		tail := "ZZZ" + randStr(rng, 190, "ab")
		s := head + tail
		got := fiutils.SanitizeString(s)
		if len(got) != 200 {
			t.Fatalf("i=%d len %d want 200", i, len(got))
		}
		// Expected: substitute-then-take-last-200 — the tail's sanitized form
		// must end the output.
		last := s[len(s)-200:]
		var exp strings.Builder
		for j := 0; j < len(last); j++ {
			c := last[j]
			ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '_' || c == '-'
			if ok {
				exp.WriteByte(c)
			} else {
				exp.WriteByte('_')
			}
		}
		if got != exp.String() {
			t.Fatalf("i=%d got %q want last-200 %q", i, got, exp.String())
		}
	}
	// Deterministic check: first-200 truncation would produce a different head.
	s := strings.Repeat("q", 300) + "TAIL"
	got := fiutils.SanitizeString(s)
	if !strings.HasSuffix(got, "TAIL") {
		t.Fatalf("tail lost: %q", got[len(got)-16:])
	}
}

// Detail 6: only a leading "~/" is replaced by $HOME; bare ~, ~user and
// mid-string ~/ are untouched.
func TestDetail06_ExpandPathLeadingTildeSlashOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got := fiutils.ExpandPath("~/x"); got != filepath.Join(home, "x") {
		t.Fatalf("ExpandPath(~/x)=%q want %q", got, filepath.Join(home, "x"))
	}
	if got := fiutils.ExpandPath("~"); got != "~" {
		t.Fatalf("ExpandPath(~)=%q", got)
	}
	if got := fiutils.ExpandPath("~user/x"); got != "~user/x" {
		t.Fatalf("ExpandPath(~user/x)=%q", got)
	}
	if got := fiutils.ExpandPath("/a/~/b"); got != "/a/~/b" {
		t.Fatalf("ExpandPath(/a/~/b)=%q", got)
	}
	if got := fiutils.ExpandPath("plain"); got != "plain" {
		t.Fatalf("ExpandPath(plain)=%q", got)
	}
	if got := fiutils.ExpandPath(""); got != "" {
		t.Fatalf("ExpandPath(\"\")=%q", got)
	}
}

// Detail 7: HashString is lowercase-hex SHA-256, never errors.
func TestDetail07_HashStringLowerHexSHA256(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		s := randStr(rng, rng.Intn(200), "abcXYZ019 \n\t!@#")
		got, err := fiutils.HashString(s)
		if err != nil {
			t.Fatalf("i=%d HashString error %v", i, err)
		}
		sum := sha256.Sum256([]byte(s))
		want := hex.EncodeToString(sum[:])
		if got != want {
			t.Fatalf("i=%d HashString(%q)=%q want %q", i, s, got, want)
		}
		if got != strings.ToLower(got) {
			t.Fatalf("i=%d hash not lowercase", i)
		}
	}
}
