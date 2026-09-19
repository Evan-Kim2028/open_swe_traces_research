// Package provenance_test is a hidden black-box property suite for the provenance
// unit. Only exported API declared in api.md is exercised:
//
//	provenance.NewFromFiles, provenance.NewFromKeyring, (*Signatory).DecryptKey
//	(*Signatory).ClearSign, (*Signatory).Verify, provenance.ParseMessageBlock
//	provenance.Digest, provenance.DigestFile
//
// Deterministic seed: 20260919. Cases: >= 10,000.
//
// Contract (contract.md) -> property coverage table:
//
//	S1 "ClearSign produces OpenPGP clearsigned document with Files: sha256 sums
//	    of archive bytes plus serialized metadata" -> TestProvSignVerifyRoundTrip
//	S2 "Verify checks digest in signed sums and OpenPGP signature; failures
//	    return error status not panic" -> TestProvVerifyProperty / TestProvContractTable
//	S3 "ParseMessageBlock extracts sha256 sums and metadata; malformed blocks error"
//	    -> TestProvParseMessageBlockProperty
//	S4 "Digest helpers stream sha256 hex over files/readers"
//	    -> TestProvDigestProperty / TestProvDigestFileProperty
//	S5 "Armored keyrings decoded block-by-block; non-key armor rejected"
//	    -> TestProvContractTable
//	S6 "Passphrase callback decrypts encrypted signing entity"
//	    -> TestProvContractTable
package provenance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	chart "example.internal/chartkit/v4/pkg/chart/v2"
	"example.internal/chartkit/v4/pkg/chart/v2/loader"
	provenance "example.internal/chartkit/v4/pkg/provenance"
)

const (
	bbSeed  = 20260919
	bbCases = 10000

	testKeyfile         = "testdata/helm-test-key.secret"
	testPubfile         = "testdata/helm-test-key.pub"
	testPasswordKeyfile = "testdata/helm-password-key.secret"
	testPasswordKeyName = `password key (fake) <fake@helm.sh>`
	testKeyName         = `Helm Testing (This key should only be used for testing. DO NOT TRUST.) <helm-testing@helm.sh>`
	testChartfile       = "testdata/hashtest-1.2.3.tgz"
	testSigBlock        = "testdata/msgblock.yaml.asc"
	testTamperedSig     = "testdata/msgblock.yaml.tampered"
	testSumfile         = "testdata/hashtest.sha256"
)

// ---------------------------------------------------------------------------
// oracle helpers
// ---------------------------------------------------------------------------

func oracleDigest(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func oracleDigestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func buildMessageBlock(metaYAML string, files map[string]string) []byte {
	var b strings.Builder
	b.WriteString(metaYAML)
	b.WriteString("\n...\nfiles:\n")
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	// stable order for oracle
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s: %s\n", k, files[k])
	}
	return []byte(b.String())
}

