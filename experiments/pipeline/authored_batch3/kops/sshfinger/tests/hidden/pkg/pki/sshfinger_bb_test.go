// Package pki_test is the hidden black-box suite for sshfinger.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// ComputeAWSKeyFingerprint, ComputeOpenSSHKeyFingerprint.
package pki_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"

	"example.internal/clustkit/pkg/pki"
	"golang.org/x/crypto/ssh"
)

var colonHexRE = regexp.MustCompile(`^([0-9a-f]{2}:)+[0-9a-f]{2}$`)

func rsaKey(t *testing.T) (ssh.PublicKey, *rsa.PublicKey, string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return pub, &k.PublicKey, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

func ed25519Key(t *testing.T) (ssh.PublicKey, string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		t.Fatal(err)
	}
	return pub, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

func ecdsaKey(t *testing.T) (ssh.PublicKey, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return pub, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
}

func colonHex(b []byte) string {
	h := hex.EncodeToString(b)
	var parts []string
	for i := 0; i < len(h); i += 2 {
		parts = append(parts, h[i:i+2])
	}
	return strings.Join(parts, ":")
}

// Detail 1 (Inferable: partially): the key line splits on whitespace, needs
// >=2 tokens, base64-decodes token[1] — the algorithm token is IGNORED and a
// trailing comment is accepted.
func TestDetail01(t *testing.T) {
	_, _, line := rsaKey(t)
	toks := strings.Fields(line)
	blob := toks[1]

	want, err := pki.ComputeOpenSSHKeyFingerprint(line)
	if err != nil {
		t.Fatalf("canonical line: %v", err)
	}
	// foreign algorithm token is ignored — same blob gives the same fingerprint
	got, err := pki.ComputeOpenSSHKeyFingerprint("bogus-algo " + blob)
	if err != nil || got != want {
		t.Fatalf("ignored-algo line: fp=%q err=%v want %q", got, err, want)
	}
	// trailing comment silently accepted
	got, err = pki.ComputeOpenSSHKeyFingerprint(line + " my comment")
	if err != nil || got != want {
		t.Fatalf("commented line: fp=%q err=%v", got, err)
	}
	// fewer than 2 tokens / bad base64 must error
	for _, bad := range []string{"", "onlyonetoken", "ssh-rsa !!!notbase64!!!"} {
		if _, err := pki.ComputeOpenSSHKeyFingerprint(bad); err == nil {
			t.Fatalf("bad input %q accepted", bad)
		}
	}
}

// Detail 2 (Inferable: yes): colon-separated lowercase hex — a colon between
// every byte pair.
func TestDetail02(t *testing.T) {
	_, _, line := rsaKey(t)
	fp, err := pki.ComputeOpenSSHKeyFingerprint(line)
	if err != nil {
		t.Fatal(err)
	}
	if !colonHexRE.MatchString(fp) {
		t.Fatalf("not colon-separated lowercase hex: %q", fp)
	}
}

func md5Colon(b []byte) string {
	s := md5.Sum(b)
	return colonHex(s[:])
}

// Detail 3 (Inferable: no): AWS fingerprint is key-type dependent — RSA
// hashes the PKIX DER in colon-hex; ed25519 is the OpenSSH SHA256 form;
// anything else errors.
func TestDetail03(t *testing.T) {
	_, rsaPub, line := rsaKey(t)
	der, err := x509.MarshalPKIXPublicKey(rsaPub)
	if err != nil {
		t.Fatal(err)
	}
	want := md5Colon(der)
	got, err := pki.ComputeAWSKeyFingerprint(line)
	if err != nil || got != want {
		t.Fatalf("rsa aws fp = %q err=%v want %q", got, err, want)
	}

	ed, edLine := ed25519Key(t)
	got, err = pki.ComputeAWSKeyFingerprint(edLine)
	if err != nil {
		t.Fatalf("ed25519 aws fp: %v", err)
	}
	if !strings.HasPrefix(got, "SHA256:") {
		t.Fatalf("ed25519 aws fp lacks SHA256 form: %q", got)
	}
	sum := sha256.Sum256(ed.Marshal())
	body := strings.TrimPrefix(got, "SHA256:")
	var dec []byte
	if d, err := base64.StdEncoding.DecodeString(body); err == nil {
		dec = d
	} else if d, err := base64.RawStdEncoding.DecodeString(body); err == nil {
		dec = d
	}
	if len(dec) == 0 || !equalBytes(dec, sum[:]) {
		t.Fatalf("ed25519 aws fp body is not sha256(wire): %q", got)
	}

	_, ecLine := ecdsaKey(t)
	if _, err := pki.ComputeAWSKeyFingerprint(ecLine); err == nil {
		t.Fatal("ecdsa aws fp did not error")
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Detail 4 (Inferable: no): OpenSSH fingerprint is md5 over the SSH wire
// encoding in colon-hex for EVERY key type — no per-algorithm branch.
func TestDetail04(t *testing.T) {
	rsaPub, _, rsaLine := rsaKey(t)
	want := md5Colon(rsaPub.Marshal())
	got, err := pki.ComputeOpenSSHKeyFingerprint(rsaLine)
	if err != nil || got != want {
		t.Fatalf("rsa openssh fp = %q err=%v want %q", got, err, want)
	}

	ed, edLine := ed25519Key(t)
	want = md5Colon(ed.Marshal())
	got, err = pki.ComputeOpenSSHKeyFingerprint(edLine)
	if err != nil || got != want {
		t.Fatalf("ed25519 openssh fp = %q err=%v want %q", got, err, want)
	}

	ec, ecLine := ecdsaKey(t)
	want = md5Colon(ec.Marshal())
	got, err = pki.ComputeOpenSSHKeyFingerprint(ecLine)
	if err != nil || got != want {
		t.Fatalf("ecdsa openssh fp = %q err=%v want %q", got, err, want)
	}
}

// Detail 5 (Inferable: partially): rsaToDER marshals the PKIX DER of the
// underlying *rsa.PublicKey — not the SSH wire form. Observable: the RSA AWS
// fingerprint differs from the OpenSSH fingerprint (DER != wire) and equals
// colon-hex(md5(der)) — already covered above; here we assert the DER/wire
// distinction explicitly.
func TestDetail05(t *testing.T) {
	_, rsaPub, line := rsaKey(t)
	aws, err := pki.ComputeAWSKeyFingerprint(line)
	if err != nil {
		t.Fatal(err)
	}
	ossl, err := pki.ComputeOpenSSHKeyFingerprint(line)
	if err != nil {
		t.Fatal(err)
	}
	if aws == ossl {
		t.Fatal("aws rsa fp == openssh fp: DER and wire encodings conflated")
	}
	der, err := x509.MarshalPKIXPublicKey(rsaPub)
	if err != nil {
		t.Fatal(err)
	}
	if aws != md5Colon(der) {
		t.Fatal("aws rsa fp is not md5 over PKIX DER")
	}
}
