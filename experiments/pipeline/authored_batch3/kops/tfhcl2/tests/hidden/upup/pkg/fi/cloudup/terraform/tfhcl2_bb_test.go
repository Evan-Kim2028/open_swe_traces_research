// Package terraform_test is the hidden black-box suite for tfhcl2.
// One TestDetailNN per DETAILS.md commitment. Exercised end-to-end through
// NewTerraformTarget + Render*/AddOutput* + Finish -> kubernetes.tf.
package terraform_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"example.internal/clustkit/dnsprovider/pkg/dnsprovider"
	"example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/cloudinstances"
	"example.internal/clustkit/upup/pkg/fi"
	"example.internal/clustkit/upup/pkg/fi/cloudup/terraform"
	tw "example.internal/clustkit/upup/pkg/fi/cloudup/terraformWriter"
	v1 "k8s.io/api/core/v1"
)

type fakeCloud struct {
	pid    kops.CloudProviderID
	region string
}

func (f *fakeCloud) ProviderID() kops.CloudProviderID            { return f.pid }
func (f *fakeCloud) Region() string                              { return f.region }
func (f *fakeCloud) DNS() (dnsprovider.Interface, error)         { return nil, nil }
func (f *fakeCloud) FindVPCInfo(id string) (*fi.VPCInfo, error)  { return nil, nil }
func (f *fakeCloud) DeleteInstance(*cloudinstances.CloudInstance) error {
	return nil
}
func (f *fakeCloud) DeregisterInstance(*cloudinstances.CloudInstance) error {
	return nil
}
func (f *fakeCloud) DeleteGroup(*cloudinstances.CloudInstanceGroup) error   { return nil }
func (f *fakeCloud) DetachInstance(*cloudinstances.CloudInstance) error     { return nil }
func (f *fakeCloud) FindClusterStatus(*kops.Cluster) (*kops.ClusterStatus, error) {
	return nil, nil
}
func (f *fakeCloud) GetApiIngressStatus(*kops.Cluster) ([]fi.ApiIngressStatus, error) {
	return nil, nil
}
func (f *fakeCloud) GetCloudGroups(*kops.Cluster, []*kops.InstanceGroup, bool, []v1.Node) (map[string]*cloudinstances.CloudInstanceGroup, error) {
	return nil, nil
}

type sub struct {
	Flag bool `cty:"flag"`
}

type res struct {
	Name     string            `cty:"name"`
	Count    int               `cty:"count"`
	Enabled  bool              `cty:"enabled"`
	Opt      *string           `cty:"opt"`
	Tags     map[string]string `cty:"tags"`
	Items    []string          `cty:"items"`
	Empty    []string          `cty:"empty"`
	EmptyMap map[string]string `cty:"emptymap"`
	Sub      *sub              `cty:"sub"`
	Subs     []*sub            `cty:"subs"`
	Lit      *tw.Literal       `cty:"lit"`
	HTTPPort int               // no cty tag — exercises snake_case fallback
}

func demoRes() *res {
	return &res{
		Name: "n", Count: 3, Enabled: true,
		Tags:  map[string]string{"b": "2", "a": "1"},
		Items: []string{"x", "y"}, Empty: []string{}, EmptyMap: map[string]string{},
		Sub:   &sub{Flag: true}, Subs: []*sub{{Flag: true}, {Flag: false}},
		Lit:   tw.LiteralTokens("aws_vpc", "main", "id"),
		HTTPPort: 9,
	}
}

func render(t *testing.T, pid kops.CloudProviderID, mutate func(tgt *terraform.TerraformTarget)) string {
	t.Helper()
	dir := t.TempDir()
	tgt := terraform.NewTerraformTarget(&fakeCloud{pid: pid, region: "r1"}, "proj1", dir, nil)
	if err := tgt.RenderResource("demo_type", "zzz", demoRes()); err != nil {
		t.Fatal(err)
	}
	if err := tgt.RenderResource("demo_type", "aaa", demoRes()); err != nil {
		t.Fatal(err)
	}
	if err := tgt.RenderDataSource("demo_ds", "d1", demoRes()); err != nil {
		t.Fatal(err)
	}
	if err := tgt.AddOutputVariable("outb", tw.LiteralTokens("a", "b")); err != nil {
		t.Fatal(err)
	}
	if err := tgt.AddOutputVariableArray("outa", tw.LiteralTokens("c")); err != nil {
		t.Fatal(err)
	}
	if err := tgt.AddOutputVariableArray("outa", tw.LiteralTokens("d")); err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(tgt)
	}
	if err := tgt.Finish(nil); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "kubernetes.tf"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func eqColumn(line string) int { return strings.Index(line, "=") }

