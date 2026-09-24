// Package kops_test is the hidden black-box suite for igrole.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package kops_test

import (
	"strings"
	"testing"

	kops "example.internal/clustkit/pkg/apis/kops"
)

// Detail 1 (Inferable: yes): matching is case-insensitive against the
// lowercased names in AllInstanceGroupRoles (visible in instancegroup.go).
func TestDetail01(t *testing.T) {
	for _, r := range kops.AllInstanceGroupRoles {
		name := string(r)
		for _, in := range []string{name, strings.ToLower(name), strings.ToUpper(name)} {
			got, ok := kops.ParseInstanceGroupRole(in, false)
			if !ok || got != r {
				t.Fatalf("strict parse %q = (%q,%v), want (%q,true)", in, got, ok, r)
			}
		}
	}
}

// Detail 2 (Inferable: no): "controlplane" (no dash) is normalized in BOTH
// modes — both spellings resolve to ControlPlane strict and lenient.
func TestDetail02(t *testing.T) {
	for _, in := range []string{"controlplane", "control-plane", "ControlPlane", "CONTROLPLANE"} {
		for _, lenient := range []bool{false, true} {
			got, ok := kops.ParseInstanceGroupRole(in, lenient)
			if !ok || got != kops.InstanceGroupRoleControlPlane {
				t.Fatalf("parse %q lenient=%v = (%q,%v), want ControlPlane", in, lenient, got, ok)
			}
		}
	}
}

// Detail 3 (Inferable: partially): lenient strips a trailing 's' on both
// sides — pluralized inputs match; strict rejects them.
func TestDetail03(t *testing.T) {
	for _, in := range []string{"nodes", "bastions", "controlplanes", "apiservers"} {
		if _, ok := kops.ParseInstanceGroupRole(in, false); ok {
			t.Fatalf("strict parse accepted plural %q", in)
		}
		got, ok := kops.ParseInstanceGroupRole(in, true)
		if !ok {
			t.Fatalf("lenient parse rejected plural %q", in)
		}
		// the plural resolves to the same role as its singular
		sing, ok2 := kops.ParseInstanceGroupRole(in[:len(in)-1], true)
		if !ok2 || got != sing {
			t.Fatalf("plural %q -> %q but singular -> %q", in, got, sing)
		}
	}
}

// Detail 4 (Inferable: no): "master" is a legacy alias that resolves only in
// lenient mode. Which role it resolves to is not pinned — shape is
// strict-rejects / lenient-accepts.
func TestDetail04(t *testing.T) {
	if _, ok := kops.ParseInstanceGroupRole("master", false); ok {
		t.Fatal("strict mode accepted master alias")
	}
	got, ok := kops.ParseInstanceGroupRole("master", true)
	if !ok {
		t.Fatal("lenient mode rejected master alias")
	}
	if got == "" {
		t.Fatal("lenient master alias returned empty role")
	}
}

// Detail 5 (Inferable: yes): failure returns ("", false), never an error.
func TestDetail05(t *testing.T) {
	for _, in := range []string{"bogus", "n", "node-node", " controlplane "} {
		for _, lenient := range []bool{false, true} {
			got, ok := kops.ParseInstanceGroupRole(in, lenient)
			if ok {
				t.Fatalf("input %q lenient=%v unexpectedly matched %q", in, lenient, got)
			}
			if got != "" {
				t.Fatalf("input %q failure returned %q, want empty", in, got)
			}
		}
	}
}

// Detail 6 (Inferable: partially): ParseRawYaml is strict — a document with
// fields absent from the destination type errors.
func TestDetail06(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
	}
	var good cfg
	if err := kops.ParseRawYaml([]byte("name: x\n"), &good); err != nil {
		t.Fatalf("valid yaml rejected: %v", err)
	}
	if good.Name != "x" {
		t.Fatalf("field not decoded: %+v", good)
	}
	var bad cfg
	if err := kops.ParseRawYaml([]byte("name: x\nbogusfield: y\n"), &bad); err == nil {
		t.Fatal("unknown field accepted — unmarshal not strict")
	}
}

// Detail 7 (Inferable: no): empty/whitespace input skips unmarshal and
// returns success, leaving dest untouched.
func TestDetail07(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
	}
	for _, in := range []string{"", "   ", "\n\t \n"} {
		dest := cfg{Name: "keep"}
		if err := kops.ParseRawYaml([]byte(in), &dest); err != nil {
			t.Fatalf("empty input %q errored: %v", in, err)
		}
		if dest.Name != "keep" {
			t.Fatalf("empty input %q mutated dest to %+v", in, dest)
		}
	}
}

// Detail 8 (Inferable: no): errors are wrapped with a fixed prefix. Asserted
// shape: two distinct parse failures share a non-trivial common prefix, and
// the marshal side also wraps (error non-nil, non-empty).
func TestDetail08(t *testing.T) {
	type cfg struct {
		Name string `json:"name"`
	}
	var d1, d2 cfg
	e1 := kops.ParseRawYaml([]byte("{not yaml"), &d1)
	e2 := kops.ParseRawYaml([]byte("unknownfield: 1"), &d2)
	if e1 == nil || e2 == nil {
		t.Fatal("expected errors")
	}
	common := 0
	for common < len(e1.Error()) && common < len(e2.Error()) &&
		e1.Error()[common] == e2.Error()[common] {
		common++
	}
	if common < 8 {
		t.Fatalf("errors share no fixed prefix: %q vs %q", e1, e2)
	}
	if _, err := kops.ToRawYaml(make(chan int)); err == nil {
		t.Fatal("ToRawYaml of unmarshalable value did not error")
	} else if err.Error() == "" {
		t.Fatal("ToRawYaml error was empty")
	}
	// Round-trip sanity on the success path.
	out, err := kops.ToRawYaml(cfg{Name: "z"})
	if err != nil {
		t.Fatalf("ToRawYaml valid: %v", err)
	}
	if !strings.Contains(string(out), "z") {
		t.Fatalf("marshal output missing value: %q", out)
	}
}
