package util

import (
	"testing"

	"github.com/blang/semver/v4"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func bbNode(labels map[string]string) *v1.Node {
	return &v1.Node{ObjectMeta: metav1.ObjectMeta{Labels: labels}}
}

// TestDetail01: GetNodeRole checks node-role.kubernetes.io/{master,
// control-plane,node,api-server} in that order and returns the short name;
// falls back to the kubernetes.io/role label.
func TestDetail01(t *testing.T) {
	cases := []struct {
		labels map[string]string
		want   string
	}{
		{map[string]string{"node-role.kubernetes.io/master": ""}, "master"},
		{map[string]string{"node-role.kubernetes.io/control-plane": ""}, "control-plane"},
		{map[string]string{"node-role.kubernetes.io/node": ""}, "node"},
		{map[string]string{"node-role.kubernetes.io/api-server": ""}, "apiserver"},
		// precedence: master wins over node
		{map[string]string{"node-role.kubernetes.io/master": "", "node-role.kubernetes.io/node": ""}, "master"},
		// precedence: control-plane wins over node
		{map[string]string{"node-role.kubernetes.io/control-plane": "", "node-role.kubernetes.io/node": ""}, "control-plane"},
		// fallback to kubernetes.io/role
		{map[string]string{"kubernetes.io/role": "worker"}, "worker"},
		// nothing
		{map[string]string{}, ""},
	}
	for _, tc := range cases {
		if got := GetNodeRole(bbNode(tc.labels)); got != tc.want {
			t.Fatalf("GetNodeRole(%v) = %q, want %q", tc.labels, got, tc.want)
		}
	}
}

// TestDetail02: ParseTaint splits on ":" — one part is key-only; two parts
// are key[=value]:effect. All of key/value/effect are always present in the
// returned map. More than two parts is an error.
func TestDetail02(t *testing.T) {
	got, err := ParseTaint("key")
	if err != nil {
		t.Fatalf("key-only: %v", err)
	}
	if got["key"] != "key" || got["value"] != "" || got["effect"] != "" {
		t.Fatalf("key-only map = %v", got)
	}
	got, err = ParseTaint("k=v:NoSchedule")
	if err != nil {
		t.Fatalf("k=v:effect: %v", err)
	}
	if got["key"] != "k" || got["value"] != "v" || got["effect"] != "NoSchedule" {
		t.Fatalf("k=v:effect map = %v", got)
	}
	got, err = ParseTaint("k:NoExecute")
	if err != nil {
		t.Fatalf("k:effect: %v", err)
	}
	if got["key"] != "k" || got["value"] != "" || got["effect"] != "NoExecute" {
		t.Fatalf("k:effect map = %v", got)
	}
	// one colon-part is a bare key — "k=v" is the whole key
	got, err = ParseTaint("k=v")
	if err != nil {
		t.Fatalf("k=v single part: %v", err)
	}
	if got["key"] != "k=v" || got["value"] != "" || got["effect"] != "" {
		t.Fatalf("k=v map = %v", got)
	}
	for _, bad := range []string{"a:b:c", "k=a=b:e"} {
		if _, err := ParseTaint(bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
}

// TestDetail03: ParseKubernetesVersion tolerates partial semver and falls
// back to the /v1.<minor>. URL form; everything else errors.
func TestDetail03(t *testing.T) {
	v, err := ParseKubernetesVersion("1.30.2")
	if err != nil || v.String() != "1.30.2" {
		t.Fatalf("1.30.2 = %v %v", v, err)
	}
	v, err = ParseKubernetesVersion("v1.29")
	if err != nil || v.String() != "1.29.0" {
		t.Fatalf("v1.29 = %v %v", v, err)
	}
	v, err = ParseKubernetesVersion("https://example.com/releases/v1.30./manifest.yaml")
	if err != nil {
		t.Fatalf("url form: %v", err)
	}
	if v.Major != 1 || v.Minor != 30 {
		t.Fatalf("url form = %v, want 1.30.x", v)
	}
	if _, err := ParseKubernetesVersion("not-a-version"); err == nil {
		t.Fatal("garbage accepted")
	}
}

// TestDetail04: IsKubernetesGTE panics when `version` won't parse and
// strips Pre/Build from the k8sVersion argument before comparing.
func TestDetail04(t *testing.T) {
	panicked := true
	func() {
		defer func() {
			if recover() == nil {
				panicked = false
			}
		}()
		IsKubernetesGTE("garbage", semver.MustParse("1.30.0"))
	}()
	if !panicked {
		t.Fatal("unparseable version did not panic")
	}

	// Pre/Build stripped from k8sVersion: 1.30.0-alpha.1 satisfies >=1.30.0
	if !IsKubernetesGTE("1.30.0", semver.MustParse("1.30.0-alpha.1")) {
		t.Fatal("pre-release component of k8sVersion not stripped")
	}
	if IsKubernetesGTE("1.31.0", semver.MustParse("1.30.0")) {
		t.Fatal("1.30.0 reported >= 1.31.0")
	}
	if !IsKubernetesGTE("1.30.0", semver.MustParse("1.30.0+build.5")) {
		t.Fatal("build metadata not stripped")
	}
}

// TestDetail05: ParseVersion is strict semver — "1.2" fails where the
// tolerant parser succeeds; String/IsInRange accessors work.
func TestDetail05(t *testing.T) {
	if _, err := ParseVersion("1.2"); err == nil {
		t.Fatal("strict parser accepted 1.2")
	}
	if _, err := ParseKubernetesVersion("1.2"); err != nil {
		t.Fatalf("tolerant parser rejected 1.2: %v", err)
	}
	v, err := ParseVersion("1.30.2")
	if err != nil {
		t.Fatalf("strict parse 1.30.2: %v", err)
	}
	if v.String() != "1.30.2" {
		t.Fatalf("String = %q", v.String())
	}
	r, err := semver.ParseRange(">=1.30.0 <1.31.0")
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsInRange(r) {
		t.Fatal("1.30.2 not in [1.30,1.31)")
	}
	r2, err := semver.ParseRange(">=1.31.0")
	if err != nil {
		t.Fatal(err)
	}
	if v.IsInRange(r2) {
		t.Fatal("1.30.2 in >=1.31.0")
	}
}
