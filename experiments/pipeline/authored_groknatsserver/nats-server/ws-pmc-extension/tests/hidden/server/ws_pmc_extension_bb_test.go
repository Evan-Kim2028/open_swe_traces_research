package server

import (
	"net/http"
	"testing"
)

func wpmc(vals ...string) http.Header {
	return http.Header{"Sec-Websocket-Extensions": vals}
}

// TestDetail01 (yes): the function reads the "Sec-Websocket-Extensions"
// header — Go's canonical MIME key for the websocket extension list.
func TestDetail01(t *testing.T) {
	// Non-canonical spellings canonicalize to the same key in a real
	// request; Header.Get on a Set-stored map proves the lookup key.
	h := http.Header{}
	h.Set("Sec-WebSocket-Extensions", wsPMCExtension)
	pmc, _ := wsPMCExtensionSupport(h, true)
	if !pmc {
		t.Fatal("extension set via the canonical MIME key was not found")
	}
	// A differently-named header must not be consulted.
	h = http.Header{"Sec-Websocket-Extension": {wsPMCExtension}}
	if pmc, _ := wsPMCExtensionSupport(h, true); pmc {
		t.Fatal("a non-matching header name was consulted")
	}
}

// TestDetail02 (yes): each header value is a comma-separated extension list;
// each extension is semicolon-separated into token plus parameters.
func TestDetail02(t *testing.T) {
	h := wpmc("foo, "+wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx+", bar")
	pmc, nct := wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("comma list = (%v, %v), want (true, true)", pmc, nct)
	}
	// Multiple header values are each lists.
	h = wpmc("foo", wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx)
	pmc, nct = wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("multi-value = (%v, %v), want (true, true)", pmc, nct)
	}
}

// TestDetail03 (yes): tokens and parameters are trimmed of space and tab.
func TestDetail03(t *testing.T) {
	h := wpmc("\t " + wsPMCExtension + " \t;\t " + wsPMCSrvNoCtx + "\t ;  " + wsPMCCliNoCtx + "  ")
	pmc, nct := wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("padded extension = (%v, %v), want (true, true)", pmc, nct)
	}
}

// TestDetail04 (yes): the extension token matches "permessage-deflate"
// case-insensitively.
func TestDetail04(t *testing.T) {
	for _, tok := range []string{"permessage-deflate", "PerMessage-Deflate", "PERMESSAGE-DEFLATE"} {
		pmc, _ := wsPMCExtensionSupport(wpmc(tok), true)
		if !pmc {
			t.Fatalf("token %q did not match", tok)
		}
	}
}

// TestDetail05 (yes): when checkPMCOnly is true, a match returns (true,
// false) without inspecting parameters.
func TestDetail05(t *testing.T) {
	// Both parameters present — must still return false for the second flag.
	pmc, nct := wsPMCExtensionSupport(wpmc(wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx), true)
	if !pmc || nct {
		t.Fatalf("checkPMCOnly with full params = (%v, %v), want (true, false)", pmc, nct)
	}
}

// TestDetail06 (shape — Inferable: no): with checkPMCOnly false, only
// parameters after the matching token in that extension are examined —
// tokens and parameters of other extensions are not.
func TestDetail06(t *testing.T) {
	// The no-context names appear as *extension tokens* before the match —
	// they must not count as parameters of the matched extension.
	h := wpmc(wsPMCSrvNoCtx + ", " + wsPMCCliNoCtx + ", " + wsPMCExtension)
	pmc, nct := wsPMCExtensionSupport(h, false)
	if !pmc || nct {
		t.Fatalf("param names as earlier tokens = (%v, %v), want (true, false)", pmc, nct)
	}
	// The parameters attached to a *different* extension do not count.
	h = wpmc("otherext; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx+", "+wsPMCExtension)
	pmc, nct = wsPMCExtensionSupport(h, false)
	if !pmc || nct {
		t.Fatalf("params on a different extension = (%v, %v), want (true, false)", pmc, nct)
	}
	// ...but the same parameter names *after* the token do.
	h = wpmc(wsPMCExtension + "; " + wsPMCSrvNoCtx + "; " + wsPMCCliNoCtx)
	pmc, nct = wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("params after the token = (%v, %v), want (true, true)", pmc, nct)
	}
}

