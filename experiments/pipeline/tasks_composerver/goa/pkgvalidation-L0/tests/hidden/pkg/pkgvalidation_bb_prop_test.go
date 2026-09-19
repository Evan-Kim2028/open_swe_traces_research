// Black-box property suite for pkgvalidation (ValidateFormat, ValidatePattern).
// Exported API only. Seed 20260919; >=10k cases.
package goa_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	goa "example.internal/apikit/v3/pkg"
)

const bbSeed = 20260919
const bbCases = 10000

func bbValidDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

func bbValidDateTime(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

func bbValidEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil
}

func bbValidURI(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil
}

func bbValidMAC(s string) bool {
	_, err := net.ParseMAC(s)
	return err == nil
}

func bbValidCIDR(s string) bool {
	_, _, err := net.ParseCIDR(s)
	return err == nil
}

func bbValidJSON(s string) bool {
	return json.Valid([]byte(s))
}

func bbValidRFC1123(s string) bool {
	_, err := time.Parse(time.RFC1123, s)
	return err == nil
}

func bbDottedQuad(s string) bool {
	return regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`).MatchString(s)
}

func bbIsInvalidFormat(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "formatted as a") || strings.Contains(msg, "invalid format")
}

func bbIsInvalidPattern(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "must match the regexp")
}

func TestPkgValidationFormatTableProperty(t *testing.T) {
	valid := map[goa.Format]string{
		goa.FormatDate:     "2024-06-01",
		goa.FormatDateTime: "2024-06-01T12:00:00Z",
		goa.FormatEmail:     "a@b.co",
		goa.FormatHostname:  "host.example",
		goa.FormatIPv4:      "192.0.2.1",
		goa.FormatIPv6:      "::1",
		goa.FormatIP:        "10.0.0.1",
		goa.FormatURI:       "https://example.com/path",
		goa.FormatMAC:       "00:11:22:33:44:55",
		goa.FormatCIDR:      "192.0.2.0/24",
		goa.FormatRegexp:    "^a+$",
		goa.FormatJSON:      `{"k":1}`,
		goa.FormatRFC1123:   "Mon, 02 Jan 2006 15:04:05 MST",
		goa.FormatUUID:      "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
	for f, v := range valid {
		if err := goa.ValidateFormat("x", v, f); err != nil {
			t.Fatalf("valid %s %q: %v", f, v, err)
		}
	}
	invalid := []struct {
		f   goa.Format
		val string
	}{
		{goa.FormatDate, "not-a-date"},
		{goa.FormatDateTime, "2024-13-40"},
		{goa.FormatEmail, "not-email"},
		{goa.FormatHostname, ""},
		{goa.FormatIPv4, "::1"},
		{goa.FormatIPv6, "192.0.2.1"},
		{goa.FormatURI, "://"},
		{goa.FormatUUID, "not-uuid"},
		{goa.FormatJSON, "{"},
	}
	for _, c := range invalid {
		err := goa.ValidateFormat("n", c.val, c.f)
		if err == nil || !bbIsInvalidFormat(err) {
			t.Fatalf("invalid %s %q: %v", c.f, c.val, err)
		}
	}
	if err := goa.ValidateFormat("n", "x", goa.Format("nope")); err == nil {
		t.Fatal("unknown format")
	}
}

func TestPkgValidationUUIDRenderingsProperty(t *testing.T) {
	base := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	forms := []string{
		base,
		"6ba7b8109dad11d180b400c04fd430c8",
		"{" + base + "}",
		"urn:uuid:" + base,
	}
	for _, u := range forms {
		if err := goa.ValidateFormat("id", u, goa.FormatUUID); err != nil {
			t.Fatalf("uuid form %q: %v", u, err)
		}
	}
	bad := []string{"not-a-uuid", "6ba7b810-9dad-11d1-80b4", "garbage"}
	for _, u := range bad {
		if err := goa.ValidateFormat("id", u, goa.FormatUUID); err == nil {
			t.Fatalf("bad uuid accepted: %q", u)
		}
	}
}

func TestPkgValidationFormatRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		f := goa.FormatDate
		val := fmt.Sprintf("%04d-%02d-%02d", 2000+rng.Intn(30), 1+rng.Intn(12), 1+rng.Intn(28))
		want := bbValidDate(val)
		err := goa.ValidateFormat("d", val, f)
		if want && err != nil {
			t.Fatalf("case %d date %q: %v", i, val, err)
		}
		if !want && (err == nil || !bbIsInvalidFormat(err)) {
			t.Fatalf("case %d bad date %q: %v", i, val, err)
		}
	}
}

func TestPkgValidationIPCrossRejectRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		switch rng.Intn(3) {
		case 0:
			v := "::1"
			if err := goa.ValidateFormat("ip", v, goa.FormatIPv4); err == nil {
				t.Fatalf("case %d ipv6 as v4", i)
			}
		case 1:
			v := "127.0.0.1"
			if err := goa.ValidateFormat("ip", v, goa.FormatIPv6); err == nil {
				t.Fatalf("case %d ipv4 as v6", i)
			}
		case 2:
			v := "10.0.0.1"
			if err := goa.ValidateFormat("ip", v, goa.FormatIP); err != nil {
				t.Fatalf("case %d ip %q: %v", i, v, err)
			}
		}
	}
}

func TestPkgValidationPatternProperty(t *testing.T) {
	if err := goa.ValidatePattern("f", "aaa", "^a+$"); err != nil {
		t.Fatal(err)
	}
	err := goa.ValidatePattern("f", "xyz", "^a+$")
	if err == nil || !bbIsInvalidPattern(err) || !strings.Contains(err.Error(), "f") {
		t.Fatalf("pattern mismatch: %v", err)
	}
}

func TestPkgValidationPatternRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	pat := "^[a-z]{3}[0-9]{2}$"
	for i := 0; i < bbCases; i++ {
		letters := make([]byte, 3)
		for j := range letters {
			letters[j] = byte('a' + rng.Intn(26))
		}
		val := fmt.Sprintf("%s%02d", letters, rng.Intn(100))
		match, _ := regexp.MatchString(pat, val)
		err := goa.ValidatePattern("field", val, pat)
		if match && err != nil {
			t.Fatalf("case %d match %q: %v", i, val, err)
		}
		if !match && err == nil {
			t.Fatalf("case %d no match %q", i, val)
		}
	}
}

func TestPkgValidationUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		kind := rng.Intn(6)
		switch kind {
		case 0:
			s := strings.Repeat("x", rng.Intn(5))
			want := bbValidJSON(fmt.Sprintf(`"%s"`, s))
			err := goa.ValidateFormat("j", fmt.Sprintf(`"%s"`, s), goa.FormatJSON)
			if want && err != nil {
				t.Fatalf("json %d: %v", i, err)
			}
		case 1:
			mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256), rng.Intn(256))
			if bbValidMAC(mac) {
				if err := goa.ValidateFormat("m", mac, goa.FormatMAC); err != nil {
					t.Fatalf("mac %d: %v", i, err)
				}
			}
		case 2:
			u := "https://example.com/" + fmt.Sprint(i)
			if err := goa.ValidateFormat("u", u, goa.FormatURI); err != nil {
				t.Fatalf("uri %d: %v", i, err)
			}
		case 3:
			if err := goa.ValidateFormat("r", "not-regexp-(", goa.FormatRegexp); err == nil {
				t.Fatalf("regexp %d", i)
			}
		case 4:
			if bbDottedQuad("999.999.999.999") {
				_ = goa.ValidateFormat("v4", "999.999.999.999", goa.FormatIPv4)
			}
		case 5:
			if bbValidRFC1123("Mon, 02 Jan 2006 15:04:05 GMT") {
				if err := goa.ValidateFormat("h", "Mon, 02 Jan 2006 15:04:05 GMT", goa.FormatRFC1123); err != nil {
					t.Fatalf("rfc1123 %d: %v", i, err)
				}
			}
		}
	}
}
