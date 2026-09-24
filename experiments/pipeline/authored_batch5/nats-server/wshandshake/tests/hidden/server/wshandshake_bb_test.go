package server

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"testing"
)

// TestDetail01: wsAcceptKey is base64(SHA1(key + RFC6455 GUID)) — the
// canonical RFC test vector pins the formula.
func TestDetail01(t *testing.T) {
	if got := wsAcceptKey("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("accept key = %q", got)
	}
}

// TestDetail02: wsMakeChallengeKey returns base64 of 16 random bytes.
func TestDetail02(t *testing.T) {
	k, err := wsMakeChallengeKey()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(k)
	if err != nil || len(raw) != 16 {
		t.Fatalf("challenge key %q should be base64 of 16 bytes: %v", k, err)
	}
}

// TestDetail03: wsHeaderContains splits header values on ',', trims
// space/tab, and matches the needle case-insensitively.
func TestDetail03(t *testing.T) {
	h := http.Header{}
	h.Add("Connection", "keep-alive, Upgrade")
	h.Add("Connection", "close")
	if !wsHeaderContains(h, "Connection", "upgrade") {
		t.Fatal("comma-separated token should match case-insensitively")
	}
	if !wsHeaderContains(h, "Connection", "CLOSE") {
		t.Fatal("second header line should match too")
	}
	if wsHeaderContains(h, "Connection", "down") {
		t.Fatal("non-token must not match")
	}
	if wsHeaderContains(h, "Missing", "x") {
		t.Fatal("missing header must not match")
	}
}

// TestDetail04: wsPMCExtensionSupport returns (supported, noContextTakeover)
// — the second requires BOTH server_ and client_ params on the
// permessage-deflate extension itself; checkPMCOnly skips the params scan.
func TestDetail04(t *testing.T) {
	h := http.Header{}
	h.Add("Sec-Websocket-Extensions", "permessage-deflate; server_no_context_takeover; client_no_context_takeover")
	sup, nct := wsPMCExtensionSupport(h, false)
	if !sup || !nct {
		t.Fatalf("both params: %v %v", sup, nct)
	}
	sup, nct = wsPMCExtensionSupport(h, true)
	if !sup || nct {
		t.Fatalf("checkPMCOnly should skip params: %v %v", sup, nct)
	}
	h = http.Header{}
	h.Add("Sec-Websocket-Extensions", "permessage-deflate; server_no_context_takeover")
	if sup, nct := wsPMCExtensionSupport(h, false); !sup || nct {
		t.Fatalf("missing client param: %v %v", sup, nct)
	}
	h = http.Header{}
	h.Add("Sec-Websocket-Extensions", "x-other; client_no_context_takeover, permessage-deflate")
	if sup, nct := wsPMCExtensionSupport(h, false); !sup || nct {
		t.Fatalf("params on a different extension must not count: %v %v", sup, nct)
	}
	h = http.Header{}
	if sup, _ := wsPMCExtensionSupport(h, false); sup {
		t.Fatal("no extension header should not report support")
	}
}

// TestDetail05: wsGetHostAndPort splits host:port, substitutes 443/80 for
// a missing port, lowercases the host, and propagates other split errors.
func TestDetail05(t *testing.T) {
	if h, p, err := wsGetHostAndPort(true, "host.com"); h != "host.com" || p != "443" || err != nil {
		t.Fatalf("tls default: %q %q %v", h, p, err)
	}
	if h, p, err := wsGetHostAndPort(false, "HOST.COM"); h != "host.com" || p != "80" || err != nil {
		t.Fatalf("non-tls default + lowercase: %q %q %v", h, p, err)
	}
	if h, p, err := wsGetHostAndPort(false, "host.com:8080"); h != "host.com" || p != "8080" || err != nil {
		t.Fatalf("explicit port kept: %q %q %v", h, p, err)
	}
	if _, _, err := wsGetHostAndPort(true, "bad:port:x"); err == nil {
		t.Fatal("malformed hostport must propagate the split error")
	}
}

