// Hidden black-box property suite for the jsversioning unit.
// Drives only the API in api.md: get/supports/set/copy/delete metadata
// helpers and errorOnRequiredApiLevel, over StreamConfig/ConsumerConfig/
// ConsumerInfo values built in-test.
// One property per DETAILS.md line; seeded random inputs, no worked examples.
package server

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

const bbJsvHiddenSeed = 20260919

func bbJsvSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return bbJsvHiddenSeed
}

func bbJsvRng(t *testing.T) *rand.Rand {
	t.Helper()
	return rand.New(rand.NewSource(bbJsvSeed()))
}

// Detail 1: getRequiredApiLevel returns the marker string, "" when absent
// OR empty.
func TestDetail01_GetRequiredApiLevel(t *testing.T) {
	if got := getRequiredApiLevel(nil); got != "" {
		t.Fatalf("nil map=%q", got)
	}
	if got := getRequiredApiLevel(map[string]string{}); got != "" {
		t.Fatalf("empty map=%q", got)
	}
	if got := getRequiredApiLevel(map[string]string{JSRequiredLevelMetadataKey: ""}); got != "" {
		t.Fatalf("empty marker=%q want \"\"", got)
	}
	rng := bbJsvRng(t)
	for i := 0; i < 20; i++ {
		v := strconv.Itoa(rng.Intn(100))
		m := map[string]string{JSRequiredLevelMetadataKey: v, "other": "x"}
		if got := getRequiredApiLevel(m); got != v {
			t.Fatalf("marker %q -> %q", v, got)
		}
	}
	// Non-numeric marker values are returned verbatim.
	if got := getRequiredApiLevel(map[string]string{JSRequiredLevelMetadataKey: "abc"}); got != "abc" {
		t.Fatalf("non-numeric=%q", got)
	}
}

// Detail 2: supportsRequiredApiLevel — absent/empty -> true; non-integer ->
// false; int <= JSApiLevel -> true; above -> false.
func TestDetail02_SupportsRequiredApiLevel(t *testing.T) {
	for _, m := range []map[string]string{
		nil, {}, {JSRequiredLevelMetadataKey: ""},
	} {
		if !supportsRequiredApiLevel(m) {
			t.Fatalf("%v should be supported (absent/empty)", m)
		}
	}
	for _, v := range []string{"abc", "1.5", "x3", "3x", " "} {
		if supportsRequiredApiLevel(map[string]string{JSRequiredLevelMetadataKey: v}) {
			t.Fatalf("non-integer %q should be unsupported", v)
		}
	}
	for lvl := 0; lvl <= JSApiLevel; lvl++ {
		if !supportsRequiredApiLevel(map[string]string{JSRequiredLevelMetadataKey: strconv.Itoa(lvl)}) {
			t.Fatalf("level %d <= %d should be supported", lvl, JSApiLevel)
		}
	}
	for _, lvl := range []int{JSApiLevel + 1, JSApiLevel + 100} {
		if supportsRequiredApiLevel(map[string]string{JSRequiredLevelMetadataKey: strconv.Itoa(lvl)}) {
			t.Fatalf("level %d > %d should be unsupported", lvl, JSApiLevel)
		}
	}
}

// Detail 3: stream static stamp — creates map when nil; strips dynamic
// fields first; marker = max over feature requirements stored as decimal
// string ALWAYS (incl "0").
func TestDetail03_StaticStreamStamp(t *testing.T) {
	// Bare config -> marker "0" still stored.
	cfg := &StreamConfig{Name: "S"}
	setStaticStreamMetadata(cfg)
	if cfg.Metadata == nil {
		t.Fatal("nil metadata not materialized")
	}
	if cfg.Metadata[JSRequiredLevelMetadataKey] != "0" {
		t.Fatalf("bare marker=%q want \"0\"", cfg.Metadata[JSRequiredLevelMetadataKey])
	}
	// Dynamic keys stripped by the static stamp.
	cfg = &StreamConfig{Name: "S", Metadata: map[string]string{
		JSServerVersionMetadataKey: "oldver",
		JSServerLevelMetadataKey:   "oldlevel",
		"user":                     "keep",
	}}
	setStaticStreamMetadata(cfg)
	if _, ok := cfg.Metadata[JSServerVersionMetadataKey]; ok {
		t.Fatal("dynamic version key survived static stamp")
	}
	if _, ok := cfg.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("dynamic level key survived static stamp")
	}
	if cfg.Metadata["user"] != "keep" {
		t.Fatal("user key dropped")
	}
	if cfg.Metadata[JSRequiredLevelMetadataKey] != "0" {
		t.Fatalf("marker=%q want \"0\"", cfg.Metadata[JSRequiredLevelMetadataKey])
	}
}

