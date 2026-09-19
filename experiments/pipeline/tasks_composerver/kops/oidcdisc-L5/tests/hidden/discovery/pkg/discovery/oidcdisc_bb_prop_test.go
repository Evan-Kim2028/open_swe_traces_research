// Package discovery_test — hidden black-box property suite for oidcdisc.
// Exported API only (api.md): NewMemoryStore, NewServer, ServeHTTP (via httptest).
// Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"objects in one universe are invisible in another" -> TestOIDCDiscoveryIsolationProperty
//	"well-known document issuer/jwks_uri and 404 when no OIDC spec" -> TestOIDCDiscoveryDocumentProperty
//	"JWKS merge by kid with LastSeen winner" -> TestOIDCJWKSMergeProperty
//	"apply/create identity and namespace checks" -> TestOIDCDiscoveryAuthProperty
//	"List of an unknown universe is empty" -> TestOIDCMemoryStoreListProperty
package discovery_test

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "example.internal/clustkit/discovery/apis/discovery.kops.k8s.io/v1alpha1"
	"example.internal/clustkit/discovery/pkg/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const bbSeed = 20260919
const bbCases = 10000

type bbSrv struct {
	store *discovery.MemoryStore
	srv   *httptest.Server
}

func bbNewServer(t *testing.T) *bbSrv {
	t.Helper()
	store := discovery.NewMemoryStore()
	h := discovery.NewServer(store)
	s := httptest.NewUnstartedServer(h)
	s.TLS = &tls.Config{ClientAuth: tls.RequestClientCert}
	s.StartTLS()
	return &bbSrv{store: store, srv: s}
}

func bbCA(t *testing.T, cn string) (*x509.Certificate, *rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return cert, key, hex.EncodeToString(h[:])
}

func bbPemCerts(certs ...*x509.Certificate) []byte {
	var b bytes.Buffer
	for _, c := range certs {
		pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	}
	return b.Bytes()
}

func bbPemKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func bbClientCert(t *testing.T, ca *x509.Certificate, caPriv *rsa.PrivateKey, cn string) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	leafKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafT := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, leafT, ca, &leafKey.PublicKey, caPriv)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return leaf, leafKey
}

func bbHTTPClient(t *testing.T, bb *bbSrv, leaf, ca *x509.Certificate, leafKey *rsa.PrivateKey) *http.Client {
	t.Helper()
	tlsCert := tls.Certificate{Certificate: [][]byte{leaf.Raw, ca.Raw}, PrivateKey: leafKey}
	pool := x509.NewCertPool()
	pool.AddCert(bb.srv.TLS.Certificates[0].Leaf)
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: pool, Certificates: []tls.Certificate{tlsCert},
	}}}
}

func bbAnonClient(t *testing.T, bb *bbSrv) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(bb.srv.TLS.Certificates[0].Leaf)
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
}

func bbPostEP(t *testing.T, cli *http.Client, base, universe, ns, body string) int {
	t.Helper()
	url := fmt.Sprintf("%s/%s/apis/discovery.clustkit.k8s.io/v1alpha1/namespaces/%s/discoveryendpoints", base, universe, ns)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestOIDCMemoryStoreListProperty(t *testing.T) {
	store := discovery.NewMemoryStore()
	ctx := context.Background()
	eps, err := store.ListDiscoveryEndpoints(ctx, "missing-universe")
	if err != nil || len(eps) != 0 {
		t.Fatalf("unknown universe list: %v len=%d", err, len(eps))
	}
	got, err := store.GetDiscoveryEndpoint(ctx, "missing", "ns", "n")
	if err != nil || got != nil {
		t.Fatalf("get missing: %v %v", got, err)
	}

	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		id := fmt.Sprintf("u%d", rng.Intn(200))
		list, err := store.ListDiscoveryEndpoints(ctx, id)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if list == nil {
			t.Fatalf("case %d nil list", i)
		}
	}
}

