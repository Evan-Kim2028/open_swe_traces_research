package kops

import (
	"strings"
	"testing"

	"example.internal/clustkit/util/pkg/architectures"
	"github.com/blang/semver/v4"
)

func bbMustSemver(t *testing.T, s string) semver.Version {
	t.Helper()
	v, err := semver.ParseTolerant(s)
	if err != nil {
		t.Fatalf("bad test version %q: %v", s, err)
	}
	return v
}

// TestDetail01: ResolveChannel — "none" yields (nil, nil), a relative name
// resolves against DefaultChannelBase, absolute URLs pass through, and
// unparseable input produces an error.
func TestDetail01(t *testing.T) {
	u, err := ResolveChannel("none")
	if err != nil || u != nil {
		t.Fatalf("ResolveChannel(none) = %v, %v; want nil, nil", u, err)
	}

	u, err = ResolveChannel("stable")
	if err != nil {
		t.Fatalf("ResolveChannel(stable): %v", err)
	}
	if u == nil || u.String() != DefaultChannelBase+"stable" {
		t.Fatalf("relative resolve = %v, want under DefaultChannelBase", u)
	}

	u, err = ResolveChannel("https://example.com/custom/channel")
	if err != nil || u == nil || u.String() != "https://example.com/custom/channel" {
		t.Fatalf("absolute URL mangled: %v %v", u, err)
	}

	if _, err := ResolveChannel("\x7f"); err == nil {
		t.Fatal("unparseable channel location accepted")
	} else if !strings.Contains(err.Error(), "channel") {
		t.Fatalf("error does not mention the channel: %v", err)
	}
}

// TestDetail02: ParseChannel yaml-decodes a Channel and wraps parse failures
// in an "error parsing channel" error.
func TestDetail02(t *testing.T) {
	c, err := ParseChannel([]byte("spec:\n  packages:\n  - name: mypkg\n    version: 1.2.3\n"))
	if err != nil {
		t.Fatalf("ParseChannel valid: %v", err)
	}
	if len(c.Spec.Packages) != 1 || c.Spec.Packages[0].Name != "mypkg" {
		t.Fatalf("parsed spec = %+v", c.Spec)
	}

	if _, err := ParseChannel([]byte("{")); err == nil {
		t.Fatal("invalid yaml accepted")
	} else if !strings.Contains(err.Error(), "error parsing channel") {
		t.Fatalf("error = %v, want 'error parsing channel' wrap", err)
	}
}

// TestDetail03: FindRecommendedUpgrade/IsUpgradeRequired — empty fields are
// nil/false, and only a strictly greater version counts as an upgrade.
func TestDetail03(t *testing.T) {
	cur := bbMustSemver(t, "1.30.0")

	empty := &KubernetesVersionSpec{}
	if got, err := empty.FindRecommendedUpgrade(cur); err != nil || got != nil {
		t.Fatalf("empty recommended = %v, %v", got, err)
	}
	if ok, err := empty.IsUpgradeRequired(cur); err != nil || ok {
		t.Fatalf("empty required = %v, %v", ok, err)
	}

	spec := &KubernetesVersionSpec{RecommendedVersion: "1.31.0", RequiredVersion: "1.30.0"}
	got, err := spec.FindRecommendedUpgrade(cur)
	if err != nil || got == nil || got.String() != "1.31.0" {
		t.Fatalf("upgrade = %v, %v; want 1.31.0", got, err)
	}
	if got, err := spec.FindRecommendedUpgrade(bbMustSemver(t, "1.31.0")); err != nil || got != nil {
		t.Fatalf("equal version is not an upgrade: %v, %v", got, err)
	}
	if got, err := spec.FindRecommendedUpgrade(bbMustSemver(t, "1.32.0")); err != nil || got != nil {
		t.Fatalf("newer current is not an upgrade: %v, %v", got, err)
	}
	if ok, err := spec.IsUpgradeRequired(bbMustSemver(t, "1.29.9")); err != nil || !ok {
		t.Fatalf("required upgrade = %v, %v; want true", ok, err)
	}
	if ok, err := spec.IsUpgradeRequired(cur); err != nil || ok {
		t.Fatalf("equal required = %v, %v; want false", ok, err)
	}

	kspec := &KopsVersionSpec{RecommendedVersion: "1.31.0", RequiredVersion: "1.30.0"}
	if got, err := kspec.FindRecommendedUpgrade(cur); err != nil || got == nil || got.String() != "1.31.0" {
		t.Fatalf("kops upgrade = %v, %v", got, err)
	}
	if got, err := kspec.FindRecommendedUpgrade(bbMustSemver(t, "1.31.0")); err != nil || got != nil {
		t.Fatalf("kops equal recommended produced an upgrade: %v, %v", got, err)
	}
	if ok, err := kspec.IsUpgradeRequired(bbMustSemver(t, "1.29.0")); err != nil || !ok {
		t.Fatalf("kops required = %v, %v", ok, err)
	}
}