// Detail 4: stream feature table — msg-TTL or delete-marker-TTL>0 -> 1;
// counter/atomic/schedules/async-persist -> 2 each; batch -> 4. Max wins.
func TestDetail04_StreamFeatureLevels(t *testing.T) {
	rng := bbJsvRng(t)
	stamp := func(cfg *StreamConfig) string {
		setStaticStreamMetadata(cfg)
		return cfg.Metadata[JSRequiredLevelMetadataKey]
	}
	cases := []struct {
		name string
		Mk   func() *StreamConfig
		want string
	}{
		{"msgttl", func() *StreamConfig { return &StreamConfig{AllowMsgTTL: true} }, "1"},
		{"dmttl", func() *StreamConfig {
			return &StreamConfig{SubjectDeleteMarkerTTL: time.Duration(1 + rng.Int63n(1e9))}
		}, "1"},
		{"counter", func() *StreamConfig { return &StreamConfig{AllowMsgCounter: true} }, "2"},
		{"atomic", func() *StreamConfig { return &StreamConfig{AllowAtomicPublish: true} }, "2"},
		{"sched", func() *StreamConfig { return &StreamConfig{AllowMsgSchedules: true} }, "2"},
		{"async", func() *StreamConfig { return &StreamConfig{PersistMode: AsyncPersistMode} }, "2"},
		{"batch", func() *StreamConfig { return &StreamConfig{AllowBatchPublish: true} }, "4"},
		{"ttl+batch", func() *StreamConfig {
			return &StreamConfig{AllowMsgTTL: true, AllowBatchPublish: true}
		}, "4"},
		{"counter+ttl", func() *StreamConfig {
			return &StreamConfig{AllowMsgCounter: true, AllowMsgTTL: true}
		}, "2"},
		{"all", func() *StreamConfig {
			return &StreamConfig{AllowMsgTTL: true, AllowMsgCounter: true,
				AllowAtomicPublish: true, AllowMsgSchedules: true,
				PersistMode: AsyncPersistMode, AllowBatchPublish: true}
		}, "4"},
	}
	for _, c := range cases {
		if got := stamp(c.Mk()); got != c.want {
			t.Fatalf("%s: marker=%q want %q", c.name, got, c.want)
		}
	}
}

// Detail 5: consumer feature table — non-nil non-zero pause-until -> 1;
// priority policy!=none OR pinned-TTL!=0 OR any priority group -> 1;
// ack-flow-control -> 4.
func TestDetail05_ConsumerFeatureLevels(t *testing.T) {
	stamp := func(cfg *ConsumerConfig) string {
		setStaticConsumerMetadata(cfg)
		return cfg.Metadata[JSRequiredLevelMetadataKey]
	}
	future := time.Now().Add(time.Hour)
	cases := []struct {
		name string
		cfg  *ConsumerConfig
		want string
	}{
		{"bare", &ConsumerConfig{}, "0"},
		{"pause", &ConsumerConfig{PauseUntil: &future}, "1"},
		{"prio-policy", &ConsumerConfig{PriorityPolicy: PriorityOverflow}, "1"},
		{"prio-pinned", &ConsumerConfig{PriorityPolicy: PriorityPinnedClient}, "1"},
		{"pinned-ttl", &ConsumerConfig{PinnedTTL: time.Minute}, "1"},
		{"prio-groups", &ConsumerConfig{PriorityGroups: []string{"g1"}}, "1"},
		{"flow", &ConsumerConfig{AckPolicy: AckFlowControl}, "4"},
		{"flow+prio", &ConsumerConfig{AckPolicy: AckFlowControl, PriorityPolicy: PriorityOverflow}, "4"},
	}
	for _, c := range cases {
		if got := stamp(c.cfg); got != c.want {
			t.Fatalf("%s: marker=%q want %q", c.name, got, c.want)
		}
	}
	// Zero-time PauseUntil does not count.
	zero := time.Time{}
	if got := stamp(&ConsumerConfig{PauseUntil: &zero}); got != "0" {
		t.Fatalf("zero pause-until marker=%q want \"0\"", got)
	}
	// Dynamic keys stripped on consumer stamp too.
	cfg := &ConsumerConfig{Metadata: map[string]string{
		JSServerVersionMetadataKey: "v", JSServerLevelMetadataKey: "9"}}
	setStaticConsumerMetadata(cfg)
	if _, ok := cfg.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("consumer dynamic level survived static stamp")
	}
}

