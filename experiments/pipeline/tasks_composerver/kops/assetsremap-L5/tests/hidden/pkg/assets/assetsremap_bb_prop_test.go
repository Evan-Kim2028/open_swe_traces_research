// Package assets_test — hidden black-box property suite for assetsremap.
// Exported API only (api.md): NewAssetBuilder, RemapImage, RemapManifest, RemapFile,
// NormalizeImage, ImageAssets, FileAssets. Seed 20260919; >=10k cases per file.
//
// Contract (contract.md) -> property coverage table:
//
//	"hub image gets proxy prepended" -> TestAssetsRemapImageContractTableProperty
//	"dotted first segment is a host and is replaced" -> TestAssetsRemapImageContractTableProperty
//	"tags survive proxy rewrite" -> TestAssetsRemapImageContractTableProperty
//	"second registry pass does not double-prefix" -> TestAssetsRemapRegistryConvergeProperty
//	"commas in file paths are %2C" -> TestAssetsRemapFileContractTableProperty
//	"empty YAML section does not panic" -> TestAssetsRemapManifestProperty
//	"concurrent remap + sorted snapshot getters" -> TestAssetsRemapSortedSnapshotsProperty
//	"nil URL is an error" -> TestAssetsRemapFileContractTableProperty
package assets_test

import (
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/assets"
	"example.internal/clustkit/util/pkg/hashing"
)

const bbSeed = 20260919
const bbCases = 10000

func bbNewBuilder(proxy, registry, fileRepo *string) *assets.AssetBuilder {
	spec := &kops.AssetsSpec{
		ContainerProxy:    proxy,
		ContainerRegistry: registry,
		FileRepository:    fileRepo,
	}
	return assets.NewAssetBuilder(nil, spec, false)
}

func bbOracleNormalize(proxy, registry *string, image string) string {
	if proxy != nil {
		containerProxy := strings.TrimSuffix(*proxy, "/")
		normalized := image
		if strings.Count(normalized, "/") <= 1 && !strings.ContainsAny(strings.Split(normalized, "/")[0], ".:") {
			normalized = containerProxy + "/" + normalized
		} else {
			re := regexp.MustCompile(`^[^/]+`)
			normalized = re.ReplaceAllString(normalized, containerProxy)
		}
		image = normalized
	}
	if registry != nil {
		registryMirror := *registry
		normalized := strings.TrimPrefix(image, "registry.k8s.io/")
		if !strings.HasPrefix(normalized, registryMirror+"/") {
			normalized = strings.ReplaceAll(normalized, "/", "-")
			normalized = registryMirror + "/" + normalized
		}
		image = normalized
	}
	return image
}

func TestAssetsRemapImageContractTableProperty(t *testing.T) {
	proxy := "proxy.example.com/"
	b := bbNewBuilder(&proxy, nil, nil)
	cases := []struct {
		in, want string
	}{
		{"weaveworks/weave-kube", "proxy.example.com/weaveworks/weave-kube"},
		{"debian", "proxy.example.com/debian"},
		{"registry.k8s.io/kube-apiserver", "proxy.example.com/kube-apiserver"},
		{"gcr.io/google_containers/kube-apiserver", "proxy.example.com/google_containers/kube-apiserver"},
		{"registry.k8s.io/kube-apiserver:1.2.3", "proxy.example.com/kube-apiserver:1.2.3"},
	}
	for _, tc := range cases {
		got := b.RemapImage(tc.in)
		if got != tc.want {
			t.Fatalf("RemapImage(%q)=%q want %q", tc.in, got, tc.want)
		}
		wantNorm := assets.NormalizeImage(b, tc.in)
		if wantNorm != tc.want {
			t.Fatalf("NormalizeImage(%q)=%q want %q", tc.in, wantNorm, tc.want)
		}
	}
}

func TestAssetsRemapRegistryConvergeProperty(t *testing.T) {
	mirror := "proxy.example.com"
	b := bbNewBuilder(nil, &mirror, nil)
	image := "kube-apiserver:1.2.3"
	want := "proxy.example.com/kube-apiserver:1.2.3"
	for i := 0; i < 2; i++ {
		image = b.RemapImage(image)
		if image != want {
			t.Fatalf("iter %d got %q", i, image)
		}
	}

	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		m := fmt.Sprintf("reg%d.example.com", rng.Intn(50))
		b := bbNewBuilder(nil, &m, nil)
		img := fmt.Sprintf("registry.k8s.io/ns/app:v%d", rng.Intn(20))
		first := b.RemapImage(img)
		second := b.RemapImage(first)
		if first != second {
			t.Fatalf("case %d non-convergent %q -> %q", i, first, second)
		}
		if first != bbOracleNormalize(nil, &m, img) {
			t.Fatalf("case %d oracle", i)
		}
	}
}

func TestAssetsRemapFileContractTableProperty(t *testing.T) {
	canonical, err := url.Parse("https://artifacts.k8s.io/binaries/kops/1.37.0/linux/arm64/nodeup")
	if err != nil {
		t.Fatal(err)
	}
	known := hashing.MustFromString("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	repo := "s3://artifact-bucket/prefix,prod"
	b := bbNewBuilder(nil, nil, &repo)
	asset, err := b.RemapFile(canonical, known)
	if err != nil {
		t.Fatal(err)
	}
	want := "s3://artifact-bucket/prefix%2Cprod/binaries/kops/1.37.0/linux/arm64/nodeup"
	if asset.DownloadURL.String() != want {
		t.Fatalf("download url %q", asset.DownloadURL.String())
	}
	b0 := bbNewBuilder(nil, nil, nil)
	if _, err := b0.RemapFile(nil, known); err == nil {
		t.Fatal("nil url")
	}
}

func TestAssetsRemapManifestProperty(t *testing.T) {
	b := bbNewBuilder(nil, nil, nil)
	empty := []byte("---\n")
	if _, err := b.RemapManifest(empty); err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		img := fmt.Sprintf("debian:%d", rng.Intn(100))
		doc := fmt.Sprintf("apiVersion: v1\nkind: Pod\nspec:\n  containers:\n  - image: %s\n", img)
		out, err := b.RemapManifest([]byte(doc))
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !strings.Contains(string(out), img) {
			t.Fatalf("case %d image preserved", i)
		}
	}
}

func TestAssetsRemapSortedSnapshotsProperty(t *testing.T) {
	b := bbNewBuilder(nil, nil, nil)
	known := hashing.MustFromString("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			b.RemapImage(fmt.Sprintf("registry.k8s.io/img-%d:latest", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			u, _ := url.Parse(fmt.Sprintf("https://example.com/f-%d", i))
			_, _ = b.RemapFile(u, known)
		}(i)
	}
	wg.Wait()
	imgs := b.ImageAssets()
	for j := 1; j < len(imgs); j++ {
		if imgs[j-1].CanonicalLocation > imgs[j].CanonicalLocation {
			t.Fatal("image assets not sorted")
		}
	}
	files := b.FileAssets()
	for j := 1; j < len(files); j++ {
		if files[j-1].CanonicalURL.String() > files[j].CanonicalURL.String() {
			t.Fatal("file assets not sorted")
		}
	}

	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		b2 := bbNewBuilder(nil, nil, nil)
		n := 1 + rng.Intn(5)
		for k := 0; k < n; k++ {
			b2.RemapImage(fmt.Sprintf("img%d", k))
		}
		prev := ""
		for _, a := range b2.ImageAssets() {
			if a.CanonicalLocation < prev {
				t.Fatalf("case %d sort", i)
			}
			prev = a.CanonicalLocation
		}
	}
}
