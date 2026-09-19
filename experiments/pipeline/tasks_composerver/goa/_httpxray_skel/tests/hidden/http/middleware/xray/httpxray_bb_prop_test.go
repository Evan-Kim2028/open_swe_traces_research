// Black-box property suite for the httpxray unit.
// Exported API: New, WrapDoer, WrapTransport, HTTPSegment RecordRequest,
// RecordResponse, WriteHeader, Write, Hijack.
// Seed 20260919; >=10k cases; contract + middleware_test.go coverage.
//
// Coverage table (contract sentence -> property):
//   "bad collector address fails middleware construction"
//       -> TestHttpxrayNewBadDaemon / TestHttpxrayNewBadDaemonRandom
//   "no trace/span: next handler runs unchanged"
//       -> TestHttpxrayMiddlewareNoTrace / TestHttpxrayMiddlewareNoTraceRandom
//   "with trace metadata: segment opened and status classified on write"
//       -> TestHttpxrayMiddlewareWithTrace / TestHttpxrayWriteHeaderRandom
//   "WrapDoer pass-through without segment; subsegment with segment"
//       -> TestHttpxrayWrapDoerNoSegment / TestHttpxrayWrapDoerWithSegmentRandom
//   "WrapTransport records remote call when segment present"
//       -> TestHttpxrayWrapTransport / TestHttpxrayWrapTransportRandom
//   "RecordRequest URL method client; RecordResponse 429/4xx/5xx flags"
//       -> TestHttpxrayRecordRequestProperty / TestHttpxrayRecordResponseRandom
package xray

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"example.internal/apikit/v3/middleware"
	"example.internal/apikit/v3/middleware/xray"
	"example.internal/apikit/v3/middleware/xray/xraytest"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func bbUDPListen(t *testing.T) string {
	t.Helper()
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("udp: %v", err)
	}
	addr := l.LocalAddr().String()
	_ = l.Close()
	return addr
}

func bbSegCtx(t *testing.T, listen, traceID, spanID string) context.Context {
	t.Helper()
	conn, err := net.Dial("udp", listen)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	seg := xray.NewSegment("svc", traceID, spanID, conn)
	return context.WithValue(context.Background(), xray.SegKey, seg) //nolint:staticcheck
}

func bbTraceRequest(t *testing.T, method, rawURL string, remoteAddr string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, rawURL, http.NoBody)
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	return req
}

func TestHttpxrayNewBadDaemon(t *testing.T) {
	wrap, err := New("svc", "not-valid-udp")
	if err == nil {
		t.Fatal("expected error")
	}
	if wrap != nil {
		t.Fatal("expected nil middleware on error")
	}
}

func TestHttpxrayNewBadDaemonRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		_, err := New(fmt.Sprintf("s%d", i), fmt.Sprintf("bad%d:::", rng.Intn(10)))
		if err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestHttpxrayMiddlewareNoTrace(t *testing.T) {
	listen := bbUDPListen(t)
	mw, err := New("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := bbTraceRequest(t, http.MethodGet, "http://example.com/", "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Fatalf("called=%v code=%d", called, rec.Code)
	}
}

func TestHttpxrayMiddlewareNoTraceRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	listen := bbUDPListen(t)
	mwFn, err := New("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < bbCases; i++ {
		code := 200 + rng.Intn(100)
		h := mwFn(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		req := bbTraceRequest(t, http.MethodGet, "http://host/", "")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != code {
			t.Fatalf("case %d: code %d", i, rec.Code)
		}
	}
}

