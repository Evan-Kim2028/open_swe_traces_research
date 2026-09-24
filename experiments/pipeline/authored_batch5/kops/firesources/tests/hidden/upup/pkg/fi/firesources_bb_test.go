package fi

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"example.internal/clustkit/util/pkg/vfs"
)

type bbFailRes struct{ err error }

func (r *bbFailRes) Open() (io.Reader, error) { return nil, r.err }

type bbFailReader struct{}

func (r *bbFailReader) Read([]byte) (int, error) { return 0, errors.New("bb read failure") }

type bbFailMidRes struct{}

func (r *bbFailMidRes) Open() (io.Reader, error) { return &bbFailReader{}, nil }

type bbTask struct{}

func (t *bbTask) Run(*Context[CloudupSubContext]) error { return nil }

// TestDetail01: ResourcesMatch streams and compares all bytes; different
// lengths are false (not an error); open/read errors propagate.
func TestDetail01(t *testing.T) {
	if ok, err := ResourcesMatch(NewStringResource("abc"), NewStringResource("abc")); err != nil || !ok {
		t.Fatalf("equal resources: %v %v", ok, err)
	}
	if ok, err := ResourcesMatch(NewStringResource("abc"), NewStringResource("abd")); err != nil || ok {
		t.Fatalf("different content: %v %v", ok, err)
	}
	// different lengths -> false, not error
	if ok, err := ResourcesMatch(NewStringResource("abc"), NewStringResource("abcd")); err != nil || ok {
		t.Fatalf("different lengths: %v %v", ok, err)
	}
	if ok, err := ResourcesMatch(NewStringResource(""), NewStringResource("abcd")); err != nil || ok {
		t.Fatalf("empty vs non-empty: %v %v", ok, err)
	}

	// force multi-chunk reads: >8192 bytes differing in the last byte
	big1 := make([]byte, 20000)
	big2 := make([]byte, 20000)
	for i := range big1 {
		big1[i] = 'a'
		big2[i] = 'a'
	}
	big2[19999] = 'b'
	if ok, err := ResourcesMatch(NewBytesResource(big1), NewBytesResource(big2)); err != nil || ok {
		t.Fatalf("last-byte difference missed: %v %v", ok, err)
	}
	if ok, err := ResourcesMatch(NewBytesResource(big1), NewBytesResource(big1)); err != nil || !ok {
		t.Fatalf("large equal resources: %v %v", ok, err)
	}

	// errors propagate
	if _, err := ResourcesMatch(&bbFailRes{err: errors.New("open boom")}, NewStringResource("x")); err == nil {
		t.Fatal("open error not propagated")
	}
	if _, err := ResourcesMatch(&bbFailMidRes{}, NewBytesResource(big1)); err == nil {
		t.Fatal("mid-read error not propagated")
	}
}

// TestDetail02: not-exist errors are returned UNWRAPPED so os.IsNotExist
// holds directly, for FileResource, VFSResource, and CopyResource.
func TestDetail02(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-file")

	fr := NewFileResource(missing)
	if _, err := fr.Open(); err == nil || !os.IsNotExist(err) {
		t.Fatalf("FileResource missing: %v (os.IsNotExist=%v)", err, os.IsNotExist(err))
	}

	p := vfs.NewMemFSPath(vfs.NewMemFSContext(), "missing")
	vr := NewVFSResource(p)
	if _, err := vr.Open(); err == nil || !os.IsNotExist(err) {
		t.Fatalf("VFSResource missing: %v (os.IsNotExist=%v)", err, os.IsNotExist(err))
	}

	if _, err := CopyResource(io.Discard, NewFileResource(missing)); err == nil || !os.IsNotExist(err) {
		t.Fatalf("CopyResource missing: %v (os.IsNotExist=%v)", err, os.IsNotExist(err))
	}

	// positive control: a real file opens and reads
	real := filepath.Join(t.TempDir(), "real")
	if err := os.WriteFile(real, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := NewFileResource(real).Open()
	if err != nil {
		t.Fatalf("FileResource existing: %v", err)
	}
	data, _ := io.ReadAll(r)
	if string(data) != "payload" {
		t.Fatalf("FileResource content = %q", data)
	}
}

// TestDetail03: BytesResource.MarshalJSON produces a JSON string (shape —
// the byte-to-string encoding is an implementation detail, only the JSON
// string form is asserted).
func TestDetail03(t *testing.T) {
	raw, err := json.Marshal(NewBytesResource([]byte("hello")))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("marshal produced invalid JSON: %q", raw)
	}
	if _, ok := v.(string); !ok {
		t.Fatalf("marshal produced %T (%s), want a JSON string", v, raw)
	}
}

// TestDetail04: TaskDependentResource.Open errors until Resource is set;
// IsReady is Resource != nil.
func TestDetail04(t *testing.T) {
	tr := &TaskDependentResource[CloudupSubContext]{}
	if tr.IsReady() {
		t.Fatal("nil Resource is ready")
	}
	if _, err := tr.Open(); err == nil {
		t.Fatal("Open with nil Resource did not error")
	}

	tr.Resource = NewStringResource("data")
	if !tr.IsReady() {
		t.Fatal("set Resource is not ready")
	}
	r, err := tr.Open()
	if err != nil {
		t.Fatalf("Open with Resource: %v", err)
	}
	data, _ := io.ReadAll(r)
	if string(data) != "data" {
		t.Fatalf("wrapped content = %q", data)
	}
}

// TestDetail05: TaskDependentResource.GetDependencies returns exactly the
// producing task.
func TestDetail05(t *testing.T) {
	task := &bbTask{}
	tr := &TaskDependentResource[CloudupSubContext]{Resource: NewStringResource("x"), Task: task}
	deps := tr.GetDependencies(map[string]Task[CloudupSubContext]{"t": task})
	if len(deps) != 1 || deps[0] != Task[CloudupSubContext](task) {
		t.Fatalf("GetDependencies = %v, want [task]", deps)
	}
}

// TestDetail06: FunctionToResource memoizes — fn runs once and its result is
// served on every Open. (The nil-result re-run edge is an implementation
// detail and is not asserted.)
func TestDetail06(t *testing.T) {
	calls := 0
	r := FunctionToResource(func() ([]byte, error) {
		calls++
		return []byte("generated"), nil
	})
	rd, err := r.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, _ := io.ReadAll(rd)
	if string(data) != "generated" {
		t.Fatalf("content = %q", data)
	}
	if _, err := r.Open(); err != nil {
		t.Fatalf("second Open: %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn ran %d times, want 1", calls)
	}
}

// TestDetail07: StringResource and BytesResource Open never error.
func TestDetail07(t *testing.T) {
	r1, err := NewStringResource("hello").Open()
	if err != nil {
		t.Fatalf("StringResource.Open: %v", err)
	}
	data, _ := io.ReadAll(r1)
	if string(data) != "hello" {
		t.Fatalf("StringResource content = %q", data)
	}
	r2, err := NewBytesResource([]byte("bytes")).Open()
	if err != nil {
		t.Fatalf("BytesResource.Open: %v", err)
	}
	data, _ = io.ReadAll(r2)
	if string(data) != "bytes" {
		t.Fatalf("BytesResource content = %q", data)
	}
}