// TestDetail06: checkOrigin accepts when neither sameOrigin nor an allowed
// list is configured, and accepts unconditionally when no Origin (or
// Sec-Websocket-Origin) header is present.
func TestDetail06(t *testing.T) {
	w := &srvWebsocket{}
	r := &http.Request{Host: "h.com:80", Header: http.Header{}}
	r.Header.Set("Origin", "http://evil.com")
	if err := w.checkOrigin(r); err != nil {
		t.Fatalf("unconfigured should accept any origin: %v", err)
	}
	w = &srvWebsocket{sameOrigin: true}
	r = &http.Request{Host: "h.com", Header: http.Header{}}
	if err := w.checkOrigin(r); err != nil {
		t.Fatalf("missing Origin header should accept: %v", err)
	}
}

// TestDetail07: sameOrigin requires host, port, AND scheme (request scheme
// from r.TLS != nil, compared case-insensitively); an allowed-list entry
// must match the (scheme, port) pair for that host — both checks apply.
func TestDetail07(t *testing.T) {
	w := &srvWebsocket{sameOrigin: true}
	r := &http.Request{Host: "h.com", Header: http.Header{}}
	r.Header.Set("Origin", "http://h.com")
	if err := w.checkOrigin(r); err != nil {
		t.Fatalf("same origin should pass: %v", err)
	}
	r.Header.Set("Origin", "http://evil.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("different host must fail")
	}
	r.Header.Set("Origin", "https://h.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("different scheme must fail")
	}
	r = &http.Request{Host: "h.com", Header: http.Header{}, TLS: &tls.ConnectionState{}}
	r.Header.Set("Origin", "http://h.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("TLS request vs http origin must fail on scheme")
	}
	// Allowed list requires the (scheme, port) pair for that host.
	w = &srvWebsocket{allowedOrigins: map[string][]*allowedOrigin{"h.com": {{scheme: "https", port: "443"}}}}
	r = &http.Request{Host: "x.com", Header: http.Header{}}
	r.Header.Set("Origin", "https://h.com")
	if err := w.checkOrigin(r); err != nil {
		t.Fatalf("allowed origin should pass: %v", err)
	}
	r.Header.Set("Origin", "https://h.com:8443")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("wrong port for allowed host must fail")
	}
	r.Header.Set("Origin", "https://z.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("unlisted host must fail")
	}
	// Both checks apply: an allowed-listed foreign host still fails
	// sameOrigin, and a same-origin host absent from the list fails too.
	w = &srvWebsocket{sameOrigin: true,
		allowedOrigins: map[string][]*allowedOrigin{"h2.com": {{scheme: "http", port: "80"}}}}
	r = &http.Request{Host: "h.com", Header: http.Header{}}
	r.Header.Set("Origin", "http://h2.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("allowed-listed host must still satisfy sameOrigin")
	}
	r.Header.Set("Origin", "http://h.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("same-origin host absent from the allowed list must fail")
	}
	// Sec-Websocket-Origin is the fallback header name.
	r = &http.Request{Host: "h.com", Header: http.Header{}}
	r.Header.Set("Sec-Websocket-Origin", "http://evil.com")
	w = &srvWebsocket{sameOrigin: true}
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("Sec-Websocket-Origin must be honoured")
	}
}

// TestDetail08: an explicit port in the request Host is kept through
// wsGetHostAndPort, so it is compared against the origin's own resolved
// port — an explicit non-default port fails against a bare-host origin.
func TestDetail08(t *testing.T) {
	w := &srvWebsocket{sameOrigin: true}
	// Request Host carries :443 without TLS; the http origin resolves to 80.
	r := &http.Request{Host: "h.com:443", Header: http.Header{}}
	r.Header.Set("Origin", "http://h.com")
	if err := w.checkOrigin(r); err == nil {
		t.Fatal("explicit 443 vs implicit 80 must fail on port")
	}
	// And symmetric: TLS request bare host vs https origin with :443 passes.
	r = &http.Request{Host: "h.com", Header: http.Header{}, TLS: &tls.ConnectionState{}}
	r.Header.Set("Origin", "https://h.com:443")
	if err := w.checkOrigin(r); err != nil {
		t.Fatalf("443 vs 443 should pass: %v", err)
	}
}
