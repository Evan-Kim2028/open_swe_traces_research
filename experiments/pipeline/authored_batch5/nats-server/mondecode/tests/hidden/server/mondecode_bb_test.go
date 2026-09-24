package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func mdBbReq(target string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), httptest.NewRequest("GET", target, nil)
}

// TestDetail01: decodeBool/decodeUint64/decodeInt — absent param is not an
// error; parse failure writes HTTP 400 naming the param and returns the error.
func TestDetail01(t *testing.T) {
	w, r := mdBbReq("/x")
	if v, err := decodeBool(w, r, "p"); v || err != nil {
		t.Fatalf("absent bool = %v, %v", v, err)
	}
	w, r = mdBbReq("/x?p=1")
	if v, err := decodeBool(w, r, "p"); !v || err != nil {
		t.Fatalf("bool '1' = %v, %v", v, err)
	}
	w, r = mdBbReq("/x?p=garbage")
	if v, err := decodeBool(w, r, "p"); err == nil || v {
		t.Fatalf("bad bool = %v, %v", v, err)
	} else if w.Code != 400 {
		t.Fatalf("bad bool wrote %d, want 400", w.Code)
	} else if !strings.Contains(w.Body.String(), "'p'") {
		t.Fatalf("error body should name the param, got %q", w.Body.String())
	}

	w, r = mdBbReq("/x?p=42")
	if v, err := decodeUint64(w, r, "p"); v != 42 || err != nil {
		t.Fatalf("u64 = %v, %v", v, err)
	}
	w, r = mdBbReq("/x")
	if v, err := decodeUint64(w, r, "p"); v != 0 || err != nil {
		t.Fatalf("absent u64 = %v, %v", v, err)
	}
	w, r = mdBbReq("/x?p=-1")
	if _, err := decodeUint64(w, r, "p"); err == nil || w.Code != 400 {
		t.Fatalf("bad u64: err=%v code=%d", err, w.Code)
	}

	w, r = mdBbReq("/x?p=-7")
	if v, err := decodeInt(w, r, "p"); v != -7 || err != nil {
		t.Fatalf("int = %v, %v", v, err)
	}
	w, r = mdBbReq("/x?p=abc")
	if _, err := decodeInt(w, r, "p"); err == nil || w.Code != 400 {
		t.Fatalf("bad int: err=%v code=%d", err, w.Code)
	}
	if !strings.Contains(w.Body.String(), "'p'") {
		t.Fatalf("int error body should name the param, got %q", w.Body.String())
	}
}

// TestDetail02: decodeState — absent → ConnOpen; open/closed/any/all matched
// case-insensitively with any/all synonymous; anything else → 400 + error.
func TestDetail02(t *testing.T) {
	w, r := mdBbReq("/x")
	if st, err := decodeState(w, r); st != ConnOpen || err != nil {
		t.Fatalf("absent = %v, %v", st, err)
	}
	for q, want := range map[string]ConnState{
		"?state=open": ConnOpen, "?state=OPEN": ConnOpen,
		"?state=closed": ConnClosed, "?state=CLOSED": ConnClosed,
		"?state=any": ConnAll, "?state=ANY": ConnAll,
		"?state=all": ConnAll, "?state=ALL": ConnAll,
	} {
		w, r := mdBbReq("/x" + q)
		if st, err := decodeState(w, r); st != want || err != nil {
			t.Fatalf("%s = %v, %v", q, st, err)
		}
	}
	w, r = mdBbReq("/x?state=bogus")
	st, err := decodeState(w, r)
	if err == nil || w.Code != 400 || st != 0 {
		t.Fatalf("bogus state: st=%v err=%v code=%d", st, err, w.Code)
	}
}

// TestDetail03: decodeSubs — subs=detail (case-insensitive) sets subsDet and
// skips bool decoding; other values are bool-decoded (garbage → 400).
func TestDetail03(t *testing.T) {
	w, r := mdBbReq("/x?subs=detail")
	if s, d, err := decodeSubs(w, r); s || !d || err != nil {
		t.Fatalf("detail: %v %v %v", s, d, err)
	}
	w, r = mdBbReq("/x?subs=DETAIL")
	if s, d, err := decodeSubs(w, r); s || !d || err != nil {
		t.Fatalf("DETAIL: %v %v %v", s, d, err)
	}
	w, r = mdBbReq("/x?subs=true")
	if s, d, err := decodeSubs(w, r); !s || d || err != nil {
		t.Fatalf("true: %v %v %v", s, d, err)
	}
	w, r = mdBbReq("/x")
	if s, d, err := decodeSubs(w, r); s || d || err != nil {
		t.Fatalf("absent: %v %v %v", s, d, err)
	}
	w, r = mdBbReq("/x?subs=garbage")
	if _, _, err := decodeSubs(w, r); err == nil || w.Code != 400 {
		t.Fatalf("garbage: err=%v code=%d", err, w.Code)
	}
}

// TestDetail04: myUptime renders the largest non-zero unit first and emits all
// lower units once one appears; years are days/365.
func TestDetail04(t *testing.T) {
	cases := map[time.Duration]string{
		0:                                    "0s",
		22 * time.Second:                     "22s",
		4*time.Minute + 22*time.Second:       "4m22s",
		4*time.Hour + 4*time.Minute + 22*time.Second: "4h4m22s",
		32*24*time.Hour + 4*time.Hour + 4*time.Minute + 22*time.Second:                     "32d4h4m22s",
		22*365*24*time.Hour + 32*24*time.Hour + 4*time.Hour + 4*time.Minute + 22*time.Second: "22y32d4h4m22s",
		time.Hour:  "1h0m0s",
		time.Minute: "1m0s",
	}
	for d, want := range cases {
		if got := myUptime(d); got != want {
			t.Fatalf("myUptime(%v) = %q, want %q", d, got, want)
		}
	}
}

// TestDetail05: redactBearerJWT — empty in, empty out; undecodable or
// non-bearer input returns the original; a decodable bearer user JWT returns
// empty.
func TestDetail05(t *testing.T) {
	if got := redactBearerJWT(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := redactBearerJWT("notajwt"); got != "notajwt" {
		t.Fatalf("undecodable = %q", got)
	}

	ukp, err := nkeys.CreateUser()
	if err != nil {
		t.Fatal(err)
	}
	upub, _ := ukp.PublicKey()

	uc := jwt.NewUserClaims(upub)
	uc.BearerToken = true
	ujwt, err := uc.Encode(ukp)
	if err != nil {
		t.Fatal(err)
	}
	if got := redactBearerJWT(ujwt); got != "" {
		t.Fatalf("bearer JWT should redact to empty, got %q", got)
	}

	uc2 := jwt.NewUserClaims(upub)
	uc2.BearerToken = false
	ujwt2, err := uc2.Encode(ukp)
	if err != nil {
		t.Fatal(err)
	}
	if got := redactBearerJWT(ujwt2); got != ujwt2 {
		t.Fatalf("non-bearer JWT must be returned unchanged")
	}
}
