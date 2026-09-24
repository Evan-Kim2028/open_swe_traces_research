package hashing

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func bbHex(b []byte) string { return hex.EncodeToString(b) }

// TestDetail01: Hash.String is "<algorithm>:<lowercase hex>" with a single
// colon.
func TestDetail01(t *testing.T) {
	h := &Hash{Algorithm: HashAlgorithmSHA256, HashValue: []byte{0xab, 0x01, 0xff}}
	got := h.String()
	want := "sha256:" + bbHex(h.HashValue)
	if got != want {
		t.Fatalf("String = %q, want %q", got, want)
	}
	if strings.Count(got, ":") != 1 {
		t.Fatalf("String has %d colons", strings.Count(got, ":"))
	}
	if h.Hex() != bbHex(h.HashValue) {
		t.Fatalf("Hex = %q", h.Hex())
	}
}

// TestDetail02: HashAlgorithm.FromString enforces the exact hex length per
// algorithm (md5=32, sha1=40, sha256=64) and errors on a length mismatch
// (error text is an implementation detail — only the error is asserted).
func TestDetail02(t *testing.T) {
	md5hex := strings.Repeat("0", 32)
	sha1hex := strings.Repeat("0", 40)
	sha256hex := strings.Repeat("0", 64)

	if h, err := HashAlgorithmMD5.FromString(md5hex); err != nil || h.Algorithm != HashAlgorithmMD5 {
		t.Fatalf("md5 valid: %v %v", h, err)
	}
	if h, err := HashAlgorithmSHA1.FromString(sha1hex); err != nil || h.Algorithm != HashAlgorithmSHA1 {
		t.Fatalf("sha1 valid: %v %v", h, err)
	}
	if h, err := HashAlgorithmSHA256.FromString(sha256hex); err != nil || h.Algorithm != HashAlgorithmSHA256 {
		t.Fatalf("sha256 valid: %v %v", h, err)
	}

	// wrong lengths rejected — even valid hex
	if _, err := HashAlgorithmSHA256.FromString(sha1hex); err == nil {
		t.Fatal("sha256 accepted a 40-char hex")
	}
	if _, err := HashAlgorithmMD5.FromString(sha256hex); err == nil {
		t.Fatal("md5 accepted a 64-char hex")
	}
	// right length but not hex
	if _, err := HashAlgorithmMD5.FromString(strings.Repeat("z", 32)); err == nil {
		t.Fatal("md5 accepted non-hex")
	}
}

// TestDetail03: FromString prefers explicit "alg:" prefixes, then guesses by
// bare hex length; unrecognized lengths error.
func TestDetail03(t *testing.T) {
	sha256hex := strings.Repeat("a", 64)
	sha1hex := strings.Repeat("b", 40)
	md5hex := strings.Repeat("c", 32)

	if h, err := FromString("sha256:" + sha256hex); err != nil || h.Algorithm != HashAlgorithmSHA256 {
		t.Fatalf("sha256: prefix: %v %v", h, err)
	}
	if h, err := FromString("md5:" + md5hex); err != nil || h.Algorithm != HashAlgorithmMD5 {
		t.Fatalf("md5: prefix: %v %v", h, err)
	}
	if h, err := FromString(sha256hex); err != nil || h.Algorithm != HashAlgorithmSHA256 {
		t.Fatalf("bare 64: %v %v", h, err)
	}
	if h, err := FromString(sha1hex); err != nil || h.Algorithm != HashAlgorithmSHA1 {
		t.Fatalf("bare 40: %v %v", h, err)
	}
	if h, err := FromString(md5hex); err != nil || h.Algorithm != HashAlgorithmMD5 {
		t.Fatalf("bare 32: %v %v", h, err)
	}
	// explicit prefix enforces that algorithm's length
	if _, err := FromString("sha256:" + sha1hex); err == nil {
		t.Fatal("sha256: accepted a 40-char hex")
	}
	// unrecognized bare length
	if _, err := FromString(strings.Repeat("d", 30)); err == nil {
		t.Fatal("30-char bare hash accepted")
	}
}