func TestOIDCDiscoveryIsolationProperty(t *testing.T) {
	bb := bbNewServer(t)
	defer bb.srv.Close()
	ca1, k1, u1 := bbCA(t, "ca1")
	ca2, k2, u2 := bbCA(t, "ca2")
	l1, lk1 := bbClientCert(t, ca1, k1, "client-a")
	l2, lk2 := bbClientCert(t, ca2, k2, "client-b")
	c1 := bbHTTPClient(t, bb, l1, ca1, lk1)
	c2 := bbHTTPClient(t, bb, l2, ca2, lk2)
	body1 := `{"metadata":{"namespace":"default","name":"client-a"},"spec":{"addresses":["1.2.3.4"]}}`
	body2 := `{"metadata":{"namespace":"default","name":"client-b"},"spec":{"addresses":["5.6.7.8"]}}`
	if bbPostEP(t, c1, bb.srv.URL, u1, "default", body1) != http.StatusCreated {
		t.Fatal("create u1")
	}
	if bbPostEP(t, c2, bb.srv.URL, u2, "default", body2) != http.StatusCreated {
		t.Fatal("create u2")
	}
	lst1, _ := bb.store.ListDiscoveryEndpoints(context.Background(), u1)
	lst2, _ := bb.store.ListDiscoveryEndpoints(context.Background(), u2)
	if len(lst1) != 1 || len(lst2) != 1 || lst1[0].Name != "client-a" || lst2[0].Name != "client-b" {
		t.Fatal("isolation")
	}

	for i := 0; i < bbCases; i++ {
		u := fmt.Sprintf("iso%d", i)
		ctx := context.Background()
		ep := &api.DiscoveryEndpoint{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: fmt.Sprintf("n%d", i)},
			Spec:       api.DiscoveryEndpointSpec{Addresses: []string{"9.9.9.9"}},
		}
		if err := bb.store.UpsertDiscoveryEndpoint(ctx, u, ep); err != nil {
			t.Fatalf("case %d upsert: %v", i, err)
		}
		other, err := bb.store.ListDiscoveryEndpoints(ctx, u+"-other")
		if err != nil || len(other) != 0 {
			t.Fatalf("case %d leak", i)
		}
	}
}

func TestOIDCDiscoveryDocumentProperty(t *testing.T) {
	bb := bbNewServer(t)
	defer bb.srv.Close()
	ca, key, uid := bbCA(t, "oidc-ca")
	leaf, leafKey := bbClientCert(t, ca, key, "oidc-node")
	cli := bbHTTPClient(t, bb, leaf, ca, leafKey)
	anon := bbAnonClient(t, bb)

	resp, err := anon.Get(bb.srv.URL + "/" + uid + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("no oidc spec status %d", resp.StatusCode)
	}
	resp.Body.Close()

	oidcBody := `{"metadata":{"namespace":"default"},"spec":{"oidc":{"keys":[{"kty":"RSA","kid":"1"}]}}}`
	if bbPostEP(t, cli, bb.srv.URL, uid, "default", oidcBody) != http.StatusCreated {
		t.Fatal("register oidc")
	}
	req, _ := http.NewRequest(http.MethodGet, bb.srv.URL+"/"+uid+"/.well-known/openid-configuration", nil)
	req.Host = "discovery.example.com"
	resp, err = anon.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("oidc status %d", resp.StatusCode)
	}
	var doc map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&doc)
	resp.Body.Close()
	issuer := "https://discovery.example.com/" + uid + "/"
	if doc["issuer"] != issuer || doc["jwks_uri"] != issuer+"openid/v1/jwks" {
		t.Fatalf("issuer/jwks_uri %+v", doc)
	}

	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		host := fmt.Sprintf("h%d.example.com", rng.Intn(50))
		req, _ := http.NewRequest(http.MethodGet, bb.srv.URL+"/"+uid+"/.well-known/openid-configuration", nil)
		req.Host = host
		resp, err := anon.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("case %d", i)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func TestOIDCJWKSMergeProperty(t *testing.T) {
	bb := bbNewServer(t)
	defer bb.srv.Close()
	_, _, uid := bbCA(t, "jwks-ca")
	anon := bbAnonClient(t, bb)
	ctx := context.Background()
	upsert := func(name, kid, n string) {
		ep := &api.DiscoveryEndpoint{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name},
			Spec: api.DiscoveryEndpointSpec{
				OIDC: &api.OIDCSpec{Keys: []api.JSONWebKey{{KeyID: kid, N: n}}},
			},
		}
		if err := bb.store.UpsertDiscoveryEndpoint(ctx, uid, ep); err != nil {
			t.Fatal(err)
		}
	}
	upsert("a", "1", "old")
	upsert("c", "2", "two")
	time.Sleep(2 * time.Second)
	upsert("b", "1", "new")
	resp, err := anon.Get(bb.srv.URL + "/" + uid + "/openid/v1/jwks")
	if err != nil {
		t.Fatal(err)
	}
	var jwks map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&jwks)
	resp.Body.Close()
	keys, _ := jwks["keys"].([]interface{})
	if len(keys) != 2 {
		t.Fatalf("key count %d", len(keys))
	}
	for _, k := range keys {
		m := k.(map[string]interface{})
		if m["kid"] == "1" && m["n"] != "new" {
			t.Fatal("kid 1 should be newer LastSeen")
		}
	}

	for i := 0; i < bbCases; i++ {
		resp, err := anon.Get(bb.srv.URL + "/" + uid + "/openid/v1/jwks")
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		var j map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&j)
		resp.Body.Close()
		ks, _ := j["keys"].([]interface{})
		if len(ks) != 2 {
			t.Fatalf("case %d keys %d", i, len(ks))
		}
	}
}

