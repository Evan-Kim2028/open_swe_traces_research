package server

import (
	"bytes"
	"errors"
	"testing"
)

// TestDetail01: encoded record = NATS/1.0 header block then raw payload;
// header order RTopic, RFlags, optional ROrigin, optional RSource; flags are
// lowercase hex.
func TestDetail01(t *testing.T) {
	rm := &mqttRetainedMsg{Topic: "a/b", Msg: []byte("hello"), Flags: 0x01, Origin: "O", Source: "S"}
	msg, hl := mqttEncodeRetainedMessage(rm)

	if !bytes.HasPrefix(msg, []byte("NATS/1.0\r\n")) {
		t.Fatalf("record must start with NATS/1.0 status line: %q", msg)
	}
	if string(msg[hl:]) != "hello" {
		t.Fatalf("payload after headerLen = %q", msg[hl:])
	}
	if !bytes.HasSuffix(msg[:hl], []byte("\r\n\r\n")) {
		t.Fatalf("header must end with a blank line: %q", msg[:hl])
	}
	// Header order: Topic, Flags, Origin, Source.
	h := msg[:hl]
	it, if_, io, is := bytes.Index(h, []byte("Nmqtt-RTopic:")), bytes.Index(h, []byte("Nmqtt-RFlags:")),
		bytes.Index(h, []byte("Nmqtt-ROrigin:")), bytes.Index(h, []byte("Nmqtt-RSource:"))
	if it < 0 || if_ < 0 || io < 0 || is < 0 || !(it < if_ && if_ < io && io < is) {
		t.Fatalf("header order wrong: %q", h)
	}
	// Lowercase hex flags.
	rm2 := &mqttRetainedMsg{Topic: "t", Msg: []byte("x"), Flags: 0xab}
	msg2, hl2 := mqttEncodeRetainedMessage(rm2)
	if !bytes.Contains(msg2[:hl2], []byte("Nmqtt-RFlags:ab\r\n")) {
		t.Fatalf("flags must render lowercase hex: %q", msg2[:hl2])
	}
	// Optional headers absent when fields unset.
	if bytes.Contains(msg2[:hl2], []byte("Nmqtt-ROrigin")) || bytes.Contains(msg2[:hl2], []byte("Nmqtt-RSource")) {
		t.Fatalf("optional headers must be omitted: %q", msg2[:hl2])
	}
}

// TestDetail02: a retained-delete record (empty Msg) emits a '-' marker byte
// before the hex flags.
func TestDetail02(t *testing.T) {
	rm := &mqttRetainedMsg{Topic: "a/b", Flags: 0x01}
	msg, hl := mqttEncodeRetainedMessage(rm)
	if !bytes.Contains(msg[:hl], []byte("Nmqtt-RFlags:-1\r\n")) {
		t.Fatalf("delete record should carry '-' marker before flags: %q", msg[:hl])
	}
	if len(msg) != hl {
		t.Fatalf("delete record should have no payload, got %q", msg[hl:])
	}
	// A record with a payload has no marker.
	rm2 := &mqttRetainedMsg{Topic: "a/b", Msg: []byte("x"), Flags: 0x01}
	msg2, hl2 := mqttEncodeRetainedMessage(rm2)
	if bytes.Contains(msg2[:hl2], []byte("Nmqtt-RFlags:-")) {
		t.Fatalf("non-delete record must not carry marker: %q", msg2[:hl2])
	}
}

// TestDetail03: mqttSliceHeaders — key ends at first ':' with spaces before it
// walked back; empty key stops the parse; leading spaces stripped from value;
// slices are capacity-limited; only caller-map keys recorded.
func TestDetail03(t *testing.T) {
	hdr := []byte("NATS/1.0\r\nNmqtt-RTopic :   a/b \r\nOther: z\r\nNmqtt-RFlags: 1\r\n\r\n")
	m := map[string][]byte{"Nmqtt-RTopic": nil, "Nmqtt-RFlags": nil}
	mqttSliceHeaders(m, hdr)
	if string(m["Nmqtt-RTopic"]) != "a/b " {
		t.Fatalf("value should keep trailing space but strip leading: %q", m["Nmqtt-RTopic"])
	}
	if string(m["Nmqtt-RFlags"]) != "1" {
		t.Fatalf("flags = %q", m["Nmqtt-RFlags"])
	}
	if _, ok := m["Other"]; ok {
		t.Fatal("keys not in the caller map must not be recorded")
	}
	if cap(m["Nmqtt-RFlags"]) != len(m["Nmqtt-RFlags"]) {
		t.Fatalf("stored slice must be capacity-limited: cap=%d len=%d", cap(m["Nmqtt-RFlags"]), len(m["Nmqtt-RFlags"]))
	}
	// An empty key stops the parse.
	m2 := map[string][]byte{"Nmqtt-RFlags": nil}
	mqttSliceHeaders(m2, []byte("NATS/1.0\r\n: x\r\nNmqtt-RFlags: 9\r\n\r\n"))
	if m2["Nmqtt-RFlags"] != nil {
		t.Fatalf("empty key must stop the parse, got %q", m2["Nmqtt-RFlags"])
	}
}