// Detail 6: dynamic stream/consumer calls — return a COPY (input never
// mutated) with fresh map = prior keys + version + level-as-string; nil
// input -> still a config holding only the two dynamic keys.
func TestDetail06_DynamicCopies(t *testing.T) {
	// nil input -> config with only the two dynamic keys.
	out := setDynamicStreamMetadata(nil)
	if out == nil {
		t.Fatal("nil input returned nil")
	}
	if len(out.Metadata) != 2 {
		t.Fatalf("nil-input metadata=%v want exactly 2 keys", out.Metadata)
	}
	if out.Metadata[JSServerLevelMetadataKey] != strconv.Itoa(JSApiLevel) {
		t.Fatalf("level=%q want %d", out.Metadata[JSServerLevelMetadataKey], JSApiLevel)
	}
	if out.Metadata[JSServerVersionMetadataKey] != VERSION {
		t.Fatalf("version=%q want %q", out.Metadata[JSServerVersionMetadataKey], VERSION)
	}
	// Input not mutated; output is a copy; prior keys preserved.
	in := &StreamConfig{Name: "S", Metadata: map[string]string{
		"user": "keep", JSRequiredLevelMetadataKey: "3"}}
	inCopy := *in
	out = setDynamicStreamMetadata(in)
	if out == in {
		t.Fatal("output aliases input")
	}
	if out.Metadata["user"] != "keep" || out.Metadata[JSRequiredLevelMetadataKey] != "3" {
		t.Fatalf("prior keys not preserved: %v", out.Metadata)
	}
	if out.Metadata[JSServerLevelMetadataKey] != strconv.Itoa(JSApiLevel) ||
		out.Metadata[JSServerVersionMetadataKey] != VERSION {
		t.Fatalf("dynamic keys: %v", out.Metadata)
	}
	if len(in.Metadata) != len(inCopy.Metadata) {
		t.Fatalf("input metadata mutated: %v", in.Metadata)
	}
	if _, ok := in.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("input gained dynamic key — not a copy")
	}
	// Fresh map: mutating out does not affect in.
	out.Metadata["user"] = "changed"
	if in.Metadata["user"] != "keep" {
		t.Fatal("output map shared with input")
	}
	// Consumer variant.
	cout := setDynamicConsumerMetadata(nil)
	if cout == nil || len(cout.Metadata) != 2 ||
		cout.Metadata[JSServerLevelMetadataKey] != strconv.Itoa(JSApiLevel) ||
		cout.Metadata[JSServerVersionMetadataKey] != VERSION {
		t.Fatalf("consumer nil-input: %v", cout)
	}
	cin := &ConsumerConfig{Durable: "d", Metadata: map[string]string{"u": "1"}}
	cout = setDynamicConsumerMetadata(cin)
	if cout == cin || len(cin.Metadata) != 1 {
		t.Fatal("consumer input mutated/aliased")
	}
	if cout.Metadata["u"] != "1" || cout.Metadata[JSServerLevelMetadataKey] != strconv.Itoa(JSApiLevel) {
		t.Fatalf("consumer dynamic: %v", cout.Metadata)
	}
}

// Detail 7: dynamic info call — nil info -> nil; else copy with
// dynamicized config.
func TestDetail07_DynamicInfoMetadata(t *testing.T) {
	if got := setDynamicConsumerInfoMetadata(nil); got != nil {
		t.Fatalf("nil info=%v want nil", got)
	}
	info := &ConsumerInfo{Config: &ConsumerConfig{Durable: "d",
		Metadata: map[string]string{"u": "1"}}}
	out := setDynamicConsumerInfoMetadata(info)
	if out == nil {
		t.Fatal("nil out")
	}
	if out == info {
		t.Fatal("out aliases info")
	}
	if out.Config == info.Config {
		t.Fatal("out.Config aliases in.Config")
	}
	if out.Config.Metadata[JSServerLevelMetadataKey] != strconv.Itoa(JSApiLevel) {
		t.Fatalf("info config not dynamicized: %v", out.Config.Metadata)
	}
	if _, ok := info.Config.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("input info mutated")
	}
}