// Detail 1 (Inferable: no): object.Write emits `key { ... }` with fields
// sorted by key; consecutive single-value fields get '=' aligned over the
// longest key in the run.
func TestDetail01(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	for _, want := range []string{`resource "demo_type" "aaa" {`, "count", "enabled", "items", "name"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	// sorted: count < enabled < items < lit < name < sub < tags
	ia, ib := strings.Index(out, "count"), strings.Index(out, "enabled")
	ic, id := strings.Index(out, "items"), strings.Index(out, "lit ")
	ie, ig := strings.Index(out, "name"), strings.Index(out, "sub {")
	if !(ia < ib && ib < ic && ic < id && id < ie && ie < ig) {
		t.Fatalf("fields not sorted by key:\n%s", out)
	}
	// alignment: '=' columns equal across a run of single-value fields, and a
	// non-single-value field resets the run (the map block's '=' sits at its
	// own column)
	var enLine, itemsLine, tagsLine string
	for _, l := range strings.Split(out, "\n") {
		s := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(s, "enabled"):
			enLine = l
		case strings.HasPrefix(s, "items"):
			itemsLine = l
		case strings.HasPrefix(s, "tags"):
			tagsLine = l
		}
	}
	if eqColumn(enLine) == -1 || eqColumn(enLine) != eqColumn(itemsLine) {
		t.Fatalf("'=' not aligned in run: %q vs %q", enLine, itemsLine)
	}
	if eqColumn(tagsLine) == eqColumn(enLine) {
		t.Fatalf("run not reset after block: %q vs %q", tagsLine, enLine)
	}
}

// Detail 2 (Inferable: partially): toElement kind coverage — bool, int,
// string, *Literal verbatim, nil ptr omitted, struct -> object block,
// slice -> list path.
func TestDetail02(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	kvRe := func(k, v string) *regexp.Regexp {
		return regexp.MustCompile(`(?m)^\s*` + k + `\s*=\s*` + v + `$`)
	}
	for _, want := range []*regexp.Regexp{
		kvRe("enabled", "true"), kvRe("count", "3"), kvRe("name", `"n"`),
		kvRe("lit", "aws_vpc.main.id"), regexp.MustCompile(`(?m)^\s*sub \{$`),
		regexp.MustCompile(`(?m)^\s*items\s*=\s*\[`),
	} {
		if !want.MatchString(out) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "opt") {
		t.Fatalf("nil pointer field emitted:\n%s", out)
	}
}

// Detail 3 (Inferable: no): an empty slice produces NO element — the field is
// omitted entirely.
func TestDetail03(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if strings.Contains(out, "empty") {
		t.Fatalf("empty slice field emitted:\n%s", out)
	}
}

// Detail 4 (Inferable: no): []string collapses to a `[a, b]` list expression;
// []struct stays a multi-block where each member writes under the same key.
func TestDetail04(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if !regexp.MustCompile(`items\s*=\s*\[\s*"x",\s*"y"\s*\]`).MatchString(out) {
		t.Fatalf("string slice not a list expression:\n%s", out)
	}
	if n := strings.Count(out, "subs {"); n != 2*3 { // two subs x (aaa, zzz, d1)
		t.Fatalf("struct slice not multi-block (n=%d):\n%s", n, out)
	}
}

// Detail 5 (Inferable: no): fieldKey prefers the cty tag; untagged Go names
// are snake_cased. Asserted shape: cty tag values appear verbatim; the
// untagged HTTPPort renders as an all-lowercase identifier whose letters
// (ignoring '_') still spell the original.
func TestDetail05(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if !strings.Contains(out, "flag = true") {
		t.Fatalf("cty tag name not used:\n%s", out)
	}
	re := regexp.MustCompile(`\n {2}([a-z_0-9]+) = 9\n`)
	m := re.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("untagged field not emitted:\n%s", out)
	}
	if strings.ReplaceAll(m[1], "_", "") != "httpport" {
		t.Fatalf("untagged field key %q is not a snake form of HTTPPort", m[1])
	}
	if strings.Contains(out, "HTTPPort") {
		t.Fatal("go name emitted verbatim")
	}
}