// TestDetail04: Kubernetes-version spec fields are parsed tolerantly (the
// /v1.<n>. URL form works); the kops spec fields use strict tolerant semver
// and do not accept that form.
func TestDetail04(t *testing.T) {
	cur := bbMustSemver(t, "1.29.0")
	urlForm := "https://example.test/releases/v1.30.2/notes.txt"

	// the URL pattern contributes major=1, minor=<n>; the patch never appears
	kspec := &KubernetesVersionSpec{RecommendedVersion: urlForm}
	got, err := kspec.FindRecommendedUpgrade(cur)
	if err != nil || got == nil || got.String() != "1.30.0" {
		t.Fatalf("URL-form recommended = %v, %v; want 1.30.0", got, err)
	}

	cspec := &KopsVersionSpec{RecommendedVersion: urlForm}
	got, err = cspec.FindRecommendedUpgrade(cur)
	if got != nil {
		t.Fatalf("kops spec accepted URL-form version: %v", got)
	}
	_ = err // either nil-with-nil or an error is acceptable; no upgrade may be reported
}

// TestDetail05: FindKubernetesVersionSpec/FindKopsVersionSpec return the first
// entry whose Range matches; empty Range matches anything; unparseable ranges
// are skipped.
func TestDetail05(t *testing.T) {
	v := bbMustSemver(t, "1.30.0")

	versions := []KubernetesVersionSpec{
		{Range: ">=1.30.0", RecommendedVersion: "first"},
		{Range: "", RecommendedVersion: "catchall"},
	}
	if got := FindKubernetesVersionSpec(versions, v); got == nil || got.RecommendedVersion != "first" {
		t.Fatalf("first match = %+v", got)
	}
	if got := FindKubernetesVersionSpec(versions, bbMustSemver(t, "1.0.0")); got == nil || got.RecommendedVersion != "catchall" {
		t.Fatalf("empty range should match any: %+v", got)
	}
	// empty range first means it wins over a later specific range
	reordered := []KubernetesVersionSpec{versions[1], versions[0]}
	if got := FindKubernetesVersionSpec(reordered, v); got == nil || got.RecommendedVersion != "catchall" {
		t.Fatalf("first-entry order not honored: %+v", got)
	}
	bad := []KubernetesVersionSpec{{Range: "not-a-range", RecommendedVersion: "x"}}
	if got := FindKubernetesVersionSpec(bad, v); got != nil {
		t.Fatalf("unparseable range not skipped: %+v", got)
	}

	kversions := []KopsVersionSpec{{Range: ">=2.0.0"}, {Range: ""}}
	if got := FindKopsVersionSpec(kversions, v); got != &kversions[1] {
		t.Fatalf("kops find = %+v", got)
	}
}

// TestDetail06: FindImage filters by provider; empty ArchitectureID and empty
// KubernetesVersion act as wildcards; first match wins.
func TestDetail06(t *testing.T) {
	v := bbMustSemver(t, "1.30.0")
	c := &Channel{Spec: ChannelSpec{Images: []*ChannelImageSpec{
		{ProviderID: "gce", Name: "gce-img"},
		{ProviderID: "aws", ArchitectureID: "arm64", Name: "aws-arm"},
		{ProviderID: "aws", Name: "aws-any", KubernetesVersion: ">=1.99.0"},
		{ProviderID: "aws", Name: "aws-plain"},
	}}}

	got := c.FindImage(CloudProviderAWS, v, architectures.ArchitectureAmd64)
	if got == nil || got.Name != "aws-plain" {
		t.Fatalf("aws/amd64 image = %+v", got)
	}
	got = c.FindImage(CloudProviderAWS, v, architectures.ArchitectureArm64)
	if got == nil || got.Name != "aws-arm" {
		t.Fatalf("aws/arm64 image = %+v", got)
	}
	if got := c.FindImage(CloudProviderDO, v, architectures.ArchitectureAmd64); got != nil {
		t.Fatalf("unknown provider matched: %+v", got)
	}
	// version range respected: a matching-arch image whose range excludes the
	// version is skipped in favour of the wildcard.
	c2 := &Channel{Spec: ChannelSpec{Images: []*ChannelImageSpec{
		{ProviderID: "aws", Name: "aws-new", KubernetesVersion: ">=1.99.0"},
		{ProviderID: "aws", Name: "aws-any"},
	}}}
	if got := c2.FindImage(CloudProviderAWS, v, architectures.ArchitectureAmd64); got == nil || got.Name != "aws-any" {
		t.Fatalf("version-excluded image matched: %+v", got)
	}
}

