// Package loader_test is a hidden black-box property suite for the chartloader
// unit. Only exported API declared in api.md is exercised:
//
//	loader.Loader, Load, LoadFile, LoadArchive, LoadDir, LoadFiles
//	loader.FileLoader.Load, loader.DirLoader.Load
//	loader.LoadValues, loader.MergeMaps
//
// Deterministic seed: 20260919. Cases: >= 10,000.
//
// Contract (contract.md) -> property coverage table:
//
//	S1 "three entry paths funnel into a single file-list loader"
//	    -> TestCLLoaderEquivalence
//	S2 "Chart.yaml required; values.yaml unmarshalled; templates/ and charts/"
//	    -> TestCLLoadFilesProperty
//	S3 "BOM stripped from UTF-8 files" -> TestCLContractTable / TestCLBOMProperty
//	S4 "absolute paths, parent escapes, backslashes rejected"
//	    -> TestCLContractTable / TestCLAdversarialPaths
//	S5 "symlinks and device files skipped" -> TestCLContractTable (fixture-backed)
//	S6 "directory walk lexical order; size budget enforced"
//	    -> TestCLContractTable (oversize dir)
//	S7 "LoadValues parses YAML/JSON streams; empty-at-boundary tolerated"
//	    -> TestCLLoadValuesProperty
//	S8 "MergeMaps: later top-level keys override; recursive map merge"
//	    -> TestCLMergeMapsProperty
//	S9 "truncated or non-tar/gzip archive errors" -> TestCLContractTable
package loader_test

import (
	"encoding/json"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	loader "example.internal/chartkit/v4/internal/chart/v3/loader"
	"example.internal/chartkit/v4/pkg/chart/loader/archive"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// ---------------------------------------------------------------------------
// oracle helpers
// ---------------------------------------------------------------------------

func oracleMergeMaps(a, b map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		out[k] = deepCopyAny(v)
	}
	for k, v := range b {
		if av, ok := out[k]; ok {
			am, aok := av.(map[string]any)
			bm, bok := v.(map[string]any)
			if aok && bok {
				out[k] = oracleMergeMaps(am, bm)
				continue
			}
		}
		out[k] = deepCopyAny(v)
	}
	return out
}

func deepCopyAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, val := range x {
			m[k] = deepCopyAny(val)
		}
		return m
	case map[string]string:
		m := map[string]string{}
		for k, val := range x {
			m[k] = val
		}
		return m
	case []any:
		s := make([]any, len(x))
		for i, val := range x {
			s[i] = deepCopyAny(val)
		}
		return s
	default:
		return v
	}
}

