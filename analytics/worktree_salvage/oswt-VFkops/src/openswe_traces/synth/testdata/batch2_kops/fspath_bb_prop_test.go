package vfs_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"example.internal/kops/util/pkg/hashing"
	"example.internal/kops/util/pkg/vfs"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func rs(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// Detail 1: CreateFile on an existing path returns exactly os.ErrExist.
func TestDetail01_CreateExistingReturnsErrExist(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 50; i++ {
		dir := t.TempDir()
		p := vfs.NewFSPath(filepath.Join(dir, "f"+strconv.Itoa(i)))
		data := []byte(strconv.Itoa(rng.Intn(1000)))
		if err := p.WriteFile(ctx, rs(data), nil); err != nil {
			t.Fatalf("i=%d seed write: %v", i, err)
		}
		err := p.CreateFile(ctx, rs([]byte("other")), nil)
		if !errors.Is(err, os.ErrExist) || err != os.ErrExist {
			t.Fatalf("i=%d CreateFile err=%v want os.ErrExist", i, err)
		}
	}
}

// Detail 2: CreateFile propagates a stat error that is not not-exist
// WITHOUT writing.
func TestDetail02_CreatePropagatesNonNotExistStatError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	// A file where a directory component is expected: stat gives ENOTDIR,
	// which is not a not-exist error.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := vfs.NewFSPath(filepath.Join(blocker, "child"))
	err := p.CreateFile(ctx, rs([]byte("data")), nil)
	if err == nil {
		t.Fatalf("CreateFile through file-as-dir succeeded")
	}
	if errors.Is(err, os.ErrExist) {
		t.Fatalf("CreateFile returned ErrExist for ENOTDIR")
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CreateFile normalized ENOTDIR to ErrNotExist: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(blocker, "child")); serr == nil {
		t.Fatalf("file was written despite stat error")
	}
}

// Detail 3: CreateFile serializes on a process-wide mutex — concurrent
// creates on the same path yield exactly one success.
func TestDetail03_CreateFileSerializes(t *testing.T) {
	ctx := context.Background()
	for trial := 0; trial < 20; trial++ {
		dir := t.TempDir()
		p := vfs.NewFSPath(filepath.Join(dir, "contended"))
		const n = 8
		var wg sync.WaitGroup
		results := make([]error, n)
		for k := 0; k < n; k++ {
			wg.Add(1)
			go func(k int) {
				defer wg.Done()
				results[k] = p.CreateFile(ctx, rs([]byte{byte(k)}), nil)
			}(k)
		}
		wg.Wait()
		var ok, exist int
		for _, err := range results {
			if err == nil {
				ok++
			} else if errors.Is(err, os.ErrExist) {
				exist++
			}
		}
		if ok != 1 {
			t.Fatalf("trial=%d ok=%d exist=%d want exactly one success", trial, ok, exist)
		}
	}
}

// Detail 4: WriteFile creates missing parents (0755) and writes via a temp
// file in the destination dir + rename — no temp residue on success.
func TestDetail04_WriteTempRenameNoResidue(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		dir := t.TempDir()
		sub := filepath.Join(dir, "a", "b", "c")
		p := vfs.NewFSPath(filepath.Join(sub, "out.txt"))
		data := []byte("payload-" + strconv.Itoa(rng.Intn(10000)))
		if err := p.WriteFile(ctx, rs(data), nil); err != nil {
			t.Fatalf("i=%d WriteFile: %v", i, err)
		}
		got, err := os.ReadFile(filepath.Join(sub, "out.txt"))
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("i=%d content %q err %v", i, got, err)
		}
		entries, err := os.ReadDir(sub)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "out.txt" {
			names := []string{}
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("i=%d residue files %v", i, names)
		}
	}
}

// Detail 5: ReadFile normalizes ENOENT to os.ErrNotExist.
func TestDetail05_ReadFileNormalizesENOENT(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vfs.NewFSPath(filepath.Join(dir, "missing"))
	_, err := p.ReadFile(ctx)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadFile missing err=%v want ErrNotExist", err)
	}
	// Nested missing path too.
	p2 := vfs.NewFSPath(filepath.Join(dir, "a", "b", "missing"))
	_, err = p2.ReadFile(ctx)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadFile nested-missing err=%v want ErrNotExist", err)
	}
}

// Detail 6: Remove does NOT normalize — missing path returns the raw
// not-exist error.
func TestDetail06_RemoveRawError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := vfs.NewFSPath(filepath.Join(dir, "missing"))
	err := p.Remove(ctx)
	if err == nil {
		t.Fatalf("Remove on missing path succeeded")
	}
	// Raw os.Remove error is a *PathError wrapping ENOENT — still errors.Is
	// ErrNotExist, but it is NOT a bare sentinel.
	if pe, ok := err.(*os.PathError); !ok || pe.Op != "remove" {
		t.Fatalf("Remove err=%T %v want *os.PathError from os.Remove", err, err)
	}
}

// Detail 7: ReadDir is one level and INCLUDES directories.
func TestDetail07_ReadDirOneLevelIncludesDirs(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		dir := t.TempDir()
		nf := 1 + rng.Intn(4)
		nd := 1 + rng.Intn(3)
		want := map[string]bool{}
		for j := 0; j < nf; j++ {
			name := "f" + strconv.Itoa(j)
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			want[name] = true
		}
		for j := 0; j < nd; j++ {
			name := "d" + strconv.Itoa(j)
			if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name, "inner"), []byte("y"), 0o644); err != nil {
				t.Fatal(err)
			}
			want[name] = true
		}
		entries, err := vfs.NewFSPath(dir).ReadDir()
		if err != nil {
			t.Fatalf("i=%d ReadDir: %v", i, err)
		}
		if len(entries) != nf+nd {
			names := []string{}
			for _, e := range entries {
				names = append(names, e.Base())
			}
			t.Fatalf("i=%d ReadDir=%v want %d entries", i, names, nf+nd)
		}
		for _, e := range entries {
			if !want[e.Base()] {
				t.Fatalf("i=%d unexpected entry %q", i, e.Base())
			}
		}
	}
}