// Detail 6 (Inferable: partially): mapStringLiteral — empty map emits
// nothing; otherwise `key = {` + sorted, aligned `"k" = v` lines + `}`.
func TestDetail06(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if strings.Contains(out, "emptymap") {
		t.Fatalf("empty map emitted:\n%s", out)
	}
	i := strings.Index(out, "tags = {")
	if i < 0 {
		t.Fatalf("map not a block:\n%s", out)
	}
	ia := strings.Index(out, `"a" = "1"`)
	ib := strings.Index(out, `"b" = "2"`)
	if ia < 0 || ib < 0 || ia > ib {
		t.Fatalf("map entries missing or unsorted:\n%s", out)
	}
}

// Detail 7 (Inferable: no): mapToElement accepts map[string]string and
// map[string]*Literal; other map kinds panic. Asserted shape: unsupported
// map kind aborts the write (panic) rather than silently dropping.
func TestDetail07(t *testing.T) {
	type badRes struct {
		M map[int]string `cty:"m"`
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("map[int]string field did not panic")
		}
	}()
	dir := t.TempDir()
	tgt := terraform.NewTerraformTarget(&fakeCloud{pid: kops.CloudProviderAWS}, "p", dir, nil)
	if err := tgt.RenderResource("t", "n", &badRes{M: map[int]string{1: "x"}}); err != nil {
		t.Fatal(err)
	}
	_ = tgt.Finish(nil)
}

// Detail 8 (Inferable: no): quote escaping is minimal — newline and tab pass
// through unescaped. Asserted shape: raw newline/tab bytes appear inside the
// emitted value. (The reference implementation emits `"` raw too — the
// DETAILS escape-set claim is a divergence, reported, and not asserted.)
func TestDetail08(t *testing.T) {
	dir := t.TempDir()
	tgt := terraform.NewTerraformTarget(&fakeCloud{pid: kops.CloudProviderAWS}, "p", dir, nil)
	type esc struct {
		V string `cty:"v"`
	}
	if err := tgt.RenderResource("t", "n", &esc{V: "xy\nz\tw"}); err != nil {
		t.Fatal(err)
	}
	if err := tgt.Finish(nil); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "kubernetes.tf"))
	out := string(b)
	if !strings.Contains(out, "y\nz") {
		t.Fatalf("newline escaped: %q", out)
	}
	if !strings.Contains(out, "z\tw") {
		t.Fatalf("tab escaped: %q", out)
	}
	if strings.Contains(out, `y\nz`) || strings.Contains(out, `z\tw`) {
		t.Fatalf("value rendered with escape sequences: %q", out)
	}
}

// Detail 9 (Inferable: partially): writeIndent emits single spaces; nested
// content indents deeper. Asserted shape: block members are space-indented
// and a nested member indents deeper than a top-level member.
func TestDetail09(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if !regexp.MustCompile(`\n {2}name\s*=`).MatchString(out) {
		t.Fatalf("no 1-level indent:\n%s", out)
	}
	if !regexp.MustCompile(`\n {4}flag\s*=`).MatchString(out) {
		t.Fatalf("no deeper indent in sub block:\n%s", out)
	}
}

// Detail 10 (Inferable: partially): finishHCL2 section order — locals/outputs,
// provider, resources, data sources, terraform {} — lands in
// Files["kubernetes.tf"].
func TestDetail10(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	locs := strings.Index(out, "locals {")
	outs := strings.Index(out, "output \"")
	prov := strings.Index(out, "provider \"")
	res := strings.Index(out, "resource \"")
	ds := strings.Index(out, "data \"")
	tf := strings.Index(out, "terraform {")
	if !(locs >= 0 && outs > locs && prov > outs && res > prov && ds > res && tf > ds) {
		t.Fatalf("section order wrong (l=%d o=%d p=%d r=%d d=%d t=%d):\n%s", locs, outs, prov, res, ds, tf, out)
	}
}

