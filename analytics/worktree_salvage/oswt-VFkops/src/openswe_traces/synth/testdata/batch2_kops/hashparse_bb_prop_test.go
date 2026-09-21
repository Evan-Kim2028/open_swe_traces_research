package hashing_test

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"example.internal/kops/util/pkg/hashing"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func randHex(rng *rand.Rand, n int) string {
	const d = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = d[rng.Intn(16)]
	}
	return string(b)
}

// Detail 1: Hash.String = "algo:" + lowercase hex; Hex = lowercase hex of
// raw bytes.
func TestDetail01_StringAndHexFormats(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	algos := []hashing.HashAlgorithm{hashing.HashAlgorithmMD5, hashing.HashAlgorithmSHA1, hashing.HashAlgorithmSHA256}
	for i := 0; i < 200; i++ {
		a := algos[rng.Intn(3)]
		raw := make([]byte, 1+rng.Intn(64))
		rng.Read(raw)
		h := &hashing.Hash{Algorithm: a, HashValue: raw}
		wantHex := hex.EncodeToString(raw)
		if h.Hex() != wantHex {
			t.Fatalf("i=%d Hex()=%q want %q", i, h.Hex(), wantHex)
		}
		if h.String() != string(a)+":"+wantHex {
			t.Fatalf("i=%d String()=%q want %q", i, h.String(), string(a)+":"+wantHex)
		}
	}
}

// Detail 2: ha.FromString checks length BEFORE hex decode; error strings
// distinguish the two failures.
func TestDetail02_FromStringLengthBeforeHex(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	lens := map[hashing.HashAlgorithm]int{
		hashing.HashAlgorithmMD5:    32,
		hashing.HashAlgorithmSHA1:   40,
		hashing.HashAlgorithmSHA256: 64,
	}
	for a, l := range lens {
		// Wrong length (even valid hex) -> length error naming the count.
		for i := 0; i < 40; i++ {
			n := rng.Intn(l + 10)
			if n == l {
				n++
			}
			s := randHex(rng, n)
			_, err := a.FromString(s)
			if err == nil {
				t.Fatalf("%s FromString len=%d accepted", a, n)
			}
			want := fmt.Sprintf(`invalid %q hash - unexpected length %d`, string(a), n)
			if err.Error() != want {
				t.Fatalf("%s len err %q want %q", a, err.Error(), want)
			}
		}
		// Right length but non-hex -> "not hex" error quoting the input.
		for i := 0; i < 20; i++ {
			s := randHex(rng, l)
			pos := rng.Intn(l)
			s = s[:pos] + "z" + s[pos+1:]
			_, err := a.FromString(s)
			if err == nil {
				t.Fatalf("%s non-hex accepted", a)
			}
			want := fmt.Sprintf(`invalid hash %q - not hex`, s)
			if err.Error() != want {
				t.Fatalf("%s hex err %q want %q", a, err.Error(), want)
			}
		}
		// Correct round-trip.
		s := randHex(rng, l)
		h, err := a.FromString(s)
		if err != nil {
			t.Fatalf("%s FromString(%q) err %v", a, s, err)
		}
		if h.Algorithm != a || h.Hex() != s {
			t.Fatalf("%s round-trip mismatch: %v", a, h)
		}
	}
}

// Detail 3: ha.FromString on an unknown algorithm -> `unknown hash
// algorithm: "<ha>"`.
func TestDetail03_UnknownAlgorithmError(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 30; i++ {
		bad := hashing.HashAlgorithm(fmt.Sprintf("sha%d", 512+rng.Intn(1000)))
		_, err := bad.FromString(randHex(rng, 64))
		if err == nil {
			t.Fatalf("unknown algorithm %q accepted", bad)
		}
		want := fmt.Sprintf(`unknown hash algorithm: %q`, string(bad))
		if err.Error() != want {
			t.Fatalf("err %q want %q", err.Error(), want)
		}
	}
}

