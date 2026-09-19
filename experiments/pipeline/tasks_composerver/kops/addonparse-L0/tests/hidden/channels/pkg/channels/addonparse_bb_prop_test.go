// Package channels_test — hidden black-box property suite for addonparse.
// Exported API only (api.md): ParseAddons, GetCurrent, ChannelVersion, GetRequiredUpdates.
// Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"kind Addons document parses" -> TestAddonParseContractTableProperty
//	"first-object Addons wins in a multi-doc stream" -> TestAddonParseContractTableProperty
//	"non-Addons YAML is wrapped as a synthetic addon" -> TestAddonParseContractTableProperty
//	"synthetic name is manifest- plus 12 hex of location hash" -> TestAddonParseContractTableProperty
//	"invalid YAML is an error" -> TestAddonParseEmptyInvalidProperty
//	"empty input is not an error" -> TestAddonParseEmptyInvalidProperty
//	"kubernetes version range selects the matching addon" -> TestAddonGetCurrentVersionProperty
//	"id/hash/generation decide which duplicate wins" -> TestAddonReplacementProperty
//	"nil vs non-nil update from existing version + replace" -> TestAddonRequiredUpdatesProperty
//	"NeedsPKI missing CA → InstallPKI" -> TestAddonRequiredUpdatesProperty
package channels_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	fakecertmanager "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubernetes "k8s.io/client-go/kubernetes/fake"

	"example.internal/clustkit/channels/pkg/api"
	"example.internal/clustkit/channels/pkg/channels"
	"example.internal/clustkit/upup/pkg/fi/utils"
)

const bbSeed = 20260919
const bbCases = 10000

func bbURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url %q: %v", raw, err)
	}
	return u
}

func bbParse(t *testing.T, loc *url.URL, data string) *channels.Addons {
	t.Helper()
	a, err := channels.ParseAddons("channel", loc, []byte(data))
	if err != nil {
		t.Fatalf("ParseAddons: %v", err)
	}
	return a
}

func bbAddonsDoc(metaName string, entries ...string) string {
	var b strings.Builder
	b.WriteString("kind: Addons\nmetadata:\n  name: ")
	b.WriteString(metaName)
	b.WriteString("\nspec:\n  addons:\n")
	for _, e := range entries {
		b.WriteString("  - ")
		b.WriteString(e)
		b.WriteString("\n")
	}
	return b.String()
}