// TestDetail07 (yes): the no-context parameter names match
// case-insensitively.
func TestDetail07(t *testing.T) {
	h := wpmc(wsPMCExtension + "; SERVER_NO_CONTEXT_TAKEOVER; Client_No_Context_Takeover")
	pmc, nct := wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("upper-cased params = (%v, %v), want (true, true)", pmc, nct)
	}
}

// TestDetail08 (yes): both parameters present -> (true, true).
func TestDetail08(t *testing.T) {
	pmc, nct := wsPMCExtensionSupport(wpmc(wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx), false)
	if !pmc || !nct {
		t.Fatalf("both params = (%v, %v), want (true, true)", pmc, nct)
	}
	// Order does not matter.
	pmc, nct = wsPMCExtensionSupport(wpmc(wsPMCExtension+"; "+wsPMCCliNoCtx+"; "+wsPMCSrvNoCtx), false)
	if !pmc || !nct {
		t.Fatalf("reversed params = (%v, %v), want (true, true)", pmc, nct)
	}
}

// TestDetail09 (yes): the extension present but missing one or both
// parameters -> (true, false).
func TestDetail09(t *testing.T) {
	for _, v := range []string{
		wsPMCExtension,
		wsPMCExtension + "; " + wsPMCSrvNoCtx,
		wsPMCExtension + "; " + wsPMCCliNoCtx,
		wsPMCExtension + "; unrelated=1",
	} {
		pmc, nct := wsPMCExtensionSupport(wpmc(v), false)
		if !pmc || nct {
			t.Fatalf("%q = (%v, %v), want (true, false)", v, pmc, nct)
		}
	}
}

// TestDetail10 (yes): no matching extension -> (false, false).
func TestDetail10(t *testing.T) {
	for _, h := range []http.Header{
		{},
		wpmc(""),
		wpmc("gzip", "br; foo=1"),
		wpmc(wsPMCExtension + "x"),        // prefix only — not a match
		wpmc("x" + wsPMCExtension),        // suffix only
		wpmc(wsPMCSrvNoCtx + "; " + wsPMCCliNoCtx),
	} {
		pmc, nct := wsPMCExtensionSupport(h, false)
		if pmc || nct {
			t.Fatalf("%v = (%v, %v), want (false, false)", h, pmc, nct)
		}
	}
}

// TestDetail11 (partially): the first matching permessage-deflate extension
// decides — a second match later in the list is not consulted.
func TestDetail11(t *testing.T) {
	// First match lacks the params, a later entry has them both: still
	// (true, false).
	h := wpmc(wsPMCExtension, wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx)
	pmc, nct := wsPMCExtensionSupport(h, false)
	if !pmc || nct {
		t.Fatalf("first match without params = (%v, %v), want (true, false)", pmc, nct)
	}
	// Same order within a single header value.
	h = wpmc(wsPMCExtension + ", " + wsPMCExtension + "; " + wsPMCSrvNoCtx + "; " + wsPMCCliNoCtx)
	pmc, nct = wsPMCExtensionSupport(h, false)
	if !pmc || nct {
		t.Fatalf("first match in same value = (%v, %v), want (true, false)", pmc, nct)
	}
	// And the reverse: first match decides positively.
	h = wpmc(wsPMCExtension+"; "+wsPMCSrvNoCtx+"; "+wsPMCCliNoCtx+", "+wsPMCExtension)
	pmc, nct = wsPMCExtensionSupport(h, false)
	if !pmc || !nct {
		t.Fatalf("first match with params = (%v, %v), want (true, true)", pmc, nct)
	}
}
