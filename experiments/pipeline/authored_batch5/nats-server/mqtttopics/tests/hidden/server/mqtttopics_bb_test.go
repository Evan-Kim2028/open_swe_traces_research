package server

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestDetail01: MQTT→subject conversion — '/'→'.', '.'→'//', '+'→'*', '#'→'>',
// with empty levels preserved ('/foo'→'/.foo', 'foo/'→'foo./',
// 'foo//bar'→'foo./.bar').
func TestDetail01(t *testing.T) {
	cases := map[string]string{
		"foo/bar":   "foo.bar",
		"foo.bar":   "foo//bar",
		"a/b/c":     "a.b.c",
		"/foo":      "/.foo",
		"foo/":      "foo./",
		"foo//bar":  "foo./.bar",
		"//":        "/././",
		"/a/":       "/.a./",
		"foo/+":     "foo.*",
		"foo/#":     "foo.>",
		"a.b/c":     "a//b.c",
		"a/+/b":     "a.*.b",
		"foo.bar.*": "foo//bar//*", // '*' is a literal byte in MQTT, not a wildcard
	}
	for in, want := range cases {
		got, err := mqttToNATSSubjectConversion([]byte(in), true)
		if err != nil {
			t.Fatalf("conv(%q): %v", in, err)
		}
		if string(got) != want {
			t.Fatalf("conv(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDetail02: reverse mapping collapses '//' back to '.' and '/.' pairs
// back to '/' via left-to-right two-character lookahead.
func TestDetail02(t *testing.T) {
	cases := map[string]string{
		"foo.bar":    "foo/bar",
		"foo//bar":   "foo.bar",
		"foo./.bar":  "foo//bar",
		"/.foo":      "/foo",
		"foo./":      "foo/",
		"a//b.c":     "a.b/c",
		"foo.bar.*":  "foo/bar/*",
		"foo.>":      "foo/>",
		"a..b":       "a//b",
	}
	for in, want := range cases {
		if got := natsSubjectToMQTTTopic([]byte(in)); string(got) != want {
			t.Fatalf("rev(%q) = %q, want %q", in, got, want)
		}
		if got := natsSubjectStrToMQTTTopic(in); string(got) != want {
			t.Fatalf("revStr(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDetail03: whitespace and DEL reject with the unsupported-characters
// error in both topics and filters.
func TestDetail03(t *testing.T) {
	for _, s := range []string{"foo bar", "foo\tbar", "foo\nbar", "foo\rbar", "foo\fbar", "foo\x7fbar"} {
		if _, err := mqttToNATSSubjectConversion([]byte(s), true); !errors.Is(err, errMQTTUnsupportedCharacters) {
			t.Fatalf("conv(%q, filter): %v", s, err)
		}
		if _, err := mqttToNATSSubjectConversion([]byte(s), false); !errors.Is(err, errMQTTUnsupportedCharacters) {
			t.Fatalf("conv(%q, publish): %v", s, err)
		}
	}
}

// TestDetail04: wildcards are rejected in publish topics with an error naming
// the topic; in filters they map at any position.
func TestDetail04(t *testing.T) {
	for _, s := range []string{"foo/+", "a/#/b", "a+b", "#", "+"} {
		_, err := mqttToNATSSubjectConversion([]byte(s), false)
		if err == nil || !strings.Contains(err.Error(), s) {
			t.Fatalf("publish %q: want error naming the topic, got %v", s, err)
		}
	}
	// In filters, wildcards convert at positions MQTT itself would reject.
	for in, want := range map[string]string{"foo/+bar": "foo.*bar", "a/#/b": "a.>.b", "+": "*", "#": ">"} {
		got, err := mqttToNATSSubjectConversion([]byte(in), true)
		if err != nil || string(got) != want {
			t.Fatalf("filter %q = %q, %v; want %q", in, got, err, want)
		}
	}
	// Wrappers share the engine.
	if _, err := mqttTopicToNATSPubSubject([]byte("a/b")); err != nil {
		t.Fatal(err)
	}
	if _, err := mqttTopicToNATSPubSubject([]byte("a/+")); err == nil {
		t.Fatal("publish wrapper must reject wildcards")
	}
	if got, err := mqttFilterToNATSSubject([]byte("a/+")); err != nil || string(got) != "a.*" {
		t.Fatalf("filter wrapper: %q %v", got, err)
	}
}

// TestDetail05: when nothing needs converting the input slice is returned
// unallocated (same backing array).
func TestDetail05(t *testing.T) {
	in := []byte("foobar")
	out, err := mqttToNATSSubjectConversion(in, true)
	if err != nil {
		t.Fatal(err)
	}
	if &in[0] != &out[0] {
		t.Fatal("no-conversion input must be returned without allocation")
	}
	if string(out) != "foobar" {
		t.Fatalf("got %q", out)
	}
}

// TestDetail06: mqttNeedSubForLevelUp is true iff len>=3 and subject ends ".>".
func TestDetail06(t *testing.T) {
	for _, s := range []string{"a.>", "a.b.>", "a.*.>"} {
		if !mqttNeedSubForLevelUp(s) {
			t.Fatalf("%q should need a level-up sub", s)
		}
	}
	for _, s := range []string{">", ".>", "a.b", "a>", "a.>.x"} {
		if mqttNeedSubForLevelUp(s) {
			t.Fatalf("%q should not need a level-up sub", s)
		}
	}
}

// TestDetail07: mqttValidateTopic/mqttValidateString reject only invalid UTF-8
// and embedded NUL; empty and wildcard spellings pass.
func TestDetail07(t *testing.T) {
	for _, b := range [][]byte{[]byte(""), []byte("foo"), []byte("foo/bar"), []byte("foo#"), []byte("a +"), []byte("a b")} {
		if err := mqttValidateTopic(b, "t"); err != nil {
			t.Fatalf("validateTopic(%q): %v", b, err)
		}
	}
	if err := mqttValidateTopic([]byte("a\x00b"), "t"); err == nil {
		t.Fatal("embedded NUL must be rejected")
	}
	if err := mqttValidateTopic([]byte("a\xffb"), "t"); err == nil {
		t.Fatal("invalid UTF-8 must be rejected")
	}
	if err := mqttValidateString("", "f"); err != nil {
		t.Fatal(err)
	}
	if err := mqttValidateString("a\x00b", "f"); err == nil {
		t.Fatal("embedded NUL must be rejected")
	}
	if err := mqttValidateString("a\xffb", "f"); err == nil {
		t.Fatal("invalid UTF-8 must be rejected")
	}
}

// TestDetail08: isMQTTReservedSubscription matches the catch-all and
// first-level-wildcard spellings; mqttMustIgnoreForReservedSub drops
// '$'-subjects only for those subscriptions.
func TestDetail08(t *testing.T) {
	for _, s := range []string{">", "*", "*.", "*.foo", "*.>"} {
		if !isMQTTReservedSubscription(s) {
			t.Fatalf("%q should be a reserved subscription", s)
		}
	}
	for _, s := range []string{"foo.*", "foo.>", "a.#", "", "#."} {
		if isMQTTReservedSubscription(s) {
			t.Fatalf("%q should not be a reserved subscription", s)
		}
	}

	rsv := &subscription{mqtt: &mqttSub{reserved: true}}
	plain := &subscription{mqtt: &mqttSub{}}
	if !mqttMustIgnoreForReservedSub(rsv, "$SYS.foo") {
		t.Fatal("reserved sub must ignore $-prefixed subjects")
	}
	if mqttMustIgnoreForReservedSub(rsv, "normal.subject") {
		t.Fatal("reserved sub must still receive normal subjects")
	}
	if mqttMustIgnoreForReservedSub(plain, "$SYS.foo") {
		t.Fatal("non-reserved sub must receive $-prefixed subjects")
	}
}

// TestDetail09: sparkbParseBirthDeathTopic recognises spBv1.0 group/type topics
// of 3-4 parts (optionally under $sparkplug/certificates/); only the
// NBIRTH/DBIRTH/NDEATH/DDEATH type tokens count.
func TestDetail09(t *testing.T) {
	cases := []struct {
		topic                    string
		birth, death, cert       bool
	}{
		{"spBv1.0/g/NBIRTH/e", true, false, false},
		{"spBv1.0/g/DBIRTH/e/d", true, false, false},
		{"spBv1.0/g/NDEATH/e", false, true, false},
		{"spBv1.0/g/DDEATH/e/d", false, true, false},
		{"spBv1.0/g/NDATA/e", false, false, false},
		{"spBv1.0/g/NBIRTH", false, false, false},
		{"spBv1.0/g/NBIRTH/e/x/y", false, false, false},
		{"other/g/NBIRTH/e", false, false, false},
		{"$sparkplug/certificates/spBv1.0/g/NBIRTH/e", true, false, true},
		{"$sparkplug/certificates/spBv1.0/g/DBIRTH/e/d", true, false, true},
		{"$sparkplug/certificates/foo", false, false, false},
	}
	for _, tc := range cases {
		b, d, c := sparkbParseBirthDeathTopic([]byte(tc.topic))
		if b != tc.birth || d != tc.death || c != tc.cert {
			t.Fatalf("%q = birth:%v death:%v cert:%v, want %v %v %v", tc.topic, b, d, c, tc.birth, tc.death, tc.cert)
		}
	}
}

// TestDetail10: sparkbReplaceDeathTimestamp — on a scan error returns the
// original buffer; substitutes field 1 when present; appends a timestamp
// field when absent.
func TestDetail10(t *testing.T) {
	// Scan error → the original buffer is returned.
	garbage := []byte{0xff, 0xff, 0xff}
	if out := sparkbReplaceDeathTimestamp(garbage); !bytes.Equal(out, garbage) {
		t.Fatalf("garbage input must return the original buffer, got %x", out)
	}
	// No field 1 → a field-1 timestamp entry is appended after the original.
	noF1 := []byte{0x10, 0x05} // field 2, varint 5
	out := sparkbReplaceDeathTimestamp(noF1)
	if len(out) <= len(noF1) || !bytes.Equal(out[:len(noF1)], noF1) {
		t.Fatalf("missing field 1 should be appended after original, got %x", out)
	}
	if out[len(noF1)] != 0x08 {
		t.Fatalf("appended entry should be field 1 varint, got %x", out[len(noF1):])
	}
	// Field 1 present → substituted; other fields preserved.
	withF1 := []byte{0x08, 0x7b, 0x10, 0x05} // field1=123, field2=5
	out2 := sparkbReplaceDeathTimestamp(withF1)
	if !bytes.HasPrefix(out2, []byte{0x08}) {
		t.Fatalf("field 1 must still lead, got %x", out2)
	}
	if !bytes.HasSuffix(out2, []byte{0x10, 0x05}) {
		t.Fatalf("field 2 must be preserved, got %x", out2)
	}
	if bytes.Equal(out2, withF1) {
		t.Fatal("timestamp should have been substituted")
	}
}
