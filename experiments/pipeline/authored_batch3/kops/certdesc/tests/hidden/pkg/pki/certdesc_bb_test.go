// Package pki_test is the hidden black-box suite for certdesc.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// PkixNameToString, BuildTypeDescription, IssueCert (+ kept tables).
package pki_test

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"regexp"
	"strings"
	"testing"

	pki "example.internal/clustkit/pkg/pki"
)

var oidKeyShape = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

// Detail 1 (Inferable: no): RDN sequence flattens to comma-joined key=value
// pairs. Shape asserted: one k=v pair per RDN, values preserved verbatim, no
// spaces, unknown OIDs get a numeric (dotted) key. The short-name map itself
// (cn/serial/c/...) is NOT pinned.
func TestDetail01(t *testing.T) {
	name := &pkix.Name{
		CommonName:         "alice",
		Organization:       []string{"corp"},
		OrganizationalUnit: []string{"eng"},
		SerialNumber:       "sn-9",
	}
	got := pki.PkixNameToString(name)
	if got == "" {
		t.Fatal("PkixNameToString returned empty for populated name")
	}
	if strings.Contains(got, " ") {
		t.Fatalf("output contains space: %q", got)
	}
	parts := strings.Split(got, ",")
	if len(parts) != 4 {
		t.Fatalf("expected 4 comma-joined pairs, got %d in %q", len(parts), got)
	}
	seenVals := map[string]bool{}
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			t.Fatalf("pair %q is not k=v", p)
		}
		seenVals[kv[1]] = true
	}
	for _, v := range []string{"alice", "corp", "eng", "sn-9"} {
		if !seenVals[v] {
			t.Fatalf("value %q missing from %q", v, got)
		}
	}

	// Unknown OID: key renders in numeric (dotted) form, value preserved.
	n2 := &pkix.Name{
		ExtraNames: []pkix.AttributeTypeAndValue{
			{Type: asn1.ObjectIdentifier{1, 2, 3, 4}, Value: "zz"},
		},
	}
	got2 := pki.PkixNameToString(n2)
	kv := strings.SplitN(got2, "=", 2)
	if len(kv) != 2 || kv[1] != "zz" {
		t.Fatalf("unknown-OID pair malformed: %q", got2)
	}
	if !oidKeyShape.MatchString(kv[0]) {
		t.Fatalf("unknown-OID key %q is not numeric-dotted", kv[0])
	}
}

// Detail 2 (Inferable: partially): every SET KeyUsage bit renders as its
// KeyUsage* Go identifier. The table is kept in the tree, so the identifier
// spellings are pinned; order across mixed bits is not.
func TestDetail02(t *testing.T) {
	cases := []struct {
		usage x509.KeyUsage
		want  string
	}{
		{x509.KeyUsageDigitalSignature, "KeyUsageDigitalSignature"},
		{x509.KeyUsageContentCommitment, "KeyUsageContentCommitment"},
		{x509.KeyUsageKeyEncipherment, "KeyUsageKeyEncipherment"},
		{x509.KeyUsageDataEncipherment, "KeyUsageDataEncipherment"},
		{x509.KeyUsageKeyAgreement, "KeyUsageKeyAgreement"},
		{x509.KeyUsageCertSign, "KeyUsageCertSign"},
		{x509.KeyUsageCRLSign, "KeyUsageCRLSign"},
		{x509.KeyUsageEncipherOnly, "KeyUsageEncipherOnly"},
		{x509.KeyUsageDecipherOnly, "KeyUsageDecipherOnly"},
	}
	for _, c := range cases {
		cert := &x509.Certificate{KeyUsage: c.usage}
		got := pki.BuildTypeDescription(cert)
		if !strings.Contains(got, c.want) {
			t.Fatalf("KeyUsage %v: %q does not contain %q", c.usage, got, c.want)
		}
	}
}

