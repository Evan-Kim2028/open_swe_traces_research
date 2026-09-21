// Hidden black-box suite for tlsutil. Exported API only: NewTLSConfig and option funcs.
package tlsutil_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.internal/helm/internal/tlsutil"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func writePEMPair(t *testing.T) (certPath, keyPath, caPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bb"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certPath = dir + "/c.pem"
	keyPath = dir + "/k.pem"
	caPath = dir + "/ca.pem"
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath, caPath
}

func TestDetail01_OptionsRunAllErrorsJoined(t *testing.T) {
	missingA := t.TempDir() + "/no-a"
	missingB := t.TempDir() + "/no-b"
	_, err := tlsutil.NewTLSConfig(
		tlsutil.WithCertKeyPairFiles(missingA, missingB),
		tlsutil.WithCAFile(t.TempDir()+"/no-ca"),
	)
	if err == nil {
		t.Fatal("expected joined errors from later options after earlier failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "unable to read cert file") {
		t.Fatalf("missing cert error in join: %v", err)
	}
	if !strings.Contains(msg, "can't read CA file") && !strings.Contains(msg, "unable to read key file") {
		t.Fatalf("later option errors must still run (errors.Join): %v", err)
	}
}

func TestDetail02_CertKeyPairEmptyNoopVsSinglePath(t *testing.T) {
	cfg, err := tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles("", ""))
	if err != nil {
		t.Fatalf("both-empty must no-op: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	_, err = tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles("missing-cert.pem", ""))
	if err == nil {
		t.Fatal("one-empty still reads the cert path")
	}
	if !strings.Contains(err.Error(), "unable to read cert file") {
		t.Fatalf("cert is read first: %v", err)
	}
	cert, key, _ := writePEMPair(t)
	_, err = tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles(cert, t.TempDir()+"/no-key"))
	if err == nil {
		t.Fatal("missing key must error after cert")
	}
	if !strings.Contains(err.Error(), "unable to read key file") {
		t.Fatalf("key error wording: %v", err)
	}
	_ = key
}

func TestDetail03_CAFileEmptyNoopVsReadError(t *testing.T) {
	cfg, err := tlsutil.NewTLSConfig(tlsutil.WithCAFile(""))
	if err != nil {
		t.Fatalf("empty CA is no-op: %v", err)
	}
	if cfg == nil {
		t.Fatal("nil config")
	}
	_, err = tlsutil.NewTLSConfig(tlsutil.WithCAFile(t.TempDir() + "/nope"))
	if err == nil {
		t.Fatal("missing CA must error")
	}
	if !strings.Contains(err.Error(), "can't read CA file") {
		t.Fatalf("CA error wording: %v", err)
	}
}

func TestDetail04_CertificateOnlyWhenBothPEMBlocks(t *testing.T) {
	cert, key, _ := writePEMPair(t)
	cfg, err := tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles(cert, key))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("both PEM blocks present → cert set, got %d", len(cfg.Certificates))
	}
	// junk cert + junk key: load fails
	dir := t.TempDir()
	cpath := dir + "/c.pem"
	kpath := dir + "/k.pem"
	if err := os.WriteFile(cpath, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kpath, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles(cpath, kpath))
	if err == nil {
		t.Fatal("bad pair must error")
	}
	if !strings.Contains(err.Error(), "unable to load cert from key pair") {
		t.Fatalf("bad pair wording: %v", err)
	}
	// single PEM block (cert file has cert, key file empty-of-PEM): silently no cert
	if err := os.WriteFile(kpath, []byte("-----BEGIN EC PRIVATE KEY-----\n\n-----END EC PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg2, err := tlsutil.NewTLSConfig(tlsutil.WithCertKeyPairFiles(cert, kpath))
	if err != nil {
		// either load error or silent no cert — both allowed if no cert is set on success
		return
	}
	if len(cfg2.Certificates) != 0 {
		t.Fatal("single valid block must not set Certificates")
	}
}

func TestDetail05_CAAppendFailure(t *testing.T) {
	dir := t.TempDir()
	ca := dir + "/ca.pem"
	// PEM-shaped but not a certificate → AppendCertsFromPEM false
	if err := os.WriteFile(ca, []byte("-----BEGIN CERTIFICATE-----\nQQ==\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := tlsutil.NewTLSConfig(tlsutil.WithCAFile(ca))
	if err == nil {
		t.Fatal("bad CA PEM must fail append")
	}
	if !strings.Contains(err.Error(), "failed to append certificates from pem block") {
		t.Fatalf("append wording: %v", err)
	}
}

func TestDetail06_InsecurePassthroughNilRootCAs(t *testing.T) {
	rng := mrand.New(mrand.NewSource(hiddenSeed()))
	for i := 0; i < 8; i++ {
		insec := rng.Intn(2) == 0
		cfg, err := tlsutil.NewTLSConfig(tlsutil.WithInsecureSkipVerify(insec))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.InsecureSkipVerify != insec {
			t.Fatalf("InsecureSkipVerify=%v got %v", insec, cfg.InsecureSkipVerify)
		}
		if cfg.RootCAs != nil {
			t.Fatal("without CA, RootCAs must be nil")
		}
	}
	_, _, ca := writePEMPair(t)
	cfg, err := tlsutil.NewTLSConfig(tlsutil.WithCAFile(ca), tlsutil.WithInsecureSkipVerify(true))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("insecure must passthrough alongside CA")
	}
	if cfg.RootCAs == nil {
		t.Fatal("CA present → RootCAs set")
	}
}

func TestDetail07_NoOptionsValidConfig(t *testing.T) {
	cfg, err := tlsutil.NewTLSConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("no options must still return a config")
	}
	if cfg.RootCAs != nil {
		t.Fatal("empty config RootCAs nil")
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("default insecure is false")
	}
}