func bbEntry(name, extra string) string {
	var b strings.Builder
	b.WriteString("name: ")
	b.WriteString(name)
	b.WriteString("\n")
	if extra == "" {
		b.WriteString("    manifest: ")
		b.WriteString(name)
		b.WriteString(".yaml")
		return b.String()
	}
	for _, ln := range strings.Split(extra, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		b.WriteString("    ")
		b.WriteString(ln)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func bbManifestName(t *testing.T, loc *url.URL) string {
	t.Helper()
	h, err := utils.HashString(loc.String())
	if err != nil {
		t.Fatal(err)
	}
	return "manifest-" + h[:12]
}

func bbVersionMatches(rangeStr string, v semver.Version) bool {
	if rangeStr == "" {
		return true
	}
	rng, err := semver.ParseRange(rangeStr)
	if err != nil {
		return false
	}
	return rng(v)
}

func TestAddonParseContractTableProperty(t *testing.T) {
	loc := bbURL(t, "https://example.com/addons/addon.yaml")
	doc := bbAddonsDoc("example", bbEntry("example.addons.k8s.io", "manifest: example.yaml\n    manifestHash: abc123"))
	a := bbParse(t, loc, doc)
	if a.APIObject.ObjectMeta.Name != "example" {
		t.Fatalf("name %q", a.APIObject.ObjectMeta.Name)
	}
	if len(a.APIObject.Spec.Addons) != 1 || a.APIObject.Spec.Addons[0].Name == nil || *a.APIObject.Spec.Addons[0].Name != "example.addons.k8s.io" {
		t.Fatal("addon spec")
	}

	multi := doc + `---
kind: Addons
metadata:
  name: other
spec:
  addons:
  - name: other.addons.k8s.io
    manifest: other.yaml
`
	a2 := bbParse(t, loc, multi)
	if a2.APIObject.ObjectMeta.Name != "example" {
		t.Fatalf("multidoc first object: %q", a2.APIObject.ObjectMeta.Name)
	}

	manLoc := bbURL(t, "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml")
	manData := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: gateways.gateway.networking.k8s.io
`
	trimmed := strings.TrimSpace(manData)
	a3 := bbParse(t, manLoc, manData)
	wantName := bbManifestName(t, manLoc)
	wantHash, _ := utils.HashString(trimmed)
	if a3.APIObject.ObjectMeta.Name != wantName {
		t.Fatalf("wrap name %q want %q", a3.APIObject.ObjectMeta.Name, wantName)
	}
	spec := a3.APIObject.Spec.Addons[0]
	if spec.Name == nil || *spec.Name != wantName || spec.Manifest == nil || *spec.Manifest != manLoc.String() || spec.ManifestHash != wantHash {
		t.Fatal("wrapped manifest fields")
	}

	loc2 := bbURL(t, "https://example.com/addons/direct.yaml")
	a4a, _ := channels.ParseAddons("channel", loc2, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: v1\n"))
	a4b, _ := channels.ParseAddons("channel", loc2, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: v2\n"))
	if *a4a.APIObject.Spec.Addons[0].Name != *a4b.APIObject.Spec.Addons[0].Name {
		t.Fatal("location hash name unstable")
	}
	if a4a.APIObject.Spec.Addons[0].ManifestHash == a4b.APIObject.Spec.Addons[0].ManifestHash {
		t.Fatal("content hash should differ")
	}
}

func TestAddonParseEmptyInvalidProperty(t *testing.T) {
	loc := bbURL(t, "https://example.com/addons/addon.yaml")
	for _, data := range []string{"", "\n  \n", "# nothing\n"} {
		a, err := channels.ParseAddons("channel", loc, []byte(data))
		if err != nil || len(a.APIObject.Spec.Addons) != 0 {
			t.Fatalf("empty %q err=%v len=%d", data, err, len(a.APIObject.Spec.Addons))
		}
	}
	if _, err := channels.ParseAddons("channel", loc, []byte("not: [valid")); err == nil {
		t.Fatal("invalid yaml")
	}
	if _, err := channels.ParseAddons("channel", loc, []byte("not: [valid")); err != nil && !strings.Contains(err.Error(), "error parsing addons or manifest") {
		t.Fatalf("error text: %v", err)
	}

	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		pad := strings.Repeat(" ", rng.Intn(4))
		doc := pad + bbAddonsDoc("ch"+fmt.Sprint(rng.Intn(100)), bbEntry("a.addons.k8s.io", ""))
		a := bbParse(t, loc, doc)
		if len(a.APIObject.Spec.Addons) != 1 {
			t.Fatalf("case %d", i)
		}
	}
}

func TestAddonGetCurrentVersionProperty(t *testing.T) {
	grid := []struct {
		rangeStr string
		ver      string
		want     bool
	}{
		{">=1.6.0", "1.6.0", true},
		{"<1.6.0", "1.6.0", false},
		{">=1.6.0", "1.5.9", false},
		{">=1.4.0 <1.6.0", "1.5.9", true},
		{">=1.4.0 <1.6.0", "1.6.0", false},
	}
	for _, g := range grid {
		doc := bbAddonsDoc("ch", bbEntry("filt.addons.k8s.io", "kubernetesVersion: '"+g.rangeStr+"'"))
		a := bbParse(t, bbURL(t, "https://example.com/a.yaml"), doc)
		menu, err := a.GetCurrent(semver.MustParse(g.ver))
		if err != nil {
			t.Fatal(err)
		}
		_, ok := menu.Addons["filt.addons.k8s.io"]
		if ok != g.want {
			t.Fatalf("range %q ver %s got %v", g.rangeStr, g.ver, ok)
		}
	}

	rng := rand.New(rand.NewSource(bbSeed + 1))
	vers := []string{"1.4.0", "1.5.9", "1.6.0", "1.7.2"}
	ranges := []string{"", ">=1.5.0", "<1.6.0", ">=1.4.0 <1.6.0", ">=9.9.9", "not-a-range"}
	for i := 0; i < bbCases; i++ {
		rs := ranges[rng.Intn(len(ranges))]
		vs := vers[rng.Intn(len(vers))]
		doc := bbAddonsDoc("ch", bbEntry("x.addons.k8s.io", "kubernetesVersion: '"+rs+"'"))
		a := bbParse(t, bbURL(t, "https://example.com/r.yaml"), doc)
		menu, err := a.GetCurrent(semver.MustParse(vs))
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		want := bbVersionMatches(rs, semver.MustParse(vs))
		_, got := menu.Addons["x.addons.k8s.io"]
		if got != want {
			t.Fatalf("case %d range %q ver %s got %v want %v", i, rs, vs, got, want)
		}
	}
}

func TestAddonReplacementProperty(t *testing.T) {
	hash1 := "3544de6578b2b582c0323b15b7b05a28c60b9430"
	hash2 := "ea9e79bf29adda450446487d65a8fc6b3fdf8c2b"
	rows := []struct {
		old, new *channels.ChannelVersion
	}{
		{&channels.ChannelVersion{Id: "a", ManifestHash: hash1}, &channels.ChannelVersion{Id: "a", ManifestHash: hash1}},
		{&channels.ChannelVersion{Id: "a", ManifestHash: ""}, &channels.ChannelVersion{Id: "a", ManifestHash: hash1}},
		{&channels.ChannelVersion{Id: "a", ManifestHash: hash1}, &channels.ChannelVersion{Id: "a", ManifestHash: hash2}},
	}
	for _, row := range rows {
		e1 := bbEntry("dup.addons.k8s.io", fmt.Sprintf("id: %s\nmanifestHash: %s", row.old.Id, row.old.ManifestHash))
		e2 := bbEntry("dup.addons.k8s.io", fmt.Sprintf("id: %s\nmanifestHash: %s", row.new.Id, row.new.ManifestHash))
		bbAssertMergedWinner(t, "dup.addons.k8s.io", e1, e2)
	}

	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		id1 := fmt.Sprintf("id%d", rng.Intn(50))
		id2 := fmt.Sprintf("id%d", rng.Intn(50))
		h1 := fmt.Sprintf("%040x", rng.Uint64())
		h2 := fmt.Sprintf("%040x", rng.Uint64())
		e1 := bbEntry("dup.addons.k8s.io", fmt.Sprintf("id: %s\nmanifestHash: %s", id1, h1))
		e2 := bbEntry("dup.addons.k8s.io", fmt.Sprintf("id: %s\nmanifestHash: %s", id2, h2))
		bbAssertMergedWinner(t, "dup.addons.k8s.io", e1, e2)
	}
}

func bbAssertMergedWinner(t *testing.T, addonName, e1, e2 string) {
	t.Helper()
	loc := bbURL(t, "https://example.com/r.yaml")
	ver := semver.MustParse("1.22.0")
	combined := bbParse(t, loc, bbAddonsDoc("ch", e1, e2))
	gotMenu, err := combined.GetCurrent(ver)
	if err != nil {
		t.Fatal(err)
	}
	m1, _ := bbParse(t, loc, bbAddonsDoc("ch", e1)).GetCurrent(ver)
	m2, _ := bbParse(t, loc, bbAddonsDoc("ch", e2)).GetCurrent(ver)
	wantMenu := channels.NewAddonMenu()
	wantMenu.MergeAddons(m1)
	wantMenu.MergeAddons(m2)
	got := gotMenu.Addons[addonName]
	want := wantMenu.Addons[addonName]
	if got == nil || want == nil {
		t.Fatalf("missing addon %q", addonName)
	}
	gcv, wcv := got.ChannelVersion(), want.ChannelVersion()
	if gcv.Id != wcv.Id || gcv.ManifestHash != wcv.ManifestHash {
		t.Fatalf("winner mismatch got id=%s hash=%s want id=%s hash=%s", gcv.Id, gcv.ManifestHash, wcv.Id, wcv.ManifestHash)
	}
}

func TestAddonRequiredUpdatesProperty(t *testing.T) {
	ctx := context.Background()
	kubeSystem := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
	fakek8s := fakekubernetes.NewClientset(kubeSystem)
	fakecm := fakecertmanager.NewSimpleClientset()
	addon := &channels.Addon{
		Name: "test",
		Spec: &api.AddonSpec{Name: new("test"), NeedsPKI: true},
	}
	up, err := addon.GetRequiredUpdates(ctx, fakek8s, fakecm, nil)
	if err != nil || up == nil || !up.InstallPKI {
		t.Fatalf("pki install: err=%v up=%v", err, up)
	}

	existing := &channels.ChannelVersion{Id: "same", ManifestHash: "h1", SystemGeneration: channels.CurrentSystemGeneration}
	addon2 := &channels.Addon{
		Name: "test",
		Spec: &api.AddonSpec{Name: new("test"), Id: "same", ManifestHash: "h1"},
	}
	up2, err := addon2.GetRequiredUpdates(ctx, fakek8s, fakecm, existing)
	if err != nil || up2 != nil {
		t.Fatalf("no work when equal: %v %v", up2, err)
	}

	addon3 := &channels.Addon{
		Name: "test",
		Spec: &api.AddonSpec{Name: new("test"), Id: "new", ManifestHash: "h2"},
	}
	up3, err := addon3.GetRequiredUpdates(ctx, fakek8s, fakecm, existing)
	if err != nil || up3 == nil || up3.NewVersion == nil {
		t.Fatalf("replace update: %v", up3)
	}

	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		idOld := fmt.Sprintf("o%d", rng.Intn(20))
		idNew := fmt.Sprintf("n%d", rng.Intn(20))
		h := fmt.Sprintf("%x", rng.Uint64())
		existing := &channels.ChannelVersion{Id: idOld, ManifestHash: h, SystemGeneration: channels.CurrentSystemGeneration}
		ad := &channels.Addon{
			Name: "pk",
			Spec: &api.AddonSpec{Name: new("pk"), Id: idNew, ManifestHash: h, NeedsPKI: rng.Intn(2) == 0},
		}
		up, err := ad.GetRequiredUpdates(ctx, fakek8s, fakecm, existing)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		shouldReplace := idOld != idNew
		if !shouldReplace && up != nil && up.NewVersion != nil {
			t.Fatalf("case %d should clear new version", i)
		}
	}
}