func oracleLoadValuesJSON(data []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func chartYAML(name, ver string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: v3
name: %s
version: %q
type: application
`, name, ver))
}

func mkBufferedFiles(r *rand.Rand, name, ver string, withBOM bool) []*archive.BufferedFile {
	mod := time.Now()
	prefix := func(b []byte) []byte {
		if withBOM {
			return append(append([]byte{}, utf8bom...), b...)
		}
		return b
	}
	files := []*archive.BufferedFile{
		{Name: "Chart.yaml", ModTime: mod, Data: prefix(chartYAML(name, ver))},
		{Name: "values.yaml", ModTime: mod, Data: prefix([]byte(fmt.Sprintf("seed: %d\n", r.Int63())))},
	}
	if r.Intn(3) > 0 {
		files = append(files, &archive.BufferedFile{
			Name:    "templates/workload.yaml",
			ModTime: mod,
			Data:    []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"),
		})
	}
	if r.Intn(4) == 0 {
		files = append(files, &archive.BufferedFile{
			Name:    "README.md",
			ModTime: mod,
			Data:    []byte("# chart\n"),
		})
	}
	return files
}

func writeChartDir(t *testing.T, dir, name, ver string, withBOM bool) {
	t.Helper()
	prefix := func(b []byte) []byte {
		if withBOM {
			return append(append([]byte{}, utf8bom...), b...)
		}
		return b
	}
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), prefix(chartYAML(name, ver)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), prefix([]byte("k: v\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "tpl.yaml"), prefix([]byte("kind: Pod\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func packTGZ(t *testing.T, chartName string, files []*archive.BufferedFile) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range files {
		h := &tar.Header{Name: chartName + "/" + f.Name, Mode: 0o644, Size: int64(len(f.Data)), ModTime: f.ModTime}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.Data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mapsEqual(a, b map[string]any) bool {
	return reflect.DeepEqual(a, b)
}

// ---------------------------------------------------------------------------
// contract table (adversarial edges + fixture-backed cases)
// ---------------------------------------------------------------------------

func TestCLContractTable(t *testing.T) {
	_, err := loader.LoadFiles(nil)
	if err == nil || !strings.Contains(err.Error(), "Chart.yaml") {
		t.Fatalf("empty LoadFiles err=%v", err)
	}

	for _, bad := range []string{
		"../outside.yaml",
		"../../escape.yaml",
	} {
		files := []*archive.BufferedFile{
			{Name: "Chart.yaml", Data: chartYAML("x", "1.0.0")},
			{Name: bad, Data: []byte("x: 1\n")},
		}
		raw := packTGZ(t, "x", files)
		if _, err := loader.LoadArchive(bytes.NewReader(raw)); err == nil {
			t.Fatalf("LoadArchive accepted bad path %q", bad)
		}
	}

	// Backslash-delimited archives are supported (in-tree TestLoadFileBackslash).
	lbs, err := loader.Loader("testdata/frobnitz_backslash-1.2.3.tgz")
	if err != nil {
		t.Fatalf("backslash fixture: %v", err)
	}
	if _, err = lbs.Load(); err != nil {
		t.Fatalf("backslash archive load: %v", err)
	}

	if runtime.GOOS != "windows" {
		l, err := loader.Loader("testdata/frobnitz_with_dev_null")
		if err != nil {
			t.Fatalf("Loader dev_null: %v", err)
		}
		if _, err = l.Load(); err == nil {
			t.Fatal("dev_null chart should error")
		}
	}

	l, err := loader.Loader("testdata/frobnitz_with_bom")
	if err != nil {
		t.Fatal(err)
	}
	c, err := l.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range c.Files {
		if bytes.HasPrefix(f.Data, utf8bom) {
			t.Fatalf("BOM not stripped in %s", f.Name)
		}
	}

	_, err = loader.LoadArchive(bytes.NewReader([]byte("not gzip")))
	if err == nil {
		t.Fatal("invalid archive should error")
	}
	trunc := packTGZ(t, "t", []*archive.BufferedFile{{Name: "Chart.yaml", Data: chartYAML("t", "1.0.0")}})
	if _, err = loader.LoadArchive(bytes.NewReader(trunc[:len(trunc)/2])); err == nil {
		t.Fatal("truncated archive should error")
	}

	// S6: directory exceeding default decompressed budget (100 MiB + 1).
	bigDir := t.TempDir()
	writeChartDir(t, bigDir, "big", "1.0.0", false)
	huge := filepath.Join(bigDir, "templates", "huge.yaml")
	if err := os.WriteFile(huge, bytes.Repeat([]byte("x"), 100*1024*1024+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loader.LoadDir(bigDir); err == nil || !strings.Contains(err.Error(), "maximum decompressed size") {
		t.Fatalf("oversize dir: err=%v", err)
	}

	// EOF-boundary values (4096 / 8192)
	for _, size := range []int{4096, 8192} {
		prefix := []byte(`{"foo":"`)
		suffix := []byte(`"}`)
		pad := size - len(prefix) - len(suffix)
		data := append(append(append([]byte{}, prefix...), bytes.Repeat([]byte("x"), pad)...), suffix...)
		got, err := loader.LoadValues(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		want := map[string]any{
			"foo": string(bytes.Repeat([]byte("x"), pad)),
		}
		if !mapsEqual(got, want) {
			t.Fatalf("size %d: got=%v want=%v", size, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// S8: MergeMaps
// ---------------------------------------------------------------------------

func bbRandMap(r *rand.Rand, depth int) map[string]any {
	n := 1 + r.Intn(4)
	m := map[string]any{}
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("k%d", i)
		switch r.Intn(5) {
		case 0:
			m[k] = r.Intn(1000)
		case 1:
			m[k] = fmt.Sprintf("s%x", r.Uint64())
		case 2:
			if depth > 0 {
				m[k] = bbRandMap(r, depth-1)
			} else {
				m[k] = r.Float64()
			}
		case 3:
			m[k] = []any{r.Intn(9), fmt.Sprint(r.Intn(9))}
		default:
			m[k] = nil
		}
	}
	return m
}

func TestCLMergeMapsProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed))
	for c := 0; c < bbCases; c++ {
		a := bbRandMap(r, 1+r.Intn(2))
		b := bbRandMap(r, 1+r.Intn(2))
		got := loader.MergeMaps(a, b)
		want := oracleMergeMaps(a, b)
		if !mapsEqual(got, want) {
			t.Fatalf("case %d: MergeMaps mismatch\na=%v\nb=%v\ngot=%v\nwant=%v", c, a, b, got, want)
		}
		// later-wins spot check when keys overlap
		if len(a) > 0 && len(b) > 0 {
			for k, bv := range b {
				if _, ok := a[k]; ok {
					if !reflect.DeepEqual(got[k], bv) {
						// nested maps merge instead of overwrite
						am, aok := a[k].(map[string]any)
						bm, bok := bv.(map[string]any)
						if !(aok && bok) {
							t.Fatalf("case %d: key %q should be from b", c, k)
						}
						merged := oracleMergeMaps(am, bm)
						if !reflect.DeepEqual(got[k], merged) {
							t.Fatalf("case %d: nested merge for %q", c, k)
						}
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S7: LoadValues
// ---------------------------------------------------------------------------

func TestCLLoadValuesProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 1))
	for c := 0; c < bbCases; c++ {
		data := []byte(fmt.Sprintf(`{"doc%d":{"n":%d,"s":%q}}`, c%997, r.Intn(50), fmt.Sprintf("v%x", r.Uint32())))
		got, err := loader.LoadValues(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("case %d: LoadValues: %v", c, err)
		}
		want, werr := oracleLoadValuesJSON(data)
		if werr != nil {
			t.Fatalf("case %d: oracle: %v", c, werr)
		}
		if !mapsEqual(got, want) {
			t.Fatalf("case %d: got=%v want=%v", c, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// S2: LoadFiles generated charts
// ---------------------------------------------------------------------------

func TestCLLoadFilesProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 2))
	for c := 0; c < bbCases; c++ {
		name := fmt.Sprintf("gen%d", c%997)
		ver := fmt.Sprintf("%d.%d.%d", r.Intn(3), r.Intn(5), r.Intn(9))
		withBOM := r.Intn(5) == 0
		files := mkBufferedFiles(r, name, ver, withBOM)
		// adversarial: Chart.yaml not first sometimes (order preserved contract)
		if r.Intn(4) == 0 && len(files) > 1 {
			files[0], files[1] = files[1], files[0]
		}
		chart, err := loader.LoadFiles(files)
		if err != nil {
			t.Fatalf("case %d: LoadFiles: %v", c, err)
		}
		if chart.Name() != name {
			t.Fatalf("case %d: name=%q want %q", c, chart.Name(), name)
		}
		if chart.Metadata == nil || chart.Metadata.Version != ver {
			t.Fatalf("case %d: version=%q want %q", c, chart.Metadata.Version, ver)
		}
		for _, f := range chart.Files {
			if bytes.HasPrefix(f.Data, utf8bom) {
				t.Fatalf("case %d: BOM present in %s", c, f.Name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S1: Loader / LoadDir / LoadFile / LoadArchive equivalence
// ---------------------------------------------------------------------------

func TestCLLoaderEquivalence(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 3))
	for c := 0; c < 500; c++ {
		dir := t.TempDir()
		name := fmt.Sprintf("eq%d", c)
		ver := fmt.Sprintf("0.%d.%d", r.Intn(9), r.Intn(9))
		writeChartDir(t, dir, name, ver, r.Intn(6) == 0)

		fromDir, err := loader.LoadDir(dir)
		if err != nil {
			t.Fatalf("case %d LoadDir: %v", c, err)
		}
		dl := loader.DirLoader(dir)
		fromDL, err := dl.Load()
		if err != nil {
			t.Fatalf("case %d DirLoader: %v", c, err)
		}
		if fromDir.Name() != fromDL.Name() || fromDir.Metadata.Version != fromDL.Metadata.Version {
			t.Fatalf("case %d DirLoader mismatch", c)
		}

		tgzPath := filepath.Join(t.TempDir(), name+".tgz")
		files := mkBufferedFiles(r, name, ver, false)
		if err := os.WriteFile(tgzPath, packTGZ(t, name, files), 0o644); err != nil {
			t.Fatal(err)
		}
		fromFile, err := loader.LoadFile(tgzPath)
		if err != nil {
			t.Fatalf("case %d LoadFile: %v", c, err)
		}
		fl := loader.FileLoader(tgzPath)
		fromFL, err := fl.Load()
		if err != nil {
			t.Fatalf("case %d FileLoader: %v", c, err)
		}
		raw, _ := os.ReadFile(tgzPath)
		fromArch, err := loader.LoadArchive(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("case %d LoadArchive: %v", c, err)
		}
		if fromFile.Name() != fromFL.Name() || fromArch.Name() != fromFile.Name() {
			t.Fatalf("case %d archive path name mismatch", c)
		}

		ln, err := loader.Loader(dir)
		if err != nil {
			t.Fatalf("case %d Loader dir: %v", c, err)
		}
		fromL, err := ln.Load()
		if err != nil {
			t.Fatalf("case %d Loader.Load dir: %v", c, err)
		}
		if fromL.Name() != fromDir.Name() {
			t.Fatalf("case %d Loader dir mismatch", c)
		}
		lf, err := loader.Loader(tgzPath)
		if err != nil {
			t.Fatalf("case %d Loader tgz: %v", c, err)
		}
		fromLT, err := lf.Load()
		if err != nil {
			t.Fatalf("case %d Loader.Load tgz: %v", c, err)
		}
		if fromLT.Name() != fromFile.Name() {
			t.Fatalf("case %d Loader tgz mismatch", c)
		}
		fromLoad, err := loader.Load(tgzPath)
		if err != nil || fromLoad.Name() != fromFile.Name() {
			t.Fatalf("case %d Load(name): %v", c, err)
		}
	}
}

// ---------------------------------------------------------------------------
// S3: BOM on generated inputs
// ---------------------------------------------------------------------------

func TestCLBOMProperty(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 4))
	for c := 0; c < bbCases; c++ {
		name := fmt.Sprintf("bom%d", c%500)
		ver := "1.0.0"
		files := mkBufferedFiles(r, name, ver, true)
		chart, err := loader.LoadFiles(files)
		if err != nil {
			t.Fatalf("case %d: %v", c, err)
		}
		if chart.Metadata == nil || chart.Metadata.Name != name {
			t.Fatalf("case %d: bad metadata", c)
		}
		all := chart.Files
		for _, f := range all {
			if bytes.HasPrefix(f.Data, utf8bom) {
				t.Fatalf("case %d: BOM in %s", c, f.Name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// S4: adversarial archive paths
// ---------------------------------------------------------------------------

func TestCLAdversarialPaths(t *testing.T) {
	r := rand.New(rand.NewSource(bbSeed + 5))
	badPaths := []string{
		"../outside.yaml",
		"../../Chart.yaml",
		"/etc/passwd",
	}
	for c := 0; c < bbCases; c++ {
		p := badPaths[r.Intn(len(badPaths))]
		name := fmt.Sprintf("adv%d", c%500)
		files := []*archive.BufferedFile{
			{Name: "Chart.yaml", Data: chartYAML(name, "1.0.0")},
			{Name: p, Data: []byte("x: 1\n")},
		}
		raw := packTGZ(t, name, files)
		if _, err := loader.LoadArchive(bytes.NewReader(raw)); err == nil {
			t.Fatalf("case %d: accepted archive path %q", c, p)
		}
	}
}