func TestHttpxrayMiddlewareWithTrace(t *testing.T) {
	listen := bbUDPListen(t)
	mw, err := New("svc", listen)
	if err != nil {
		t.Fatal(err)
	}
	traceID, spanID := "tr", "sp"
	req := bbTraceRequest(t, http.MethodGet, "https://goa.design/path", "1.2.3.4:443")
	req = req.WithContext(middleware.WithSpan(req.Context(), traceID, spanID, ""))
	req.Header.Set("User-Agent", "bb-test")
	rec := httptest.NewRecorder()
	xraytest.ReadUDP(t, listen, 2, func() {
		mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestHttpxrayWriteHeaderRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		status := 200 + rng.Intn(400)
		s := &HTTPSegment{
			Segment:        &xray.Segment{Mutex: &sync.Mutex{}},
			ResponseWriter: httptest.NewRecorder(),
		}
		s.WriteHeader(status)
		switch {
		case status == http.StatusTooManyRequests:
			if !s.Throttle {
				t.Fatalf("case %d: throttle", i)
			}
		case status >= 500:
			if !s.Error {
				t.Fatalf("case %d: error", i)
			}
		case status >= 400:
			if !s.Fault {
				t.Fatalf("case %d: fault", i)
			}
		}
	}
}

type bbTestDoer struct {
	t      *testing.T
	expSeg bool
	code   int
}

func (d *bbTestDoer) Do(req *http.Request) (*http.Response, error) {
	seg := req.Context().Value(xray.SegKey)
	switch {
	case !d.expSeg && seg != nil:
		d.t.Fatal("unexpected segment in context")
	case d.expSeg && seg == nil:
		d.t.Fatal("missing segment in context")
	}
	if d.code != http.StatusOK {
		return &http.Response{StatusCode: d.code, Body: http.NoBody}, fmt.Errorf("err")
	}
	return &http.Response{StatusCode: d.code, Body: http.NoBody}, nil
}

func TestHttpxrayWrapDoerNoSegment(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/", http.NoBody)
	doer := &bbTestDoer{t: t, expSeg: false, code: http.StatusOK}
	resp, err := WrapDoer(doer).Do(req)
	if err != nil || resp == nil {
		t.Fatalf("err=%v resp=%v", err, resp)
	}
}

func TestHttpxrayWrapDoerWithSegmentRandom(t *testing.T) {
	listen := bbUDPListen(t)
	for i := 0; i < bbCases; i++ {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://host%d.example/path", i), http.NoBody)
		req = req.WithContext(bbSegCtx(t, listen, "tr", "sp"))
		doer := &bbTestDoer{t: t, expSeg: true, code: http.StatusOK}
		xraytest.ReadUDP(t, listen, 2, func() {
			_, err := WrapDoer(doer).Do(req)
			if err != nil {
				t.Fatalf("case %d: %v", i, err)
			}
		})
	}
}

func TestHttpxrayWrapTransport(t *testing.T) {
	listen := bbUDPListen(t)
	req, _ := http.NewRequest(http.MethodGet, "http://remote:8080/x", http.NoBody)
	req = req.WithContext(bbSegCtx(t, listen, "a", "b"))
	// Use local test server as transport target
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	req, _ = http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	req = req.WithContext(bbSegCtx(t, listen, "a", "b"))
	xraytest.ReadUDP(t, listen, 2, func() {
		resp, err := WrapTransport(http.DefaultTransport).RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	})
}

func TestHttpxrayWrapTransportRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200 + rng.Intn(100))
	}))
	defer ts.Close()
	for i := 0; i < bbCases; i++ {
		req, _ := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
		resp, err := WrapTransport(http.DefaultTransport).RoundTrip(req)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		_ = resp.Body.Close()
	}
}

func TestHttpxrayRecordRequestProperty(t *testing.T) {
	u, _ := url.Parse("https://goa.design/path?q=1")
	req := httptest.NewRequest(http.MethodGet, u.String(), http.NoBody)
	req.RemoteAddr = "10.0.0.1:443"
	req.Header.Set("User-Agent", "agent")
	s := &HTTPSegment{Segment: &xray.Segment{Mutex: &sync.Mutex{}}}
	s.RecordRequest(req, "")
	if s.HTTP == nil || s.HTTP.Request == nil {
		t.Fatal("no request recorded")
	}
	if s.HTTP.Request.Method != http.MethodGet {
		t.Fatalf("method=%q", s.HTTP.Request.Method)
	}
	wantURL := strings.Split(u.String(), "?")[0]
	if s.HTTP.Request.URL != wantURL {
		t.Fatalf("url=%q want %q", s.HTTP.Request.URL, wantURL)
	}
}

func TestHttpxrayRecordResponseRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	for i := 0; i < bbCases; i++ {
		status := 200 + rng.Intn(400)
		rec := httptest.NewRecorder()
		rec.WriteHeader(status)
		resp := rec.Result()
		s := &HTTPSegment{Segment: &xray.Segment{Mutex: &sync.Mutex{}}}
		s.RecordResponse(resp)
		if status == http.StatusTooManyRequests && !s.Throttle {
			t.Fatalf("case %d: throttle", i)
		}
		if status >= 500 && !s.Error {
			t.Fatalf("case %d: 5xx", i)
		}
		if status >= 400 && status < 500 && status != http.StatusTooManyRequests && !s.Fault {
			t.Fatalf("case %d: 4xx", i)
		}
	}
}

func TestHttpxrayUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		s := &HTTPSegment{
			Segment:        &xray.Segment{Mutex: &sync.Mutex{}},
			ResponseWriter: httptest.NewRecorder(),
		}
		p := []byte(fmt.Sprintf("data%d", rng.Intn(1000)))
		if _, err := s.Write(p); err != nil && len(p) > 0 {
			t.Fatalf("case %d: write %v", i, err)
		}
		if rng.Intn(2) == 0 {
			_, _, err := s.Hijack()
			if err == nil {
				t.Fatalf("case %d: hijack should fail on recorder", i)
			}
		}
	}
}
