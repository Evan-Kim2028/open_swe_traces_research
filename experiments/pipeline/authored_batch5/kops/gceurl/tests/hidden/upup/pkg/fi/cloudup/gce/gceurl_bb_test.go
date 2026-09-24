package gce

import (
	"reflect"
	"strings"
	"testing"

	"example.internal/clustkit/pkg/apis/kops"
)

// TestDetail01: BuildURL composes the documented URL — version defaults to
// v1, segments are slash-separated, and projects/global/regions/zones are
// independent optional segments.
func TestDetail01(t *testing.T) {
	u := &GoogleCloudURL{Type: "instanceGroups", Name: "ig-x"}
	if got := u.BuildURL(); got != "https://www.googleapis.com/compute/v1/instanceGroups/ig-x" {
		t.Fatalf("BuildURL = %q", got)
	}
	u = &GoogleCloudURL{Version: "beta", Project: "p", Zone: "z", Type: "t", Name: "n"}
	if got := u.BuildURL(); got != "https://www.googleapis.com/compute/beta/projects/p/zones/z/t/n" {
		t.Fatalf("BuildURL = %q", got)
	}
	u = &GoogleCloudURL{Global: true, Type: "t", Name: "n"}
	if got := u.BuildURL(); got != "https://www.googleapis.com/compute/v1/global/t/n" {
		t.Fatalf("BuildURL = %q", got)
	}
	u = &GoogleCloudURL{Region: "r", Type: "t", Name: "n"}
	if got := u.BuildURL(); got != "https://www.googleapis.com/compute/v1/regions/r/t/n" {
		t.Fatalf("BuildURL = %q", got)
	}

	// round-trip: parse(build(u)) == u for explicit versions
	for _, u := range []*GoogleCloudURL{
		{Version: "v1", Type: "t", Name: "n"},
		{Version: "beta", Project: "p", Region: "r", Type: "t", Name: "n"},
		{Version: "v1", Project: "p", Zone: "z", Type: "t", Name: "n"},
		{Version: "v1", Global: true, Type: "t", Name: "n"},
	} {
		got, err := ParseGoogleCloudURL(u.BuildURL())
		if err != nil || !reflect.DeepEqual(got, u) {
			t.Fatalf("round-trip of %q = %+v, %v", u.BuildURL(), got, err)
		}
	}
}

// TestDetail02: the parser rejects non-https schemes, foreign hosts,
// non-compute services, and non v1/beta versions.
func TestDetail02(t *testing.T) {
	for _, u := range []string{
		"http://www.googleapis.com/compute/v1/t/n",
		"https://other.example.com/compute/v1/t/n",
		"https://www.googleapis.com/storage/v1/t/n",
		"https://www.googleapis.com/compute/v2/t/n",
		"https://www.googleapis.com/compute/alpha/t/n",
		"compute/v1/t/n",
		"",
	} {
		if got, err := ParseGoogleCloudURL(u); err == nil {
			t.Fatalf("%q accepted as %+v", u, got)
		}
	}
	for _, u := range []string{
		"https://www.googleapis.com/compute/v1/t/n",
		"https://www.googleapis.com/compute/beta/t/n",
	} {
		if _, err := ParseGoogleCloudURL(u); err != nil {
			t.Fatalf("%q rejected: %v", u, err)
		}
	}
}

// TestDetail03: a "regions" segment with a following region name is parsed
// into Region; zones and projects similarly consume their value token.
// (The under-fed "regions" fall-through quirk is exercised only for shape:
// it must not panic, and if it parses it cannot claim a Region.)
func TestDetail03(t *testing.T) {
	got, err := ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/regions/r/t/n")
	if err != nil {
		t.Fatalf("regions url: %v", err)
	}
	if got.Region != "r" || got.Type != "t" || got.Name != "n" {
		t.Fatalf("regions parse = %+v", got)
	}
	got, err = ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/zones/z/t/n")
	if err != nil || got.Zone != "z" {
		t.Fatalf("zones parse = %+v %v", got, err)
	}
	got, err = ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/projects/p/t/n")
	if err != nil || got.Project != "p" {
		t.Fatalf("projects parse = %+v %v", got, err)
	}

	// under-fed regions segment: must not panic; if it parses, Region stays
	// empty and the remaining token is the Name
	got, err = ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/regions/x")
	if err == nil && (got.Region != "" || got.Name != "x") {
		t.Fatalf("under-fed regions parse = %+v", got)
	}
}