// Detail 8: copy*Metadata — strips dynamic fields from new cfg first, then
// marker copied when present in prev (creating map); else deleted AND map
// reset to nil when emptied.
func TestDetail08_CopyMetadata(t *testing.T) {
	// Marker present in prev -> copied, creating map; dynamics stripped.
	prev := &StreamConfig{Metadata: map[string]string{
		JSRequiredLevelMetadataKey: "4",
		JSServerVersionMetadataKey: "prevver",
		JSServerLevelMetadataKey:   "prevlevel",
		"other":                    "prev",
	}}
	cfg := &StreamConfig{Metadata: map[string]string{
		JSServerVersionMetadataKey: "newver",
		JSServerLevelMetadataKey:   "newlevel",
		"user":                     "keep",
	}}
	copyStreamMetadata(cfg, prev)
	if _, ok := cfg.Metadata[JSServerVersionMetadataKey]; ok {
		t.Fatal("dynamic version key not stripped from cfg")
	}
	if _, ok := cfg.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("dynamic level key not stripped from cfg")
	}
	if cfg.Metadata[JSRequiredLevelMetadataKey] != "4" {
		t.Fatalf("marker=%q want copied \"4\"", cfg.Metadata[JSRequiredLevelMetadataKey])
	}
	if cfg.Metadata["user"] != "keep" {
		t.Fatal("user key lost")
	}
	// Marker absent in prev -> deleted from cfg; map nil when emptied.
	prev = &StreamConfig{Metadata: map[string]string{"other": "x"}}
	cfg = &StreamConfig{Metadata: map[string]string{
		JSRequiredLevelMetadataKey: "9", "user": "keep"}}
	copyStreamMetadata(cfg, prev)
	if _, ok := cfg.Metadata[JSRequiredLevelMetadataKey]; ok {
		t.Fatal("stale marker kept")
	}
	if cfg.Metadata["user"] != "keep" {
		t.Fatal("user key lost on delete path")
	}
	// Nil-collapse: deleting the only key resets map to nil.
	cfg = &StreamConfig{Metadata: map[string]string{JSRequiredLevelMetadataKey: "9"}}
	copyStreamMetadata(cfg, prev)
	if cfg.Metadata != nil {
		t.Fatalf("map=%v want nil after emptying", cfg.Metadata)
	}
	// Consumer variant mirrors it.
	cprev := &ConsumerConfig{Metadata: map[string]string{JSRequiredLevelMetadataKey: "2"}}
	ccfg := &ConsumerConfig{Metadata: map[string]string{
		JSServerLevelMetadataKey: "dyn", "user": "k"}}
	copyConsumerMetadata(ccfg, cprev)
	if ccfg.Metadata[JSRequiredLevelMetadataKey] != "2" || ccfg.Metadata["user"] != "k" {
		t.Fatalf("consumer copy: %v", ccfg.Metadata)
	}
	if _, ok := ccfg.Metadata[JSServerLevelMetadataKey]; ok {
		t.Fatal("consumer dynamic not stripped")
	}
}

// Detail 9: setOrDelete* — same set-or-delete-and-nil rule per key.
func TestDetail09_SetOrDelete(t *testing.T) {
	prev := &StreamConfig{Metadata: map[string]string{"k1": "v1"}}
	cfg := &StreamConfig{Metadata: map[string]string{"k1": "old", "k2": "x"}}
	setOrDeleteInStreamMetadata(cfg, prev, "k1")
	if cfg.Metadata["k1"] != "v1" {
		t.Fatalf("k1=%q want copied v1", cfg.Metadata["k1"])
	}
	// Key absent in prev -> deleted.
	setOrDeleteInStreamMetadata(cfg, prev, "k2")
	if _, ok := cfg.Metadata["k2"]; ok {
		t.Fatal("k2 not deleted")
	}
	// Deleting last key -> nil.
	cfg2 := &StreamConfig{Metadata: map[string]string{"only": "x"}}
	setOrDeleteInStreamMetadata(cfg2, prev, "only")
	if cfg2.Metadata != nil {
		t.Fatalf("map=%v want nil", cfg2.Metadata)
	}
	// Setting into nil map creates it.
	cfg3 := &StreamConfig{}
	setOrDeleteInStreamMetadata(cfg3, prev, "k1")
	if cfg3.Metadata == nil || cfg3.Metadata["k1"] != "v1" {
		t.Fatalf("nil map not materialized: %v", cfg3.Metadata)
	}
	// Consumer variant.
	cprev := &ConsumerConfig{Metadata: map[string]string{"k": "v"}}
	ccfg := &ConsumerConfig{}
	setOrDeleteInConsumerMetadata(ccfg, cprev, "k")
	if ccfg.Metadata == nil || ccfg.Metadata["k"] != "v" {
		t.Fatalf("consumer set: %v", ccfg.Metadata)
	}
	setOrDeleteInConsumerMetadata(ccfg, &ConsumerConfig{}, "k")
	if ccfg.Metadata != nil {
		t.Fatalf("consumer delete-nil: %v", ccfg.Metadata)
	}
}