// Detail 8: ReadTree is recursive and EXCLUDES directories — files only,
// full joined paths.
func TestDetail08_ReadTreeFilesOnlyRecursive(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 40; i++ {
		dir := t.TempDir()
		deep := filepath.Join(dir, "x", "y", "z")
		if err := os.MkdirAll(deep, 0o755); err != nil {
			t.Fatal(err)
		}
		files := []string{
			filepath.Join(dir, "top"),
			filepath.Join(deep, "deep"),
			filepath.Join(dir, "x", "mid" + strconv.Itoa(i)),
		}
		for _, f := range files {
			if err := os.WriteFile(f, []byte("d"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		entries, err := vfs.NewFSPath(dir).ReadTree(ctx)
		if err != nil {
			t.Fatalf("i=%d ReadTree: %v", i, err)
		}
		if len(entries) != len(files) {
			t.Fatalf("i=%d ReadTree returned %d entries want %d", i, len(entries), len(files))
		}
		got := map[string]bool{}
		for _, e := range entries {
			got[e.Path()] = true
			// Directories must never appear.
			fi, err := os.Stat(e.Path())
			if err != nil || fi.IsDir() {
				t.Fatalf("i=%d ReadTree entry %q is dir or missing", i, e.Path())
			}
		}
		for _, f := range files {
			if !got[f] {
				t.Fatalf("i=%d ReadTree missing %q (got %v)", i, f, got)
			}
		}
	}
}

// Detail 9: RemoveAll deletes FILES only and leaves the empty directory
// skeleton.
func TestDetail09_RemoveAllLeavesDirSkeleton(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	deep := filepath.Join(dir, "x", "y")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{filepath.Join(dir, "a"), filepath.Join(deep, "b")} {
		if err := os.WriteFile(f, []byte("d"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := vfs.NewFSPath(dir).RemoveAll(ctx); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	for _, d := range []string{dir, filepath.Join(dir, "x"), deep} {
		fi, err := os.Stat(d)
		if err != nil || !fi.IsDir() {
			t.Fatalf("dir skeleton %q lost: %v", d, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "x" {
		t.Fatalf("unexpected residue in %q", dir)
	}
	if e, _ := os.ReadDir(deep); len(e) != 0 {
		t.Fatalf("file residue under %q", deep)
	}
}

// Detail 10: RemoveAllVersions removes only the single path.
func TestDetail10_RemoveAllVersionsSinglePath(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim")
	keeper := filepath.Join(dir, "keeper")
	for _, f := range []string{victim, keeper} {
		if err := os.WriteFile(f, []byte("d"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := vfs.NewFSPath(victim).RemoveAllVersions(ctx); err != nil {
		t.Fatalf("RemoveAllVersions: %v", err)
	}
	if _, err := os.Stat(victim); !os.IsNotExist(err) {
		t.Fatalf("victim remains: %v", err)
	}
	if _, err := os.Stat(keeper); err != nil {
		t.Fatalf("sibling clobbered: %v", err)
	}
}

// Detail 11: PreferredHash is SHA256.
func TestDetail11_PreferredHashSHA256(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	dir := t.TempDir()
	p := vfs.NewFSPath(filepath.Join(dir, "x"))
	if err := p.WriteFile(ctx, rs([]byte("abc")), nil); err != nil {
		t.Fatal(err)
	}
	h, err := p.PreferredHash()
	if err != nil {
		t.Fatal(err)
	}
	if h.Algorithm != hashing.HashAlgorithmSHA256 {
		t.Fatalf("PreferredHash algorithm=%q want sha256", h.Algorithm)
	}
}

// Detail 12: WriteTo streams contents and reports byte count.
func TestDetail12_WriteToStreamsAndCounts(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 50; i++ {
		dir := t.TempDir()
		data := make([]byte, rng.Intn(4096))
		rng.Read(data)
		p := vfs.NewFSPath(filepath.Join(dir, "f"))
		if err := p.WriteFile(ctx, rs(data), nil); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		n, err := p.WriteTo(&out)
		if err != nil {
			t.Fatalf("i=%d WriteTo: %v", i, err)
		}
		if n != int64(len(data)) || !bytes.Equal(out.Bytes(), data) {
			t.Fatalf("i=%d WriteTo n=%d len(out)=%d len(data)=%d", i, n, out.Len(), len(data))
		}
	}
	// Error propagation on missing file.
	var out bytes.Buffer
	p := vfs.NewFSPath(filepath.Join(t.TempDir(), "missing"))
	if _, err := p.WriteTo(&out); err == nil {
		t.Fatalf("WriteTo on missing file succeeded")
	}
}

// Detail 13: WriteFile replaces an existing file (no exclusive-create check).
func TestDetail13_WriteReplacesExisting(t *testing.T) {
	ctx := context.Background()
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		dir := t.TempDir()
		p := vfs.NewFSPath(filepath.Join(dir, "f"))
		if err := p.WriteFile(ctx, rs([]byte("first")), nil); err != nil {
			t.Fatal(err)
		}
		second := []byte("second-" + strconv.Itoa(rng.Intn(100)))
		if err := p.WriteFile(ctx, rs(second), nil); err != nil {
			t.Fatalf("i=%d overwrite WriteFile: %v", i, err)
		}
		got, err := os.ReadFile(p.Path())
		if err != nil || !bytes.Equal(got, second) {
			t.Fatalf("i=%d content %q err %v", i, got, err)
		}
	}
}
