package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"k8s.io/client-go/util/homedir"
)

// TestDetail01: StringSlicesEqual is order-sensitive;
// StringSlicesEqualIgnoreOrder sorts a copy (input not mutated).
func TestDetail01(t *testing.T) {
	if StringSlicesEqual([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("StringSlicesEqual ignored order")
	}
	if !StringSlicesEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("StringSlicesEqual on equal slices")
	}
	if StringSlicesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("StringSlicesEqual ignored length")
	}
	if !StringSlicesEqualIgnoreOrder([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("StringSlicesEqualIgnoreOrder on reordered slices")
	}
	if StringSlicesEqualIgnoreOrder([]string{"a", "b"}, []string{"a", "c"}) {
		t.Fatal("StringSlicesEqualIgnoreOrder on differing slices")
	}
	in := []string{"b", "a"}
	StringSlicesEqualIgnoreOrder(in, []string{"a", "b"})
	if in[0] != "b" || in[1] != "a" {
		t.Fatal("StringSlicesEqualIgnoreOrder mutated its input")
	}
}

// TestDetail02: HashString returns the lowercase sha256 hex (64 chars).
func TestDetail02(t *testing.T) {
	sum := sha256.Sum256([]byte("abc"))
	want := hex.EncodeToString(sum[:])
	got, err := HashString("abc")
	if err != nil {
		t.Fatalf("HashString: %v", err)
	}
	if got != want {
		t.Fatalf("HashString(abc) = %q, want %q", got, want)
	}
	if len(got) != 64 || got != strings.ToLower(got) {
		t.Fatalf("not 64 lowercase hex chars: %q", got)
	}
}

// TestDetail03: v4-mapped addresses are not IPv6; IsIPv4CIDR rejects any
// string containing ":" (a v4-mappable v6 CIDR is neither).
func TestDetail03(t *testing.T) {
	if IsIPv6IP("::ffff:10.0.0.1") {
		t.Fatal("v4-mapped IP is IPv6")
	}
	if !IsIPv6IP("fd00::1") {
		t.Fatal("fd00::1 is IPv6")
	}
	if IsIPv6IP("10.0.0.1") || IsIPv6IP("garbage") {
		t.Fatal("v4/invalid IP is IPv6")
	}

	if !IsIPv6CIDR("fd00::/64") {
		t.Fatal("fd00::/64 is IPv6 CIDR")
	}
	if IsIPv6CIDR("10.0.0.0/8") {
		t.Fatal("v4 CIDR is IPv6")
	}
	if !IsIPv4CIDR("10.0.0.0/8") {
		t.Fatal("10.0.0.0/8 is IPv4 CIDR")
	}
	if IsIPv4CIDR("fd00::/64") {
		t.Fatal("v6 CIDR is IPv4")
	}
	// v4-mappable v6 CIDR: parses as v6 (not IPv6) but contains ":" (not IPv4)
	mapped := "::ffff:a00:0/104"
	if IsIPv6CIDR(mapped) {
		t.Fatal("v4-mappable v6 CIDR is IPv6")
	}
	if IsIPv4CIDR(mapped) {
		t.Fatal("v6-form CIDR containing ':' is IPv4")
	}
}

// TestDetail04: ParseCIDRNotation parses the documented `/N#hex` form into
// (newSize, netNum) and errors on input that does not have that shape.
// (Exact acceptance rules beyond the documented form are not asserted.)
func TestDetail04(t *testing.T) {
	size, num, err := ParseCIDRNotation("/20#1")
	if err != nil {
		t.Fatalf("documented form rejected: %v", err)
	}
	if size != 20 || num != 1 {
		t.Fatalf("= (%d, %d), want (20, 1)", size, num)
	}
	if _, _, err := ParseCIDRNotation("garbage"); err == nil {
		t.Fatal("garbage accepted")
	}
	if _, _, err := ParseCIDRNotation(""); err == nil {
		t.Fatal("empty string accepted")
	}
}

// TestDetail05: CIDRSubnet(prefix, newSize, netNum) yields the netNum-th
// subnet of prefix at newSize (big-endian numbering).
func TestDetail05(t *testing.T) {
	got, err := CIDRSubnet("10.0.0.0/16", 24, 0)
	if err != nil || got != "10.0.0.0/24" {
		t.Fatalf("subnet 0 = %q %v", got, err)
	}
	got, err = CIDRSubnet("10.0.0.0/16", 24, 1)
	if err != nil || got != "10.0.1.0/24" {
		t.Fatalf("subnet 1 = %q %v", got, err)
	}
	got, err = CIDRSubnet("10.0.0.0/16", 24, 3)
	if err != nil || got != "10.0.3.0/24" {
		t.Fatalf("subnet 3 = %q %v", got, err)
	}
	if _, err := CIDRSubnet("not-a-cidr", 24, 0); err == nil {
		t.Fatal("invalid prefix accepted")
	}
}

// TestDetail06: SanitizeString replaces disallowed characters rather than
// removing them, and keeps only a bounded tail of a long input (per api.md:
// allowed set [a-zA-Z0-9_-], replacement '_', last 200 chars).
func TestDetail06(t *testing.T) {
	if got := SanitizeString("abc-DEF_123"); got != "abc-DEF_123" {
		t.Fatalf("allowed chars changed: %q", got)
	}
	got := SanitizeString("a b")
	if strings.Contains(got, " ") {
		t.Fatalf("disallowed char kept: %q", got)
	}
	if len(got) != 3 {
		t.Fatalf("disallowed char removed instead of replaced: %q", got)
	}
	long := "head" + strings.Repeat("b", 300)
	got = SanitizeString(long)
	if len(got) > 200 {
		t.Fatalf("no length cap: %d chars", len(got))
	}
	if strings.Contains(got, "head") {
		t.Fatal("prefix kept instead of tail")
	}
}

// TestDetail07: ExpandPath expands only a leading "~/" via homedir; a bare
// "~" and other inputs pass through unchanged.
func TestDetail07(t *testing.T) {
	home := homedir.HomeDir()
	if home == "" {
		t.Skip("no home dir")
	}
	if got := ExpandPath("~/sub/dir"); got != home+"/sub/dir" {
		t.Fatalf("~/ expansion = %q, want %q", got, home+"/sub/dir")
	}
	if got := ExpandPath("~"); got != "~" {
		t.Fatalf("bare ~ expanded: %q", got)
	}
	if got := ExpandPath("~other/x"); got != "~other/x" {
		t.Fatalf("~other expanded: %q", got)
	}
	if got := ExpandPath("/abs/path"); got != "/abs/path" {
		t.Fatalf("abs path changed: %q", got)
	}
}

// TestDetail08: YamlUnmarshal/YamlMarshal honour json field tags.
func TestDetail08(t *testing.T) {
	type bbY struct {
		CamelName string `json:"camelName,omitempty"`
		Plain     int    `json:"plain"`
	}
	var out bbY
	if err := YamlUnmarshal([]byte("camelName: v\nplain: 3\n"), &out); err != nil {
		t.Fatalf("YamlUnmarshal: %v", err)
	}
	if out.CamelName != "v" || out.Plain != 3 {
		t.Fatalf("unmarshal = %+v", out)
	}
	b, err := YamlMarshal(&bbY{CamelName: "v", Plain: 3})
	if err != nil {
		t.Fatalf("YamlMarshal: %v", err)
	}
	if !strings.Contains(string(b), "camelName: v") {
		t.Fatalf("marshal output %q lacks json-tagged field", b)
	}
}