// TestDetail04: after TYPE/NAME the parser requires end-of-input; a missing
// NAME or trailing segments error.
func TestDetail04(t *testing.T) {
	if _, err := ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/t/n/extra"); err == nil {
		t.Fatal("trailing segment accepted")
	}
	if _, err := ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/t/n/e/more"); err == nil {
		t.Fatal("multiple trailing segments accepted")
	}
	if _, err := ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/t"); err == nil {
		t.Fatal("missing NAME accepted")
	}
	got, err := ParseGoogleCloudURL("https://www.googleapis.com/compute/v1/t/n")
	if err != nil || got.Type != "t" || got.Name != "n" {
		t.Fatalf("minimal parse = %+v %v", got, err)
	}
}

// TestDetail05: EncodeGCELabel escapes per byte — lowercase alnum pass
// through, everything else becomes "-" + two lowercase hex digits.
func TestDetail05(t *testing.T) {
	if got := EncodeGCELabel("a.b"); got != "a-2eb" {
		t.Fatalf("EncodeGCELabel(a.b) = %q", got)
	}
	if got := EncodeGCELabel("abc09"); got != "abc09" {
		t.Fatalf("EncodeGCELabel(abc09) = %q", got)
	}
	if got := EncodeGCELabel("A"); got != "-41" {
		t.Fatalf("EncodeGCELabel(A) = %q — uppercase must be escaped", got)
	}
	if got := EncodeGCELabel("a b"); got != "a-20b" {
		t.Fatalf("EncodeGCELabel(a b) = %q", got)
	}
	// bytes, not runes: U+00E9 is two UTF-8 bytes -> two escapes
	if got := EncodeGCELabel("é"); got != "-c3-a9" {
		t.Fatalf("EncodeGCELabel(é) = %q", got)
	}
}

// TestDetail06: DecodeGCELabel inverts EncodeGCELabel; malformed escapes
// return an error that echoes the input label.
func TestDetail06(t *testing.T) {
	got, err := DecodeGCELabel("a-2eb")
	if err != nil || got != "a.b" {
		t.Fatalf("DecodeGCELabel(a-2eb) = %q %v", got, err)
	}
	for _, s := range []string{"plain", "a.b", "x y/z", "-2e-2e"} {
		enc := EncodeGCELabel(s)
		dec, err := DecodeGCELabel(enc)
		if err != nil || dec != s {
			t.Fatalf("round-trip %q -> %q -> %q %v", s, enc, dec, err)
		}
	}
	if _, err := DecodeGCELabel("-zz"); err == nil {
		t.Fatal("invalid hex escape accepted")
	} else if !strings.Contains(err.Error(), "-zz") {
		t.Fatalf("error does not echo the label: %v", err)
	}
	if _, err := DecodeGCELabel("abc-4"); err == nil {
		t.Fatal("truncated escape accepted")
	}
}

// TestDetail07: TagForRole composes the role prefix with
// ClusterPrefixedName at the documented length cap.
func TestDetail07(t *testing.T) {
	got := TagForRole("my.cluster", kops.InstanceGroupRoleNode)
	want := ClusterPrefixedName("k8s-io-role-node", "my.cluster", 63)
	if got != want {
		t.Fatalf("TagForRole = %q, want %q", got, want)
	}
	gotCP := TagForRole("my.cluster", kops.InstanceGroupRoleControlPlane)
	if !strings.Contains(gotCP, "control-plane") {
		t.Fatalf("control-plane tag = %q", gotCP)
	}
	long := TagForRole("averyveryveryverylongclustername.example.com", kops.InstanceGroupRoleNode)
	if len(long) > 63 {
		t.Fatalf("tag len = %d > 63", len(long))
	}
}