func TestOIDCDiscoveryAuthProperty(t *testing.T) {
	bb := bbNewServer(t)
	defer bb.srv.Close()
	ca, key, uid := bbCA(t, "auth-ca")
	leaf, leafKey := bbClientCert(t, ca, key, "right-name")
	cli := bbHTTPClient(t, bb, leaf, ca, leafKey)
	bad := `{"metadata":{"namespace":"default","name":"wrong-name"},"spec":{"addresses":["1.1.1.1"]}}`
	if bbPostEP(t, cli, bb.srv.URL, uid, "default", bad) != http.StatusForbidden {
		t.Fatal("name mismatch")
	}
	badNS := `{"metadata":{"namespace":"other"},"spec":{"addresses":["1.1.1.1"]}}`
	if bbPostEP(t, cli, bb.srv.URL, uid, "other-ns", badNS) != http.StatusForbidden {
		t.Fatal("namespace mismatch")
	}

	ssaLeaf, ssaKey := bbClientCert(t, ca, key, "ssa-node")
	cfg := &rest.Config{
		Host: bb.srv.URL + "/" + uid,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: bbPemCerts(bb.srv.TLS.Certificates[0].Leaf), CertData: bbPemCerts(ssaLeaf, ca), KeyData: bbPemKey(ssaKey),
		},
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ep := &api.DiscoveryEndpoint{
		TypeMeta: metav1.TypeMeta{Kind: "DiscoveryEndpoint", APIVersion: "discovery.kops.k8s.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "ssa-node", Namespace: "default"},
		Spec:       api.DiscoveryEndpointSpec{Addresses: []string{"8.8.8.8"}},
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ep)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dc.Resource(api.DiscoveryEndpointGVR).Namespace("default").Create(context.Background(), &unstructured.Unstructured{Object: u}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		ns := "default"
		if rng.Intn(3) == 0 {
			ns = "kube-system"
		}
		body := fmt.Sprintf(`{"metadata":{"namespace":"%s"},"spec":{"addresses":["10.0.0.%d"]}}`, ns, rng.Intn(200))
		code := bbPostEP(t, cli, bb.srv.URL, uid, ns, body)
		if code != http.StatusCreated {
			t.Fatalf("case %d code %d", i, code)
		}
	}
}