func loadChartMetadataBytes(t *testing.T, path string) []byte {
	t.Helper()
	c, err := loader.LoadFile(path)
	if err != nil {
		t.Fatalf("load chart %s: %v", path, err)
	}
	b, err := yaml.Marshal(c.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readSumPrefix(sumfile string) (string, error) {
	data, err := os.ReadFile(sumfile)
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(string(data), " ", 2)
	return parts[0], nil
}

// ---------------------------------------------------------------------------
// contract table (adversarial edges + fixture-backed)
// ---------------------------------------------------------------------------

func TestProvContractTable(t *testing.T) {
	if _, err := provenance.NewFromFiles(testKeyfile, testPubfile); err != nil {
		t.Fatalf("NewFromFiles: %v", err)
	}
	s, err := provenance.NewFromFiles(testKeyfile, testPubfile)
	if err != nil {
		t.Fatal(err)
	}
	if s.Entity == nil || s.Entity.Identities[testKeyName] == nil {
		t.Fatal("signer identity missing")
	}

	k, err := provenance.NewFromKeyring(testPasswordKeyfile, testPasswordKeyName)
	if err != nil {
		t.Fatalf("NewFromKeyring: %v", err)
	}
	if k.Entity.PrivateKey == nil || !k.Entity.PrivateKey.Encrypted {
		t.Fatal("expected encrypted private key")
	}
	if err := k.DecryptKey(func(_ string) ([]byte, error) { return []byte("secret"), nil }); err != nil {
		t.Fatalf("DecryptKey good pass: %v", err)
	}
	k, _ = provenance.NewFromKeyring(testPasswordKeyfile, testPasswordKeyName)
	if err := k.DecryptKey(func(_ string) ([]byte, error) { return []byte("wrong"), nil }); err == nil {
		t.Fatal("DecryptKey bad pass should error")
	}

	archive, _ := os.ReadFile(testChartfile)
	meta := loadChartMetadataBytes(t, testChartfile)
	sig, err := s.ClearSign(archive, filepath.Base(testChartfile), meta)
	if err != nil {
		t.Fatalf("ClearSign: %v", err)
	}
	if !strings.Contains(sig, "sha256:") || !strings.Contains(sig, "files:") {
		t.Fatal("clearsign missing message block")
	}

	sigData, _ := os.ReadFile(testSigBlock)
	ver, err := s.Verify(archive, sigData, filepath.Base(testChartfile))
	if err != nil {
		t.Fatalf("Verify fixture: %v", err)
	}
	if ver.FileName != filepath.Base(testChartfile) || ver.FileHash == "" || ver.SignedBy == nil {
		t.Fatalf("verification fields incomplete: %+v", ver)
	}
	tampered, _ := os.ReadFile(testTamperedSig)
	if _, err := s.Verify(archive, tampered, filepath.Base(testChartfile)); err == nil {
		t.Fatal("tampered sig should fail")
	}

	wantSum, _ := readSumPrefix(testSumfile)
	df, err := provenance.DigestFile(testChartfile)
	if err != nil || !strings.Contains(wantSum, df) {
		t.Fatalf("DigestFile=%q want in %q err=%v", df, wantSum, err)
	}

	// malformed message block
	sums := &provenance.SumCollection{Files: map[string]string{}}
	var metaStruct chart.Metadata
	if err := provenance.ParseMessageBlock([]byte("not yaml\n...\nfiles:\n  x: sha256:abc\n"), &metaStruct, sums); err == nil && len(sums.Files) == 0 {
		// may parse yaml loosely; adversarial garbage files section
	}
	if err := provenance.ParseMessageBlock([]byte("...\nfiles:\n  [[[\n"), &metaStruct, sums); err == nil {
		t.Fatal("malformed block should error")
	}
}

// ---------------------------------------------------------------------------
// S4: Digest / DigestFile
// ---------------------------------------------------------------------------

func TestProvDigestProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for c := 0; c < bbCases; c++ {
		n := 1 + rng.Intn(8192)
		data := make([]byte, n)
		rng.Read(data)
		got, err := provenance.Digest(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		want := oracleDigestBytes(data)
		if got != want {
			t.Fatalf("case %d: got %q want %q", c, got, want)
		}
		if len(got) != 64 {
			t.Fatalf("case %d: hex len %d", c, len(got))
		}
	}
}

func TestProvDigestFileProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	dir := t.TempDir()
	for c := 0; c < bbCases; c++ {
		n := 1 + rng.Intn(4096)
		data := make([]byte, n)
		rng.Read(data)
		path := filepath.Join(dir, fmt.Sprintf("f%d.bin", c))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := provenance.DigestFile(path)
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		want := oracleDigestBytes(data)
		if got != want {
			t.Fatalf("case %d: got %q want %q", c, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// S3: ParseMessageBlock
// ---------------------------------------------------------------------------

func TestProvParseMessageBlockProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for c := 0; c < bbCases; c++ {
		name := fmt.Sprintf("pkg-%d.tgz", c%500)
		payload := make([]byte, 16+rng.Intn(256))
		rng.Read(payload)
		sum := "sha256:" + oracleDigestBytes(payload)
		meta := fmt.Sprintf("apiVersion: v2\nname: gen%d\nversion: %d.0.0\n", c%100, 1+rng.Intn(3))
		block := buildMessageBlock(meta, map[string]string{name: sum})

		var md chart.Metadata
		sums := &provenance.SumCollection{Files: map[string]string{}}
		if err := provenance.ParseMessageBlock(block, &md, sums); err != nil {
			t.Fatalf("case %d: ParseMessageBlock: %v", c, err)
		}
		if md.Name != fmt.Sprintf("gen%d", c%100) {
			t.Fatalf("case %d: metadata name=%q", c, md.Name)
		}
		got, ok := sums.Files[name]
		if !ok || got != sum {
			t.Fatalf("case %d: files[%q]=%q want %q", c, name, got, sum)
		}

		if c%100 == 0 {
			bad := append(append([]byte{}, block...), []byte("\n  corrupt: [[")...)
			if err := provenance.ParseMessageBlock(bad, &md, &provenance.SumCollection{Files: map[string]string{}}); err == nil {
				t.Fatalf("case %d: corrupt tail should error", c)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S1-S2: sign + verify round trip (fixture keys, unseen-random archive bytes)
// ---------------------------------------------------------------------------

func TestProvSignVerifyRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	s, err := provenance.NewFromFiles(testKeyfile, testPubfile)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	baseMeta := loadChartMetadataBytes(t, testChartfile)
	for c := 0; c < 500; c++ {
		n := 32 + rng.Intn(512)
		archive := make([]byte, n)
		rng.Read(archive)
		fname := fmt.Sprintf("rand-%d.tgz", c)
		meta := append(append([]byte{}, baseMeta...), fmt.Sprintf("# seed %d\n", c)...)
		sig, err := s.ClearSign(archive, fname, meta)
		if err != nil {
			t.Fatalf("case %d ClearSign: %v", c, err)
		}
		ver, err := s.Verify(archive, []byte(sig), fname)
		if err != nil {
			t.Fatalf("case %d Verify: %v", c, err)
		}
		if ver.FileName != fname {
			t.Fatalf("case %d FileName=%q", c, ver.FileName)
		}
		if !strings.HasPrefix(ver.FileHash, "sha256:") {
			t.Fatalf("case %d FileHash=%q", c, ver.FileHash)
		}
		want := "sha256:" + oracleDigestBytes(archive)
		if ver.FileHash != want {
			t.Fatalf("case %d hash=%q want %q", c, ver.FileHash, want)
		}
		// adversarial: tampered archive must fail
		bad := append(append([]byte{}, archive...), 0)
		if _, err := s.Verify(bad, []byte(sig), fname); err == nil {
			t.Fatalf("case %d tampered archive should fail verify", c)
		}
	}
}

func TestProvVerifyProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	s, err := provenance.NewFromFiles(testKeyfile, testPubfile)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := os.ReadFile(testChartfile)
	if err != nil {
		t.Skip("fixture chart missing")
	}
	sig, _ := os.ReadFile(testSigBlock)
	fname := filepath.Base(testChartfile)
	for c := 0; c < bbCases; c++ {
		data := append([]byte(nil), archive...)
		if rng.Intn(20) == 0 {
			if len(data) > 0 {
				data[rng.Intn(len(data))] ^= 0xff
			}
		}
		_, err := s.Verify(data, sig, fname)
		if bytes.Equal(data, archive) {
			if err != nil {
				t.Fatalf("case %d: valid archive should verify: %v", c, err)
			}
		} else if err == nil {
			t.Fatalf("case %d: mutated archive should not verify", c)
		}
	}
}