// Detail 11 (Inferable: no): scalar outputs go to the locals object; array
// outputs collapse to a `[...]` list expression there; then one
// `output "name"` block per name in sorted order.
func TestDetail11(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	locs := out[strings.Index(out, "locals {"):strings.Index(out, "output \"")]
	if !strings.Contains(locs, "outa = [") || !strings.Contains(locs, "outb") {
		t.Fatalf("locals missing outputs: %q", locs)
	}
	ia := strings.Index(out, `output "outa"`)
	ib := strings.Index(out, `output "outb"`)
	if ia < 0 || ib < 0 || ia > ib {
		t.Fatalf("output blocks missing or unsorted:\n%s", out)
	}
}

// Detail 12 (Inferable: no): per-provider body rules — the provider block
// exists; verbatim-name providers carry region; GCE additionally carries
// project; the files-alias extra provider renders a second block with an
// alias. Asserted shape only (names are not pinned beyond the verbatim aws
// case).
func TestDetail12(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	if !strings.Contains(out, `provider "aws"`) {
		t.Fatalf("no provider block:\n%s", out)
	}
	awsBlock := out[strings.Index(out, `provider "aws"`):]
	if !strings.Contains(awsBlock[:strings.Index(awsBlock, "}")], "region") {
		t.Fatal("aws provider lacks region")
	}
	if strings.Contains(awsBlock[:strings.Index(awsBlock, "}")], "project") {
		t.Fatal("non-gce provider has project")
	}
	out = render(t, kops.CloudProviderGCE, nil)
	pi := strings.Index(out, "provider \"")
	gceBlock := out[pi:strings.Index(out[pi:], "}")+pi]
	if !strings.Contains(gceBlock, "project") || !strings.Contains(gceBlock, "region") {
		t.Fatalf("gce provider lacks project/region: %q", gceBlock)
	}
	// files-alias extra provider
	out = render(t, kops.CloudProviderAWS, func(tgt *terraform.TerraformTarget) {
		tgt.EnsureTerraformProvider("aws", map[string]string{"region": "r9"})
	})
	if n := strings.Count(out, `provider "aws"`); n != 2 {
		t.Fatalf("extra provider block missing (n=%d):\n%s", n, out)
	}
	if !strings.Contains(out, "alias") {
		t.Fatal("extra provider has no alias")
	}
}

// Detail 13 (Inferable: partially): resources and data sources iterate types
// then names, both sorted, under `resource "t" "n"` / `data "t" "n"` headers
// separated by blank lines.
func TestDetail13(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	ia := strings.Index(out, `resource "demo_type" "aaa"`)
	iz := strings.Index(out, `resource "demo_type" "zzz"`)
	if ia < 0 || iz < 0 || ia > iz {
		t.Fatalf("resource headers missing or unsorted:\n%s", out)
	}
	if !strings.Contains(out, `data "demo_ds" "d1"`) {
		t.Fatalf("data header missing:\n%s", out)
	}
	if !strings.Contains(out, "}\n\nresource") {
		t.Fatal("blocks not blank-line separated")
	}
}

// Detail 14 (Inferable: no): terraform {} pins required_version and one
// required_providers entry per provider (sorted, fixed source/version table);
// aliased providers emit configuration_aliases. Asserted shape: the block
// exists with required_version + required_providers keys; an entry for the
// used provider carries source/version; a files-alias produces a
// configuration_aliases line. (Under the reference implementation an
// unlisted provider yields an EMPTY required_providers block, not an abort —
// the DETAILS abort claim is a divergence, reported.)
func TestDetail14(t *testing.T) {
	out := render(t, kops.CloudProviderAWS, nil)
	tf := out[strings.Index(out, "terraform {"):]
	if !strings.Contains(tf, "required_version") || !strings.Contains(tf, "required_providers") {
		t.Fatalf("terraform block incomplete: %q", tf)
	}
	if !strings.Contains(tf, "source") || !strings.Contains(tf, "version") {
		t.Fatalf("provider entry lacks source/version: %q", tf)
	}
	out = render(t, kops.CloudProviderAWS, func(tgt *terraform.TerraformTarget) {
		tgt.EnsureTerraformProvider("aws", map[string]string{"region": "r9"})
	})
	if !strings.Contains(out, "configuration_aliases") {
		t.Fatalf("aliased provider lacks configuration_aliases:\n%s", out)
	}
}