// TestDetail04: mqttDecodeRetainedMessage slices only the headers it needs and
// falls back to legacy JSON when RFlags is absent; a nil decode yields
// errMQTTInvalidRetainedMessage.
func TestDetail04(t *testing.T) {
	subj := "$MQTT.rmsgs.a.b"
	rm := &mqttRetainedMsg{Topic: "a/b", Msg: []byte("hi"), Flags: 0x01}
	msg, hl := mqttEncodeRetainedMessage(rm)
	dec, err := mqttDecodeRetainedMessage(subj, msg[:hl], msg[hl:])
	if err != nil {
		t.Fatalf("round-trip decode: %v", err)
	}
	if dec.Subject != "a.b" || dec.Topic != "a/b" || string(dec.Msg) != "hi" || dec.Flags != 0x01 {
		t.Fatalf("decoded = %+v", dec)
	}

	// Legacy JSON fallback when RFlags is absent.
	legacy := []byte("NATS/1.0\r\n\r\n{\"origin\":\"O\",\"subject\":\"a.b\",\"topic\":\"a/b\",\"msg\":\"aGk=\",\"flags\":2}")
	dec2, err := mqttDecodeRetainedMessage(subj, legacy[:12], legacy[12:])
	if err != nil || dec2.Flags != 2 || string(dec2.Msg) != "hi" {
		t.Fatalf("legacy JSON decode: %+v %v", dec2, err)
	}
	// A null JSON decode yields errMQTTInvalidRetainedMessage.
	jnull := []byte("NATS/1.0\r\n\r\nnull")
	if _, err := mqttDecodeRetainedMessage(subj, jnull[:12], jnull[12:]); !errors.Is(err, errMQTTInvalidRetainedMessage) {
		t.Fatalf("null JSON should yield invalid-retained-message, got %v", err)
	}
	// Undecodable JSON is an error.
	jbad := []byte("NATS/1.0\r\n\r\n{not json}")
	if _, err := mqttDecodeRetainedMessage(subj, jbad[:12], jbad[12:]); err == nil {
		t.Fatal("garbage JSON must error")
	}
}

// TestDetail05: flags — a leading '-' is stripped, the rest is base-16; parse
// failure, flags >= mqttPacketFlagMask, or QoS > 2 → errMQTTInvalidRetainFlags.
func TestDetail05(t *testing.T) {
	subj := "$MQTT.rmsgs.a.b"
	decode := func(flags string) error {
		h := []byte("NATS/1.0\r\nNmqtt-RFlags: " + flags + "\r\n\r\n")
		_, err := mqttDecodeRetainedMessage(subj, h, []byte("x"))
		return err
	}
	if err := decode("zz"); !errors.Is(err, errMQTTInvalidRetainFlags) {
		t.Fatalf("unparseable flags: %v", err)
	}
	if err := decode("6"); !errors.Is(err, errMQTTInvalidRetainFlags) {
		t.Fatalf("qos=3 must be invalid: %v", err)
	}
	if err := decode("f"); !errors.Is(err, errMQTTInvalidRetainFlags) {
		t.Fatalf("flags >= mask must be invalid: %v", err)
	}
	if err := decode("4"); err != nil {
		t.Fatalf("qos=2 flags must be valid: %v", err)
	}
	// Delete marker strips '-' then validates the remainder.
	if err := decode("-4"); err != nil {
		t.Fatalf("delete marker with valid flags: %v", err)
	}
	if err := decode("-2b"); !errors.Is(err, errMQTTInvalidRetainFlags) {
		t.Fatalf("delete marker with invalid flags: %v", err)
	}
}

// TestDetail06: rm.Subject is the stream subject minus the $MQTT.rmsgs.
// prefix; rm.Topic is reconstructed from that subject, never from headers.
func TestDetail06(t *testing.T) {
	subj := "$MQTT.rmsgs.a.b.c"
	h := []byte("NATS/1.0\r\nNmqtt-RTopic:WRONG\r\nNmqtt-RFlags: 1\r\n\r\n")
	rm, err := mqttDecodeRetainedMessage(subj, h, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if rm.Subject != "a.b.c" {
		t.Fatalf("Subject = %q", rm.Subject)
	}
	if rm.Topic != "a/b/c" {
		t.Fatalf("Topic must be reconstructed from subject, got %q", rm.Topic)
	}
}

// TestDetail07: rm.Msg borrows the payload slice — no copy.
func TestDetail07(t *testing.T) {
	subj := "$MQTT.rmsgs.a.b"
	h := []byte("NATS/1.0\r\nNmqtt-RFlags: 1\r\n\r\n")
	m := []byte("payload")
	rm, err := mqttDecodeRetainedMessage(subj, h, m)
	if err != nil {
		t.Fatal(err)
	}
	m[0] = 'X'
	if rm.Msg[0] != 'X' {
		t.Fatalf("rm.Msg must borrow m, got %q", rm.Msg)
	}
}