// Detail 3 (Inferable: partially): the parse side exact-matches the same
// tables and fails closed. Observable through IssueCert's type expansion:
// unknown / near-miss tokens error and name the offending token; a real
// token does not produce the unrecognized-option error.
func TestDetail03(t *testing.T) {
	ctx := context.Background()
	for _, bad := range []string{
		"KeyUsageBogus",
		"keyusagecertsign",
		"KeyUsageCertSignX",
		"ExtKeyUsageUnknown",
		"ExtKeyUsageServerAuthx",
		"CA,KeyUsageCertSign,KeyUsageNope",
	} {
		_, _, _, err := pki.IssueCert(ctx, &pki.IssueCertRequest{
			Type:    bad,
			Subject: pkix.Name{CommonName: "t"},
		}, nil)
		if err == nil {
			t.Fatalf("Type %q: expected error, got success", bad)
		}
		if !strings.Contains(err.Error(), "KeyUsageNope") && !strings.Contains(err.Error(), bad) {
			t.Fatalf("Type %q: error %q does not name the offending token", bad, err)
		}
	}
	// Exact match on a real token must NOT take the unrecognized-option path:
	// a CA-flavoured type stays self-signed (nil keystore is never touched) and
	// the parsed bit must land on the issued certificate.
	cert, _, _, err := pki.IssueCert(ctx, &pki.IssueCertRequest{
		Type:    "CA,KeyUsageCertSign",
		Subject: pkix.Name{CommonName: "t"},
	}, nil)
	if err != nil {
		t.Fatalf("valid type rejected: %v", err)
	}
	if cert == nil || cert.Certificate == nil {
		t.Fatal("IssueCert returned no certificate")
	}
	if cert.Certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("issued cert missing KeyUsageCertSign bit")
	}
	if cert.Certificate.KeyUsage&x509.KeyUsageCRLSign != 0 {
		t.Fatal("issued cert has KeyUsageCRLSign though token was not given")
	}
}

// Detail 4 (Inferable: no): known ExtKeyUsage renders by name (table kept ->
// pinned); unknown renders as some distinct non-empty token. The fallback
// spelling is NOT pinned.
func TestDetail04(t *testing.T) {
	cert := &x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	got := pki.BuildTypeDescription(cert)
	if !strings.Contains(got, "ExtKeyUsageServerAuth") {
		t.Fatalf("known ext usage not rendered by name: %q", got)
	}

	unk := &x509.Certificate{ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsage(77)}}
	gotU := pki.BuildTypeDescription(unk)
	if gotU == "" {
		t.Fatal("unknown ext usage rendered as empty string")
	}
	if strings.Contains(gotU, "ExtKeyUsageServerAuth") || strings.Contains(gotU, "ExtKeyUsageClientAuth") {
		t.Fatalf("unknown ext usage rendered a known name: %q", gotU)
	}
	if strings.Contains(gotU, ",") {
		t.Fatalf("single unknown ext usage produced multiple tokens: %q", gotU)
	}
}

// Detail 5 (Inferable: no on the line, but the canonical table survives in
// kept issue.go, so the short names ARE derivable): canonical usage combos
// collapse to the wellKnownCertificateTypes key; non-canonical combos do not.
func TestDetail05(t *testing.T) {
	cases := []struct {
		cert *x509.Certificate
		want string
	}{
		{&x509.Certificate{IsCA: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}, "ca"},
		{&x509.Certificate{KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}, "server"},
		{&x509.Certificate{KeyUsage: x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}, "client"},
		{&x509.Certificate{KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}}, "clientServer"},
		{&x509.Certificate{IsCA: true}, "CA"},
	}
	for i, c := range cases {
		if got := pki.BuildTypeDescription(c.cert); got != c.want {
			t.Fatalf("case %d: got %q want %q", i, got, c.want)
		}
	}
	// Sorted collection: a two-usage non-CA cert produces both names.
	got := pki.BuildTypeDescription(&x509.Certificate{KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign})
	for _, tok := range []string{"KeyUsageCertSign", "KeyUsageCRLSign"} {
		if !strings.Contains(got, tok) {
			t.Fatalf("multi-usage description %q missing %q", got, tok)
		}
	}
}

// Detail 6 (Inferable: partially): comma join carries no spaces; IssueCert
// parses the same comma-separated form back — round-trip through a real
// issued CA cert collapses to the short type name.
func TestDetail06(t *testing.T) {
	got := pki.BuildTypeDescription(&x509.Certificate{
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	})
	if strings.Contains(got, ", ") {
		t.Fatalf("comma join contains space: %q", got)
	}
	if !strings.Contains(got, ",") {
		t.Fatalf("multi-token description missing comma: %q", got)
	}

	ctx := context.Background()
	cert, _, _, err := pki.IssueCert(ctx, &pki.IssueCertRequest{
		Type:    "CA,KeyUsageCRLSign,KeyUsageCertSign",
		Subject: pkix.Name{CommonName: "roundtrip"},
	}, nil)
	if err != nil {
		t.Fatalf("IssueCert comma-form type failed: %v", err)
	}
	if cert == nil || cert.Certificate == nil {
		t.Fatal("IssueCert returned no certificate")
	}
	if !cert.Certificate.IsCA {
		t.Fatal("issued cert is not CA")
	}
	if desc := pki.BuildTypeDescription(cert.Certificate); desc != "ca" {
		t.Fatalf("round-trip description got %q want ca", desc)
	}
}
