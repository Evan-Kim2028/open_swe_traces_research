// Package pki_test is a hidden black-box property suite for issuecert (pki).
// Exported API only (api.md): IssueCert, GeneratePrivateKey, ParsePEM*, Keystore.
// Seed 20260919; >=10k cases.
//
// Contract (contract.md) -> property coverage table:
//
//	"self-signed CA usages and issuer==subject" -> TestIssuecertCAProperty
//	"client usages" -> TestIssuecertClientProperty
//	"clientServer DNS + IP SANs" -> TestIssuecertClientServerProperty
//	"server EKU" -> TestIssuecertServerProperty
//	"PEM private key round-trip" -> TestIssuecertPEMRandomProperty
//	"certificate PEM round-trip" -> TestIssuecertPEMRandomProperty
package pki_test

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"math/rand"
	"net"
	"testing"
	"time"

	"example.internal/clustkit/pkg/pki"
)

const bbSeed = 20260919
const bbCases = 10000

type bbKeystore struct {
	certs map[string]*pki.Certificate
	keys  map[string]*pki.PrivateKey
}

func (k *bbKeystore) FindPrimaryKeypair(ctx context.Context, name string) (*pki.Certificate, *pki.PrivateKey, error) {
	return k.certs[name], k.keys[name], nil
}

func bbCtx() context.Context { return context.Background() }

func bbIssueCA(t *testing.T) (*pki.Certificate, *pki.PrivateKey) {
	t.Helper()
	cert, key, ca, err := pki.IssueCert(bbCtx(), &pki.IssueCertRequest{
		Type:    "ca",
		Subject: pkix.Name{CommonName: "test-ca"},
	}, nil)
	if err != nil {
		t.Fatalf("IssueCert ca: %v", err)
	}
	if ca == nil || cert == nil || key == nil {
		t.Fatal("ca issuance returns cert, key, ca")
	}
	if cert.Certificate == nil || !cert.Certificate.IsCA {
		t.Fatal("ca type must be CA")
	}
	if cert.Certificate.Issuer.String() != cert.Certificate.Subject.String() {
		t.Fatal("self-signed issuer==subject")
	}
	return cert, key
}

func TestIssuecertCAProperty(t *testing.T) {
	if pki.DefaultPrivateKeySize <= 0 {
		t.Fatal("DefaultPrivateKeySize exported")
	}
	bbIssueCA(t)
}

func TestIssuecertClientProperty(t *testing.T) {
	caCert, caKey := bbIssueCA(t)
	ks := &bbKeystore{
		certs: map[string]*pki.Certificate{"ca": caCert},
		keys:  map[string]*pki.PrivateKey{"ca": caKey},
	}
	serial := big.NewInt(42)
	cert, _, _, err := pki.IssueCert(bbCtx(), &pki.IssueCertRequest{
		Type:    "client",
		Signer:  "ca",
		Subject: pkix.Name{CommonName: "client"},
		Serial:  serial,
	}, ks)
	if err != nil {
		t.Fatal(err)
	}
	x := cert.Certificate
	if x.SerialNumber.Cmp(serial) != 0 {
		t.Fatalf("serial kept: %v", x.SerialNumber)
	}
	if !hasEKU(x, x509.ExtKeyUsageClientAuth) {
		t.Fatal("client EKU")
	}
}

func TestIssuecertClientServerProperty(t *testing.T) {
	caCert, caKey := bbIssueCA(t)
	ks := &bbKeystore{
		certs: map[string]*pki.Certificate{"ca": caCert},
		keys:  map[string]*pki.PrivateKey{"ca": caKey},
	}
	cert, _, _, err := pki.IssueCert(bbCtx(), &pki.IssueCertRequest{
		Type:           "clientServer",
		Signer:         "ca",
		Subject:        pkix.Name{CommonName: "dual"},
		AlternateNames: []string{" 10.0.0.1 ", "host.example.com", ""},
	}, ks)
	if err != nil {
		t.Fatal(err)
	}
	x := cert.Certificate
	if !hasEKU(x, x509.ExtKeyUsageClientAuth) || !hasEKU(x, x509.ExtKeyUsageServerAuth) {
		t.Fatal("clientServer EKU")
	}
	if len(x.IPAddresses) != 1 || !x.IPAddresses[0].Equal(net.ParseIP("10.0.0.1")) {
		t.Fatalf("IP SAN: %v", x.IPAddresses)
	}
	if len(x.DNSNames) != 1 || x.DNSNames[0] != "host.example.com" {
		t.Fatalf("DNS SAN: %v", x.DNSNames)
	}
}

func TestIssuecertServerProperty(t *testing.T) {
	caCert, caKey := bbIssueCA(t)
	ks := &bbKeystore{
		certs: map[string]*pki.Certificate{"ca": caCert},
		keys:  map[string]*pki.PrivateKey{"ca": caKey},
	}
	key, err := pki.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	cert, issuedKey, ca, err := pki.IssueCert(bbCtx(), &pki.IssueCertRequest{
		Type:       "server",
		Signer:     "ca",
		Subject:    pkix.Name{CommonName: "server"},
		PrivateKey: key,
		Validity:   365 * 24 * time.Hour,
	}, ks)
	if err != nil {
		t.Fatal(err)
	}
	if ca != caCert {
		t.Fatal("non-CA returns keystore CA pointer")
	}
	if issuedKey != key {
		t.Fatal("supplied private key preserved")
	}
	if !hasEKU(cert.Certificate, x509.ExtKeyUsageServerAuth) {
		t.Fatal("server EKU")
	}
}

func hasEKU(c *x509.Certificate, e x509.ExtKeyUsage) bool {
	for _, u := range c.ExtKeyUsage {
		if u == e {
			return true
		}
	}
	return false
}

func TestIssuecertPEMRandomProperty(t *testing.T) {
	key, err := pki.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	keyPEM, err := key.AsBytes()
	if err != nil {
		t.Fatal(err)
	}
	caCert, caKey := bbIssueCA(t)
	ks := &bbKeystore{
		certs: map[string]*pki.Certificate{"ca": caCert},
		keys:  map[string]*pki.PrivateKey{"ca": caKey},
	}
	cert, _, _, err := pki.IssueCert(bbCtx(), &pki.IssueCertRequest{
		Type:    "server",
		Signer:  "ca",
		Subject: pkix.Name{CommonName: "pemrtt"},
	}, ks)
	if err != nil {
		t.Fatal(err)
	}
	certPEM, err := cert.AsBytes()
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		if rng.Intn(2) == 0 {
			k2, err := pki.ParsePEMPrivateKey(keyPEM)
			if err != nil {
				t.Fatalf("case %d parse key: %v", i, err)
			}
			if k2 == nil {
				t.Fatalf("case %d nil key", i)
			}
		} else {
			c2, err := pki.ParsePEMCertificate(certPEM)
			if err != nil {
				t.Fatalf("case %d parse cert: %v", i, err)
			}
			if c2.Certificate.Subject.CommonName != "pemrtt" {
				t.Fatalf("case %d subject drift", i)
			}
		}
	}
}