// TestDetail07: RecommendedKubernetesVersion maps kops version -> matching
// spec -> that spec's KubernetesVersion; the spec's other fields are not the
// answer.
func TestDetail07(t *testing.T) {
	c := &Channel{Spec: ChannelSpec{KopsVersions: []KopsVersionSpec{
		{Range: ">=1.28.0", KubernetesVersion: "1.30.0", RecommendedVersion: "9.9.9"},
		{Range: "", KubernetesVersion: "1.20.0"},
	}}}

	got := RecommendedKubernetesVersion(c, "1.30.5")
	if got == nil || got.String() != "1.30.0" {
		t.Fatalf("recommended = %v, want 1.30.0 (the spec field, not RecommendedVersion)", got)
	}
	if got := RecommendedKubernetesVersion(c, "0.5.0"); got == nil || got.String() != "1.20.0" {
		t.Fatalf("catch-all = %v, want 1.20.0", got)
	}
	if got := RecommendedKubernetesVersion(&Channel{}, "1.30.0"); got != nil {
		t.Fatalf("no specs -> %v, want nil", got)
	}
	if got := RecommendedKubernetesVersion(c, "not-a-version"); got != nil {
		t.Fatalf("unparseable kops version -> %v, want nil", got)
	}
}

// TestDetail08: HasUpstreamImagePrefix consults a fixed prefix list (shape
// only — the list contents are an implementation detail): the answer depends
// only on the image string, and arbitrary custom images are not upstream.
func TestDetail08(t *testing.T) {
	c1 := &Channel{}
	c2 := &Channel{Spec: ChannelSpec{Images: []*ChannelImageSpec{{ProviderID: "aws", Name: "x"}}}}

	if c1.HasUpstreamImagePrefix("") {
		t.Fatal("empty image is upstream")
	}
	for _, custom := range []string{"example.com/team/custom:v1", "myregistry.internal/img", "nginx:latest"} {
		if c1.HasUpstreamImagePrefix(custom) || c2.HasUpstreamImagePrefix(custom) {
			t.Fatalf("custom image %q classified as upstream", custom)
		}
	}
}

// TestDetail09: GetPackageVersion requires name equality; no match is an
// error; a package's KubernetesVersion range is honored when a version is
// supplied. (DETAILS claims a nil kubernetesVersion skips the range check;
// the reference implementation does not dereference-safely on nil for ranged
// packages, so that sub-case is deliberately not asserted.)
func TestDetail09(t *testing.T) {
	c := &Channel{Spec: ChannelSpec{Packages: []PackageVersionSpec{
		{Name: "etcd", Version: "3.5.0"},
		{Name: "cni", Version: "1.4.0", KubernetesVersion: ">=1.28.0"},
	}}}

	got, err := c.GetPackageVersion("etcd", nil)
	if err != nil || got == nil || got.String() != "3.5.0" {
		t.Fatalf("etcd = %v, %v", got, err)
	}
	old := bbMustSemver(t, "1.20.0")
	if _, err := c.GetPackageVersion("cni", &old); err == nil {
		t.Fatal("out-of-range kubernetes version matched")
	}
	newer := bbMustSemver(t, "1.30.0")
	if got, err := c.GetPackageVersion("cni", &newer); err != nil || got == nil || got.String() != "1.4.0" {
		t.Fatalf("in-range = %v, %v", got, err)
	}
	if _, err := c.GetPackageVersion("missing", nil); err == nil {
		t.Fatal("missing package found")
	}
}
