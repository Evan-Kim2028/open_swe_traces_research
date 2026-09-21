// Hidden black-box suite for httpgetter. Exported API only: NewHTTPGetter, HTTPGetter.Get.
package getter_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.internal/helm/pkg/getter"
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

func TestDetail01_GetCopiesThenOverlaysNoMutation(t *testing.T) {
	_ = hiddenSeed()
	var ua atomic.Value
	ua.Store("")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua.Store(r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter(getter.WithUserAgent("stored"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(srv.URL, getter.WithUserAgent("call")); err != nil {
		t.Fatal(err)
	}
	if ua.Load().(string) != "call" {
		t.Fatalf("call opts must overlay stored, UA=%q", ua.Load())
	}
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if ua.Load().(string) != "stored" {
		t.Fatalf("stored opts must be unmodified after a call overlay, UA=%q", ua.Load())
	}
}

func TestDetail02_AcceptIfSetUADefaultAndOverride(t *testing.T) {
	var accept, ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		ua = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if accept != "" {
		t.Fatalf("Accept must be omitted when empty, got %q", accept)
	}
	if ua == "" {
		t.Fatal("User-Agent always set (Helm default)")
	}
	if !strings.Contains(strings.ToLower(ua), "helm") && ua == "" {
		t.Fatalf("default UA %q", ua)
	}
	g2, err := getter.NewHTTPGetter(getter.WithUserAgent("mine"), getter.WithAcceptHeader("application/gzip"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g2.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if ua != "mine" {
		t.Fatalf("UA override %q", ua)
	}
	if accept != "application/gzip" {
		t.Fatalf("Accept %q", accept)
	}
}

func TestDetail03_BasicAuthScopePassAllOrSameHostPort(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter(
		getter.WithBasicAuth("u", "p"),
		getter.WithURL("http://other.example:9"),
	)
	if err != nil {
		t.Fatal(err)
	}
	auth = "SENTINEL"
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if auth != "" {
		t.Fatalf("different host must withhold auth, got %q", auth)
	}
	g2, err := getter.NewHTTPGetter(
		getter.WithBasicAuth("u", "p"),
		getter.WithURL(srv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g2.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if auth == "" {
		t.Fatal("same scheme+host:port must send basic auth")
	}
	g3, err := getter.NewHTTPGetter(
		getter.WithBasicAuth("u", "p"),
		getter.WithURL("http://other.example"),
		getter.WithPassCredentialsAll(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	auth = ""
	if _, err := g3.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if auth == "" {
		t.Fatal("passCredentialsAll must send auth even on host mismatch")
	}
	g4, err := getter.NewHTTPGetter(getter.WithBasicAuth("u", ""))
	if err != nil {
		t.Fatal(err)
	}
	auth = "SENTINEL"
	if _, err := g4.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if auth != "" {
		t.Fatal("empty password must not send auth")
	}
}

func TestDetail04_DualParseErrorWording(t *testing.T) {
	g, err := getter.NewHTTPGetter()
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Get("http://[::1")
	if err == nil {
		t.Fatal("href parse error")
	}
	a := err.Error()
	g2, err := getter.NewHTTPGetter(getter.WithURL("http://[::1"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	_, err = g2.Get(srv.URL)
	if err == nil {
		t.Fatal("stored URL parse error must surface")
	}
	b := err.Error()
	if a == b {
		t.Fatalf("two parse sites must wrap distinctly: %q", a)
	}
	if !strings.Contains(a, "unable to parse") && !strings.Contains(b, "unable to parse") {
		t.Fatalf("parse wraps:\n href=%q\n stored=%q", a, b)
	}
	if strings.Contains(a, "unable to parse getter URL") && strings.Contains(b, "unable to parse getter URL") && a == b {
		t.Fatal("wraps must differ")
	}
}

func TestDetail05_Non200ErrorSpacesAroundColon(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter()
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Get(srv.URL)
	if err == nil {
		t.Fatal("non-200 must error")
	}
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("prefix: %v", err)
	}
	if !strings.Contains(err.Error(), " : ") {
		t.Fatalf("spaces around colon: %q", err.Error())
	}
}

func TestDetail06_NeedsCustomTLSFormula(t *testing.T) {
	// cert alone (no key) is NOT custom: request against httptest HTTP still works
	// without trying to load TLS files.
	g, err := getter.NewHTTPGetter(getter.WithTLSClientConfig("cert-only.pem", "", ""))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatalf("cert alone must not force custom TLS (would fail to read cert): %v", err)
	}
	g2, err := getter.NewHTTPGetter(getter.WithTLSClientConfig("no.pem", "no.key", ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g2.Get(srv.URL); err == nil {
		t.Fatal("cert+key pair must take custom TLS path (missing files error)")
	}
	g3, err := getter.NewHTTPGetter(getter.WithTLSClientConfig("", "", "no-ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g3.Get(srv.URL); err == nil {
		t.Fatal("ca-only is custom TLS")
	}
	g4, err := getter.NewHTTPGetter(getter.WithInsecureSkipVerifyTLS(true))
	if err != nil {
		t.Fatal(err)
	}
	// insecure is custom TLS but still works against plain HTTP
	if _, err := g4.Get(srv.URL); err != nil {
		t.Fatalf("insecure custom TLS against http: %v", err)
	}
}

func TestDetail07_TransportPrecedenceExplicitThenCustomThenShared(t *testing.T) {
	var viaCustom atomic.Bool
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&http.Transport{Proxy: http.ProxyFromEnvironment}).DialContext,
	}
	// wrap via RoundTripper: http.Transport is used as-is when WithTransport is set
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter(getter.WithTransport(tr))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	_ = viaCustom
	// two getters without custom TLS share a lazily-built transport: both succeed
	gA, _ := getter.NewHTTPGetter()
	gB, _ := getter.NewHTTPGetter()
	if _, err := gA.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := gB.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
}

func TestDetail08_DisableCompressionTimeoutAlwaysApplied(t *testing.T) {
	var ae string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ae = r.Header.Get("Accept-Encoding")
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	g, err := getter.NewHTTPGetter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(srv.URL); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ae, "gzip") {
		t.Fatalf("DisableCompression: client must not advertise gzip, Accept-Encoding=%q", ae)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	t.Cleanup(slow.Close)
	gt, err := getter.NewHTTPGetter(getter.WithTimeout(50 * time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = gt.Get(slow.URL)
	if err == nil {
		t.Fatal("timeout must apply")
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("timeout not applied, elapsed %s", time.Since(start))
	}
}