// TestDetail04: NewHasher constructs the matching standard hash for the
// three known algorithms; anything else exits (verified via child process).
func TestDetail04(t *testing.T) {
	if os.Getenv("BB_EXIT_CHILD") == "1" {
		HashAlgorithm("not-an-algorithm").NewHasher()
		os.Exit(0)
	}

	md5sum := md5.Sum([]byte("abc"))
	sha1sum := sha1.Sum([]byte("abc"))
	sha256sum := sha256.Sum256([]byte("abc"))
	cases := []struct {
		alg  HashAlgorithm
		want string // hex digest of "abc"
	}{
		{HashAlgorithmMD5, bbHex(md5sum[:])},
		{HashAlgorithmSHA1, bbHex(sha1sum[:])},
		{HashAlgorithmSHA256, bbHex(sha256sum[:])},
	}
	for _, tc := range cases {
		h := tc.alg.NewHasher()
		if h == nil {
			t.Fatalf("NewHasher(%s) = nil", tc.alg)
		}
		h.Write([]byte("abc"))
		if got := bbHex(h.Sum(nil)); got != tc.want {
			t.Fatalf("%s digest = %q, want %q", tc.alg, got, tc.want)
		}
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail04$")
	cmd.Env = append(os.Environ(), "BB_EXIT_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("unknown algorithm did not exit")
	}
}

// TestDetail05: HashFile propagates not-exist unwrapped; it digests real
// file contents with the chosen algorithm.
func TestDetail05(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-file")
	if _, err := HashAlgorithmSHA256.HashFile(missing); err == nil || !os.IsNotExist(err) {
		t.Fatalf("missing file: %v (os.IsNotExist=%v)", err, os.IsNotExist(err))
	}

	p := filepath.Join(t.TempDir(), "data")
	content := []byte("hash me\n")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := HashAlgorithmSHA256.HashFile(p)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	sum := sha256.Sum256(content)
	if h.Algorithm != HashAlgorithmSHA256 || h.Hex() != bbHex(sum[:]) {
		t.Fatalf("HashFile digest = %v %q", h.Algorithm, h.Hex())
	}
}

// TestDetail06: MustFromString returns the parsed hash on success and exits
// on a parse error (verified via child process).
func TestDetail06(t *testing.T) {
	if os.Getenv("BB_EXIT_CHILD") == "1" {
		MustFromString("definitely-not-a-hash")
		os.Exit(0)
	}

	hexv := strings.Repeat("a", 64)
	h := MustFromString("sha256:" + hexv)
	if h == nil || h.Algorithm != HashAlgorithmSHA256 || h.Hex() != hexv {
		t.Fatalf("MustFromString = %+v", h)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail06$")
	cmd.Env = append(os.Environ(), "BB_EXIT_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("MustFromString on garbage did not exit")
	}
}

// TestDetail07: Equal requires both algorithm and digest bytes to match.
func TestDetail07(t *testing.T) {
	digest := []byte{1, 2, 3}
	a := &Hash{Algorithm: HashAlgorithmSHA256, HashValue: digest}
	b := &Hash{Algorithm: HashAlgorithmSHA256, HashValue: []byte{1, 2, 3}}
	if !a.Equal(b) || !b.Equal(a) {
		t.Fatal("equal hashes not equal")
	}
	c := &Hash{Algorithm: HashAlgorithmMD5, HashValue: []byte{1, 2, 3}}
	if a.Equal(c) || c.Equal(a) {
		t.Fatal("different algorithms with equal bytes compared equal")
	}
	d := &Hash{Algorithm: HashAlgorithmSHA256, HashValue: []byte{1, 2, 4}}
	if a.Equal(d) {
		t.Fatal("different digests compared equal")
	}
}
