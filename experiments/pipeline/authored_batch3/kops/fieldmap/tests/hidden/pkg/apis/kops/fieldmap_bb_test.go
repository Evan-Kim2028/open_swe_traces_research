// Package kops_test is the hidden black-box suite for fieldmap.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package kops_test

import (
	"testing"

	kops "example.internal/clustkit/pkg/apis/kops"
)

// The mapping table itself is kept in the tree (visible to the solver), so
// the v1alpha2/v1alpha3 spellings are pinned; direction is the commitment.
var fieldMapPairs = [][2]string{
	{"spec.masterPublicName", "spec.api.publicName"},
	{"spec.topology.dns.type", "spec.networking.topology.dns"},
	{"spec.topology.bastion.loadBalancer.type", "spec.networking.topology.bastion.loadBalancer.type"},
	{"spec.externalDns.provider", "spec.externalDNS.provider"},
	{"spec.externalDns.priorityClassName", "spec.externalDNS.priorityClassName"},
}

// Detail 1 (Inferable: partially): HumanPath* maps v1alpha3 -> v1alpha2.
func TestDetail01(t *testing.T) {
	for _, p := range fieldMapPairs {
		if got := kops.HumanPathForClusterField(p[1]); got != p[0] {
			t.Fatalf("HumanPathForClusterField(%q) = %q, want %q", p[1], got, p[0])
		}
	}
}

// Detail 2 (Inferable: partially): InternalPath* maps v1alpha2 -> v1alpha3.
func TestDetail02(t *testing.T) {
	for _, p := range fieldMapPairs {
		if got := kops.InternalPathForClusterField(p[0]); got != p[1] {
			t.Fatalf("InternalPathForClusterField(%q) = %q, want %q", p[0], got, p[1])
		}
	}
}

// Detail 3 (Inferable: partially): unmapped paths pass through verbatim in
// both directions (not an error, not empty).
func TestDetail03(t *testing.T) {
	for _, s := range []string{
		"spec.unmappedField",
		"spec.elsewhere.deeper.path",
		"metadata.name",
		"",
	} {
		if got := kops.HumanPathForClusterField(s); got != s {
			t.Fatalf("HumanPathForClusterField(%q) = %q, want passthrough", s, got)
		}
		if got := kops.InternalPathForClusterField(s); got != s {
			t.Fatalf("InternalPathForClusterField(%q) = %q, want passthrough", s, got)
		}
	}
}

// Detail 4 (Inferable: yes): only table entries translate; matching is
// exact-string — prefixes, suffixes and case variants pass through.
func TestDetail04(t *testing.T) {
	for _, s := range []string{
		"spec.api.publicName.extra",
		"spec.api.publicNameX",
		"SPEC.API.PUBLICNAME",
		"spec.api.PublicName",
		"spec.masterPublicName.suffix",
		"xspec.masterPublicName",
	} {
		if got := kops.InternalPathForClusterField(s); got != s {
			t.Fatalf("InternalPathForClusterField(%q) = %q, want verbatim", s, got)
		}
		if got := kops.HumanPathForClusterField(s); got != s {
			t.Fatalf("HumanPathForClusterField(%q) = %q, want verbatim", s, got)
		}
	}
}

// Detail 5 (Inferable: yes): NewClusterField stores the path verbatim and the
// wrappers delegate through the same mapping.
func TestDetail05(t *testing.T) {
	in := "spec.api.publicName"
	f := kops.NewClusterField(in)
	if f == nil {
		t.Fatal("NewClusterField returned nil")
	}
	if f.Path != in {
		t.Fatalf("Path = %q, want stored verbatim %q", f.Path, in)
	}
	if f.HumanPath() != kops.HumanPathForClusterField(in) {
		t.Fatal("HumanPath wrapper does not delegate")
	}
	if f.InternalPath() != kops.InternalPathForClusterField(in) {
		t.Fatal("InternalPath wrapper does not delegate")
	}
	if f.PathInV1Alpha2() != "spec.masterPublicName" {
		t.Fatalf("PathInV1Alpha2 = %q", f.PathInV1Alpha2())
	}
	if f.PathInV1Alpha3() != "spec.api.publicName" {
		t.Fatalf("PathInV1Alpha3 = %q", f.PathInV1Alpha3())
	}
	f2 := kops.NewClusterField("spec.masterPublicName")
	if f2.PathInV1Alpha3() != "spec.api.publicName" || f2.PathInV1Alpha2() != "spec.masterPublicName" {
		t.Fatal("PathInV1Alpha* did not map")
	}
}

// Detail 6 (Inferable: no): HumanPath is named for display but returns the
// OLD (v1alpha2) spelling — i.e. it is not an echo of the input.
func TestDetail06(t *testing.T) {
	for _, p := range fieldMapPairs {
		got := kops.HumanPathForClusterField(p[1])
		if got == p[1] {
			t.Fatalf("HumanPath(%q) echoed the input — direction backwards", p[1])
		}
		if got != p[0] {
			t.Fatalf("HumanPath(%q) = %q, want old spelling %q", p[1], got, p[0])
		}
	}
}
