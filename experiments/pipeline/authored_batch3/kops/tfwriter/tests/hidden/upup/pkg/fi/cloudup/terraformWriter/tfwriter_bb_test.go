// Package terraformWriter_test is the hidden black-box suite for tfwriter.
// One TestDetailNN per DETAILS.md commitment. The sanitize map is
// solver-derivable (the contract states `.->-`, `/->--`, `:->_`,
// digit-first -> `prefix_`).
package terraformWriter_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	tw "example.internal/clustkit/upup/pkg/fi/cloudup/terraformWriter"
)

func newWriter() *tw.TerraformWriter {
	w := &tw.TerraformWriter{}
	w.InitTerraformWriter()
	return w
}

// Detail 1 (Inferable: no — but the contract states the map): `.` -> `-`,
// `/` -> `--`, `:` -> `_`, leading digit -> `prefix_`.
func TestDetail01(t *testing.T) {
	w := newWriter()
	if err := w.RenderResource("t", "a.b/c:d", struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := w.RenderResource("t", "9lives", struct{}{}); err != nil {
		t.Fatal(err)
	}
	m, err := w.GetResourcesByType()
	if err != nil {
		t.Fatal(err)
	}
	got := m["t"]
	if _, ok := got["a-b--c_d"]; !ok {
		t.Fatalf("a.b/c:d not legalized as contract map: %v", got)
	}
	if _, ok := got["prefix_9lives"]; !ok {
		t.Fatalf("digit-first name not prefixed: %v", got)
	}
}

// Detail 2 (Inferable: partially): legalized-name collisions are detected at
// query time — the Get* calls error.
func TestDetail02(t *testing.T) {
	w := newWriter()
	// inserts succeed (detection is deferred to query)
	if err := w.RenderResource("t", "a.b", struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := w.RenderResource("t", "a-b", struct{}{}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.GetResourcesByType(); err == nil {
		t.Fatal("legalized-name collision not detected at query time")
	}
}

// Detail 3 (Inferable: partially): items group by type then legalized name;
// same name under different types does NOT collide.
func TestDetail03(t *testing.T) {
	w := newWriter()
	if err := w.RenderResource("t1", "n", "x"); err != nil {
		t.Fatal(err)
	}
	if err := w.RenderResource("t2", "n", "y"); err != nil {
		t.Fatal(err)
	}
	m, err := w.GetResourcesByType()
	if err != nil {
		t.Fatalf("same name across types collided: %v", err)
	}
	if len(m) != 2 || m["t1"]["n"] == nil || m["t2"]["n"] == nil {
		t.Fatalf("grouping wrong: %v", m)
	}
	if err := w.RenderDataSource("t1", "n", "x"); err != nil {
		t.Fatal(err)
	}
	dm, err := w.GetDataSourcesByType()
	if err != nil || dm["t1"]["n"] == nil {
		t.Fatalf("data source grouping wrong: %v %v", dm, err)
	}
}

// Detail 4 (Inferable: no): duplicate scalar output key errors; an array
// output on a scalar key errors differently. Asserted shape: both error.
func TestDetail04(t *testing.T) {
	w := newWriter()
	l := tw.LiteralTokens("a")
	if err := w.AddOutputVariable("k", l); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariable("k", l); err == nil {
		t.Fatal("duplicate scalar key accepted")
	}
	if err := w.AddOutputVariableArray("k", l); err == nil {
		t.Fatal("array on scalar key accepted")
	}
}

// Detail 5 (Inferable: yes): AddOutputVariableArray on an absent key creates
// the entry; repeat calls accumulate.
func TestDetail05(t *testing.T) {
	w := newWriter()
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("b")); err != nil {
		t.Fatal(err)
	}
	outs, err := w.GetOutputs()
	if err != nil {
		t.Fatal(err)
	}
	if len(outs["k"].ValueArray) != 2 {
		t.Fatalf("array outputs did not accumulate: %v", outs["k"])
	}
}

// Detail 6 (Inferable: partially): GetOutputs legalizes keys (collisions are
// errors) and dedups+sorts ValueArray while leaving scalar Value untouched.
func TestDetail06(t *testing.T) {
	w := newWriter()
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("b")); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.AddOutputVariableArray("k", tw.LiteralTokens("a")); err != nil {
		t.Fatal(err)
	}
	scal := tw.LiteralTokens("s")
	if err := w.AddOutputVariable("solo", scal); err != nil {
		t.Fatal(err)
	}
	outs, err := w.GetOutputs()
	if err != nil {
		t.Fatal(err)
	}
	if arr := outs["k"].ValueArray; len(arr) != 2 || arr[0].String != "a" || arr[1].String != "b" {
		t.Fatalf("ValueArray not deduped+sorted: %v", arr)
	}
	if outs["solo"].Value != scal {
		t.Fatal("scalar Value touched")
	}

	w2 := newWriter()
	if err := w2.AddOutputVariable("k.a", tw.LiteralTokens("x")); err != nil {
		t.Fatal(err)
	}
	if err := w2.AddOutputVariable("k-a", tw.LiteralTokens("y")); err != nil {
		t.Fatal(err)
	}
	if _, err := w2.GetOutputs(); err == nil {
		t.Fatal("legalized output-key collision not detected")
	}
}

// Detail 7 (Inferable: no): EnsureTerraformProvider returns the existing
// provider on exact-arg match and aborts the process on conflict.
func TestDetail07(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		w := newWriter()
		w.EnsureTerraformProvider("aws", map[string]string{"a": "1"})
		w.EnsureTerraformProvider("aws", map[string]string{"a": "2"})
		return
	}
	w := newWriter()
	p1 := w.EnsureTerraformProvider("aws", map[string]string{"a": "1"})
	p2 := w.EnsureTerraformProvider("aws", map[string]string{"a": "1"})
	if p1 != p2 {
		t.Fatal("identical re-registration did not return existing")
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestDetail07")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("conflicting registration did not abort: %s", out)
	}
}

