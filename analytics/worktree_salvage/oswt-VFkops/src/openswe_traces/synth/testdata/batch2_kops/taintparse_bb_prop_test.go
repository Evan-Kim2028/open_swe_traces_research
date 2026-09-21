package util_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kopsutil "example.internal/kops/pkg/apis/kops/util"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

// Detail 1: accepts key, key:effect, key=value:effect; map always has
// exactly keys key/value/effect (empty strings allowed).
func TestDetail01_TaintFormsAndMapShape(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("key%d", rng.Intn(1000))
		effect := []string{"NoSchedule", "PreferNoSchedule", "NoExecute", "eff" + strconv.Itoa(i)}[rng.Intn(4)]
		var in string
		want := map[string]string{"key": key, "value": "", "effect": ""}
		switch rng.Intn(3) {
		case 0:
			in = key
		case 1:
			in = key + ":" + effect
			want["effect"] = effect
		case 2:
			val := "val" + strconv.Itoa(rng.Intn(100))
			in = key + "=" + val + ":" + effect
			want["value"] = val
			want["effect"] = effect
		}
		m, err := kopsutil.ParseTaint(in)
		if err != nil {
			t.Fatalf("i=%d ParseTaint(%q) err %v", i, in, err)
		}
		if len(m) != 3 {
			t.Fatalf("i=%d map has %d keys: %v", i, len(m), m)
		}
		for k, v := range want {
			got, ok := m[k]
			if !ok {
				t.Fatalf("i=%d map missing key %q: %v", i, k, m)
			}
			if got != v {
				t.Fatalf("i=%d m[%q]=%q want %q (input %q)", i, k, got, v, in)
			}
		}
	}
}

// Detail 2: two or more ":" -> error "invalid taint spec: <input>"; the
// returned map is empty but non-nil.
func TestDetail02_TwoColonsError(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		in := fmt.Sprintf("k%d:v%d:eff%d:extra%d", rng.Intn(10), rng.Intn(10), rng.Intn(10), rng.Intn(10))
		m, err := kopsutil.ParseTaint(in)
		if err == nil {
			t.Fatalf("i=%d ParseTaint(%q) accepted", i, in)
		}
		want := "invalid taint spec: " + in
		if err.Error() != want {
			t.Fatalf("i=%d err %q want %q", i, err.Error(), want)
		}
		if m == nil {
			t.Fatalf("i=%d nil map returned", i)
		}
		if len(m) != 0 {
			t.Fatalf("i=%d map has entries %v", i, m)
		}
	}
}

// Detail 3: two or more "=" errors ONLY when a colon is present; without
// a colon the whole string becomes the key verbatim.
func TestDetail03_EqualsRequiresColon(t *testing.T) {
	// a=b:c -> one = with colon: parsed normally.
	m, err := kopsutil.ParseTaint("a=b:c")
	if err != nil || m["value"] != "b" || m["effect"] != "c" {
		t.Fatalf("a=b:c -> %v err %v", m, err)
	}
	// a=b=c:d -> two = with colon: error.
	if _, err := kopsutil.ParseTaint("a=b=c:d"); err == nil {
		t.Fatalf("a=b=c:d accepted")
	}
	// a=b (no colon): whole string is the key — not an error, not split.
	m, err = kopsutil.ParseTaint("a=b")
	if err != nil {
		t.Fatalf("a=b errored: %v", err)
	}
	if m["key"] != "a=b" || m["value"] != "" || m["effect"] != "" {
		t.Fatalf("a=b -> %v want key=a=b", m)
	}
	// a=b=c (no colon): also whole-string key.
	m, err = kopsutil.ParseTaint("a=b=c")
	if err != nil {
		t.Fatalf("a=b=c errored: %v", err)
	}
	if m["key"] != "a=b=c" {
		t.Fatalf("a=b=c -> %v", m)
	}
}

func nodeWith(labels map[string]string) *v1.Node {
	return &v1.Node{ObjectMeta: metav1.ObjectMeta{Labels: labels}}
}

// Detail 4: GetNodeRole precedence — master > control-plane > node >
// api-server > legacy kubernetes.io/role.
func TestDetail04_RolePrecedence(t *testing.T) {
	cases := []struct {
		labels map[string]string
		want   string
	}{
		{map[string]string{"node-role.kubernetes.io/master": ""}, "master"},
		{map[string]string{"node-role.kubernetes.io/control-plane": ""}, "control-plane"},
		{map[string]string{"node-role.kubernetes.io/node": ""}, "node"},
		{map[string]string{"node-role.kubernetes.io/api-server": ""}, "apiserver"},
		{map[string]string{
			"node-role.kubernetes.io/master":        "",
			"node-role.kubernetes.io/control-plane": "",
			"node-role.kubernetes.io/node":          "",
		}, "master"},
		{map[string]string{
			"node-role.kubernetes.io/control-plane": "",
			"node-role.kubernetes.io/node":          "",
			"node-role.kubernetes.io/api-server":    "",
		}, "control-plane"},
		{map[string]string{
			"node-role.kubernetes.io/node":       "",
			"node-role.kubernetes.io/api-server": "",
		}, "node"},
		{map[string]string{
			"node-role.kubernetes.io/api-server": "",
			"kubernetes.io/role":                 "legacy-role",
		}, "apiserver"},
	}
	for i, c := range cases {
		got := kopsutil.GetNodeRole(nodeWith(c.labels))
		if got != c.want {
			t.Fatalf("i=%d labels %v -> %q want %q", i, c.labels, got, c.want)
		}
	}
}

// Detail 5: legacy fallback returns the label's VALUE, not a fixed
// string; absent -> "".
func TestDetail05_LegacyLabelValue(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		val := fmt.Sprintf("custom-role-%d", rng.Intn(1000))
		got := kopsutil.GetNodeRole(nodeWith(map[string]string{"kubernetes.io/role": val}))
		if got != val {
			t.Fatalf("i=%d legacy role %q want %q", i, got, val)
		}
	}
	if got := kopsutil.GetNodeRole(nodeWith(map[string]string{})); got != "" {
		t.Fatalf("no labels -> %q want \"\"", got)
	}
	if got := kopsutil.GetNodeRole(nodeWith(nil)); got != "" {
		t.Fatalf("nil labels -> %q want \"\"", got)
	}
}

// Detail 6: new-style labels match on key PRESENCE — the value is ignored.
func TestDetail06_KeyPresenceValueIgnored(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	keys := []struct{ key, role string }{
		{"node-role.kubernetes.io/master", "master"},
		{"node-role.kubernetes.io/control-plane", "control-plane"},
		{"node-role.kubernetes.io/node", "node"},
		{"node-role.kubernetes.io/api-server", "apiserver"},
	}
	for i, c := range keys {
		val := fmt.Sprintf("arbitrary-%d", rng.Intn(100))
		got := kopsutil.GetNodeRole(nodeWith(map[string]string{c.key: val}))
		if got != c.role {
			t.Fatalf("i=%d %s=%q -> %q want %q", i, c.key, val, got, c.role)
		}
	}
}