// Detail 4: FromString prefix dispatch — md5:/sha1:/sha256: delegate; no
// prefix infers by length only; unrecognized prefixes are treated as bare
// strings judged by total length.
func TestDetail04_PrefixDispatchAndLengthInference(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		a := []hashing.HashAlgorithm{hashing.HashAlgorithmMD5, hashing.HashAlgorithmSHA1, hashing.HashAlgorithmSHA256}[rng.Intn(3)]
		l := map[hashing.HashAlgorithm]int{hashing.HashAlgorithmMD5: 32, hashing.HashAlgorithmSHA1: 40, hashing.HashAlgorithmSHA256: 64}[a]
		s := randHex(rng, l)
		h, err := hashing.FromString(string(a) + ":" + s)
		if err != nil {
			t.Fatalf("i=%d prefixed parse err %v", i, err)
		}
		if h.Algorithm != a || h.Hex() != s {
			t.Fatalf("i=%d prefixed parse %v want %s:%s", i, h, a, s)
		}
		// Unprefixed: inferred by length.
		h2, err := hashing.FromString(s)
		if err != nil {
			t.Fatalf("i=%d unprefixed parse err %v", i, err)
		}
		if h2.Algorithm != a {
			t.Fatalf("i=%d inferred algorithm %q want %q", i, h2.Algorithm, a)
		}
	}
	// Unrecognized prefix -> judged by total length.
	h, err := hashing.FromString("sha512:" + randHex(rng, 57))
	if err == nil {
		// 7+57 = 64 chars -> would infer sha256 with a bogus hex; either way
		// it must NOT be treated as a sha512 algorithm.
		if h.Algorithm == hashing.HashAlgorithm("sha512") {
			t.Fatalf("sha512 prefix honored")
		}
	}
	// Length that matches no algorithm -> error.
	for i := 0; i < 30; i++ {
		n := rng.Intn(100)
		if n == 32 || n == 40 || n == 64 {
			continue
		}
		if _, err := hashing.FromString(randHex(rng, n)); err == nil {
			t.Fatalf("i=%d len=%d inferred a hash", i, n)
		}
	}
}

// Detail 5: MustFromString exits fatally on parse error — observable only
// as a successful return on valid input (fatal paths can't be caught).
func TestDetail05_MustFromStringValidInput(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	s := randHex(rng, 64)
	h := hashing.MustFromString("sha256:" + s)
	if h.Algorithm != hashing.HashAlgorithmSHA256 || h.Hex() != s {
		t.Fatalf("MustFromString round-trip mismatch: %v", h)
	}
}

// Detail 6: Hash/HashFile stream-hash; HashFile returns the raw error for
// a nonexistent file.
func TestDetail06_HashAndHashFile(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		data := make([]byte, rng.Intn(2048))
		rng.Read(data)
		a := []hashing.HashAlgorithm{hashing.HashAlgorithmMD5, hashing.HashAlgorithmSHA1, hashing.HashAlgorithmSHA256}[rng.Intn(3)]
		h, err := a.Hash(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("i=%d Hash err %v", i, err)
		}
		var want []byte
		switch a {
		case hashing.HashAlgorithmMD5:
			s := md5.Sum(data)
			want = s[:]
		case hashing.HashAlgorithmSHA1:
			s := sha1.Sum(data)
			want = s[:]
		case hashing.HashAlgorithmSHA256:
			s := sha256.Sum256(data)
			want = s[:]
		}
		if h.Algorithm != a || !bytes.Equal(h.HashValue, want) {
			t.Fatalf("i=%d Hash %s mismatch", i, a)
		}
	}
	// HashFile on a real file.
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	payload := []byte("hello")
	if err := os.WriteFile(p, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := hashing.HashAlgorithmSHA256.HashFile(p)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	want := sha256.Sum256(payload)
	if !bytes.Equal(h.HashValue, want[:]) {
		t.Fatalf("HashFile digest mismatch")
	}
	// Nonexistent file: raw os error (os.IsNotExist true).
	_, err = hashing.HashAlgorithmSHA256.HashFile(filepath.Join(dir, "missing"))
	if err == nil {
		t.Fatalf("HashFile missing accepted")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("HashFile missing err=%v want not-exist", err)
	}
}

// Detail 7: Equal requires same algorithm AND same bytes.
func TestDetail07_EqualAlgorithmSensitive(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		raw := make([]byte, 16)
		rng.Read(raw)
		l := &hashing.Hash{Algorithm: hashing.HashAlgorithmSHA256, HashValue: raw}
		r := &hashing.Hash{Algorithm: hashing.HashAlgorithmSHA256, HashValue: append([]byte{}, raw...)}
		if !l.Equal(r) {
			t.Fatalf("i=%d identical hashes unequal", i)
		}
		r2 := &hashing.Hash{Algorithm: hashing.HashAlgorithmSHA1, HashValue: append([]byte{}, raw...)}
		if l.Equal(r2) {
			t.Fatalf("i=%d different algorithms equal", i)
		}
		r3 := &hashing.Hash{Algorithm: hashing.HashAlgorithmSHA256, HashValue: append([]byte{}, raw...)}
		r3.HashValue[0] ^= 0xff
		if l.Equal(r3) {
			t.Fatalf("i=%d different bytes equal", i)
		}
	}
}