// Detail 8 (Inferable: partially): InitTerraformWriter initializes only the
// file and output maps; provider/resource collections lazy-init on first
// use. Asserted shape: after init, Files is usable and Providers is empty.
func TestDetail08(t *testing.T) {
	w := &tw.TerraformWriter{}
	w.InitTerraformWriter()
	if w.Files == nil {
		t.Fatal("Files not initialized")
	}
	if len(w.Providers) != 0 {
		t.Fatal("Providers eagerly initialized")
	}
	// everything still works
	if err := w.RenderResource("t", "n", "x"); err != nil {
		t.Fatal(err)
	}
	w.EnsureTerraformProvider("p", map[string]string{})
	if len(w.Providers) != 1 {
		t.Fatal("Providers did not lazy-init")
	}
}

// Detail 9 (Inferable: partially): AddFilePath stages bytes under data/<id>
// and returns a module-path literal; AddFileBytes wraps it in a file-call.
func TestDetail09(t *testing.T) {
	w := newWriter()
	lit, err := w.AddFilePath("mytype", "myname", "mykey", []byte("data1"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Files) != 1 {
		t.Fatalf("file not staged: %v", w.Files)
	}
	for path, contents := range w.Files {
		if !strings.HasPrefix(path, "data/") {
			t.Fatalf("file not staged under data/: %q", path)
		}
		if string(contents) != "data1" {
			t.Fatalf("wrong content: %q", contents)
		}
	}
	if !strings.Contains(lit.String, "path.module") || !strings.Contains(lit.String, "data/") {
		t.Fatalf("literal not a module path: %q", lit.String)
	}
	litB, err := w.AddFileBytes("mytype", "myname", "mykey", []byte("x"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(litB.String, "file") {
		t.Fatalf("not wrapped in a file call: %q", litB.String)
	}
}

// Detail 10 (Inferable: yes): RenderResource/RenderDataSource always succeed.
func TestDetail10(t *testing.T) {
	w := newWriter()
	for _, name := range []string{"a", "a", "weird.name/here:x", ""} {
		if err := w.RenderResource("t", name, "v"); err != nil {
			t.Fatalf("RenderResource(%q): %v", name, err)
		}
		if err := w.RenderDataSource("t", name, "v"); err != nil {
			t.Fatalf("RenderDataSource(%q): %v", name, err)
		}
	}
}