// Detail 10: deleteDynamicMetadata removes exactly the two dynamic keys,
// leaves others.
func TestDetail10_DeleteDynamicMetadata(t *testing.T) {
	m := map[string]string{
		JSServerVersionMetadataKey: "v",
		JSServerLevelMetadataKey:   "l",
		JSRequiredLevelMetadataKey: "r",
		"user":                     "keep",
	}
	deleteDynamicMetadata(m)
	if len(m) != 2 {
		t.Fatalf("m=%v want 2 keys left", m)
	}
	if m[JSRequiredLevelMetadataKey] != "r" || m["user"] != "keep" {
		t.Fatalf("wrong keys removed: %v", m)
	}
	// Nil/empty-safe.
	deleteDynamicMetadata(nil)
	deleteDynamicMetadata(map[string]string{})
}

// Detail 11: errorOnRequiredApiLevel — absent/empty header -> no reject;
// non-integer header -> reject; integer > server level -> reject; <= -> ok.
func TestDetail11_ErrorOnRequiredApiLevel(t *testing.T) {
	// hdr is the whole NATS message-header block; the level lives in the
	// Nats-Required-Api-Level field.
	mk := func(v string) []byte {
		return []byte("NATS/1.0\r\n" + JSRequiredApiLevel + ": " + v + "\r\n\r\n")
	}
	if errorOnRequiredApiLevel(nil) {
		t.Fatal("nil header should not reject")
	}
	if errorOnRequiredApiLevel([]byte{}) {
		t.Fatal("empty header should not reject")
	}
	if errorOnRequiredApiLevel([]byte("NATS/1.0\r\nX-Other: 9\r\n\r\n")) {
		t.Fatal("header without the field should not reject")
	}
	// Non-integer values reject.
	for _, v := range []string{"abc", "1.5", " 6", "6 "} {
		if !errorOnRequiredApiLevel(mk(v)) {
			t.Fatalf("value %q should reject", v)
		}
	}
	for lvl := 0; lvl <= JSApiLevel; lvl++ {
		if errorOnRequiredApiLevel(mk(strconv.Itoa(lvl))) {
			t.Fatalf("level %d <= %d should not reject", lvl, JSApiLevel)
		}
	}
	for _, lvl := range []int{JSApiLevel + 1, JSApiLevel + 50} {
		if !errorOnRequiredApiLevel(mk(strconv.Itoa(lvl))) {
			t.Fatalf("level %d > %d should reject", lvl, JSApiLevel)
		}
	}
}

// Detail 12: level stored/returned as decimal STRING everywhere (Atoi both
// ways).
func TestDetail12_DecimalStringLevels(t *testing.T) {
	cfg := &StreamConfig{AllowBatchPublish: true}
	setStaticStreamMetadata(cfg)
	raw := cfg.Metadata[JSRequiredLevelMetadataKey]
	lvl, err := strconv.Atoi(raw)
	if err != nil || lvl != 4 {
		t.Fatalf("Atoi(%q)=(%d,%v) want (4,nil)", raw, lvl, err)
	}
	out := setDynamicStreamMetadata(nil)
	lvl, err = strconv.Atoi(out.Metadata[JSServerLevelMetadataKey])
	if err != nil || lvl != JSApiLevel {
		t.Fatalf("dynamic level %q not decimal JSApiLevel", out.Metadata[JSServerLevelMetadataKey])
	}
	// getRequiredApiLevel round-trips through Atoi for supports check.
	m := map[string]string{JSRequiredLevelMetadataKey: strconv.Itoa(JSApiLevel)}
	if getRequiredApiLevel(m) != strconv.Itoa(JSApiLevel) {
		t.Fatal("level not returned as decimal string")
	}
	if !supportsRequiredApiLevel(m) {
		t.Fatal("max level should be supported")
	}
}
