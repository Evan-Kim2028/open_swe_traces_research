package filesystem

import (
	"bytes"
	_ "crypto/sha1"
	_ "crypto/sha256"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-billy/v6/util"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/format/gitignore"
	"example.internal/gitkit/v6/plumbing/format/index"
	"example.internal/gitkit/v6/utils/merkletrie/noder"
)

func blobHash(t *testing.T, data []byte) plumbing.Hash {
	t.Helper()
	h, err := plumbing.FromObjectFormat(config.SHA1).Compute(plumbing.BlobObject, data)
	if err != nil {
		t.Fatalf("compute blob hash: %v", err)
	}
	return h
}

func findChild(t *testing.T, parent noder.Noder, name string) noder.Noder {
	t.Helper()
	ch, err := parent.Children()
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	for _, c := range ch {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func childNames(t *testing.T, parent noder.Noder) []string {
	t.Helper()
	ch, err := parent.Children()
	if err != nil {
		t.Fatalf("Children: %v", err)
	}
	names := make([]string, 0, len(ch))
	for _, c := range ch {
		names = append(names, c.Name())
	}
	return names
}

func hasName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// entryInjectFS wraps a billy filesystem so tests can present directory
// entries that a memory filesystem cannot create (sockets) or emulate a
// directory that disappears between the parent's listing and the child's
// own ReadDir.
type entryInjectFS struct {
	billy.Filesystem
	injectAt string
	inject   []iofs.DirEntry
	missing  map[string]bool
}

func cleanPath(p string) string {
	return path.Clean("/" + strings.TrimPrefix(p, "/"))
}

func (f *entryInjectFS) ReadDir(p string) ([]iofs.DirEntry, error) {
	if f.missing[cleanPath(p)] {
		return nil, &os.PathError{Op: "open", Path: p, Err: os.ErrNotExist}
	}
	l, err := f.Filesystem.ReadDir(p)
	if err != nil {
		return l, err
	}
	if cleanPath(p) == f.injectAt {
		l = append(l, f.inject...)
	}
	return l, nil
}

func (f *entryInjectFS) Stat(p string) (iofs.FileInfo, error) {
	if f.missing[cleanPath(p)] {
		return nil, &os.PathError{Op: "stat", Path: p, Err: os.ErrNotExist}
	}
	return f.Filesystem.Stat(p)
}

func (f *entryInjectFS) Lstat(p string) (iofs.FileInfo, error) {
	if f.missing[cleanPath(p)] {
		return nil, &os.PathError{Op: "lstat", Path: p, Err: os.ErrNotExist}
	}
	return f.Filesystem.Lstat(p)
}

type fakeEntry struct {
	name string
	mode os.FileMode
}

func (e fakeEntry) Name() string               { return e.name }
func (e fakeEntry) IsDir() bool                { return e.mode.IsDir() }
func (e fakeEntry) Type() iofs.FileMode        { return iofs.FileMode(e.mode).Type() }
func (e fakeEntry) Info() (iofs.FileInfo, error) {
	return fakeInfo{e}, nil
}

type fakeInfo struct{ e fakeEntry }

func (i fakeInfo) Name() string       { return i.e.name }
func (i fakeInfo) Size() int64        { return 0 }
func (i fakeInfo) Mode() os.FileMode  { return i.e.mode }
func (i fakeInfo) ModTime() time.Time { return time.Now() }
func (i fakeInfo) IsDir() bool        { return i.e.mode.IsDir() }
func (i fakeInfo) Sys() any           { return nil }

// statFields fills the OS-derived metadata fields of an index entry from
// the file's real stat, so only the documented size/mode/mtime fields can
// be exercised as mismatches in tests.
func statFields(t *testing.T, e *index.Entry, fi os.FileInfo) {
	t.Helper()
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		e.Dev = uint32(st.Dev)
		e.Inode = uint32(st.Ino)
		e.UID = st.Uid
		e.GID = st.Gid
	}
	e.Size = uint32(fi.Size())
	e.ModifiedAt = fi.ModTime()
	e.Mode, _ = filemode.NewFromOSFileMode(fi.Mode())
}

var sentinelHash = plumbing.NewHash("ffffffffffffffffffffffffffffffffffffffff")

// 1. File node hash is blob-hash bytes followed by file-mode bytes —
//    content and mode both participate. Inferable: doc.
func TestDetail01(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)
	if err := util.WriteFile(fs, "a.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewRootNode(fs, nil)
	f := findChild(t, root, "a.txt")
	if f == nil {
		t.Fatal("a.txt not in root children")
	}
	h := f.Hash()
	if len(h) != 24 {
		t.Fatalf("hash len %d, want 24", len(h))
	}
	if !bytes.Equal(h[:20], blobHash(t, []byte("hello")).Bytes()) {
		t.Fatalf("hash prefix is not the blob hash of the content")
	}

	// Mode participates: chmod the same content and the hash changes.
	if err := os.Chmod(filepath.Join(dir, "a.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	f2 := findChild(t, NewRootNode(fs, nil), "a.txt")
	h2 := f2.Hash()
	if bytes.Equal(h, h2) {
		t.Fatal("hash unchanged after mode change")
	}
	if !bytes.Equal(h2[:20], blobHash(t, []byte("hello")).Bytes()) {
		t.Fatal("blob-hash prefix changed when only the mode changed")
	}

	// Content participates.
	if err := util.WriteFile(fs, "a.txt", []byte("goodbye"), 0o755); err != nil {
		t.Fatal(err)
	}
	f3 := findChild(t, NewRootNode(fs, nil), "a.txt")
	if bytes.Equal(h2, f3.Hash()) {
		t.Fatal("hash unchanged after content change")
	}
}

// 2. A directory's hash is always 24 zero bytes. Inferable: doc.
func TestDetail02(t *testing.T) {
	fs := memfs.New()
	if err := fs.MkdirAll("sub/deep", 0o755); err != nil {
		t.Fatal(err)
	}
	root := NewRootNode(fs, nil)
	if !bytes.Equal(root.Hash(), make([]byte, 24)) {
		t.Fatal("root dir hash is not 24 zero bytes")
	}
	sub := findChild(t, root, "sub")
	if sub == nil || !sub.IsDir() {
		t.Fatal("sub not found or not a dir")
	}
	if !bytes.Equal(sub.Hash(), make([]byte, 24)) {
		t.Fatal("dir hash is not 24 zero bytes")
	}
}

// 3. When size, mode and mtime all match the index entry and the entry is
//    not racy, the stored index hash is reused rather than re-reading the
//    file. Inferable: doc.
func TestDetail03(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)
	if err := util.WriteFile(fs, "a.txt", []byte("real content"), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "a.txt"), mtime, mtime); err != nil {
		t.Fatal(err)
	}
	fi, err := fs.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	idx := &index.Index{ModTime: mtime.Add(time.Hour)}
	e, err := idx.Add("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	statFields(t, e, fi)
	e.Hash = sentinelHash

	f := findChild(t, NewRootNodeWithOptions(fs, nil, Options{Index: idx}), "a.txt")
	if f == nil {
		t.Fatal("a.txt missing")
	}
	if !bytes.Equal(f.Hash()[:20], sentinelHash.Bytes()) {
		t.Fatalf("metadata match did not reuse stored index hash; got %x", f.Hash()[:20])
	}
}

// 4. Racy guard: a file whose mtime is equal to or newer than the index
//    ModTime is content-hashed even when the other metadata matches.
//    Inferable: partially — assert the named condition's observable.
func TestDetail04(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)
	if err := util.WriteFile(fs, "a.txt", []byte("real content"), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "a.txt"), mtime, mtime); err != nil {
		t.Fatal(err)
	}
	fi, err := fs.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	real := blobHash(t, []byte("real content"))

	for name, idxMod := range map[string]time.Time{
		"equal": mtime,
		"newer": mtime.Add(-time.Hour),
	} {
		idx := &index.Index{ModTime: idxMod}
		e, err := idx.Add("a.txt")
		if err != nil {
			t.Fatal(err)
		}
		statFields(t, e, fi)
		e.Hash = sentinelHash

		f := findChild(t, NewRootNodeWithOptions(fs, nil, Options{Index: idx}), "a.txt")
		if bytes.Equal(f.Hash()[:20], sentinelHash.Bytes()) {
			t.Fatalf("%s: racy file reused stored index hash", name)
		}
		if !bytes.Equal(f.Hash()[:20], real.Bytes()) {
			t.Fatalf("%s: racy file hash is not the content hash", name)
		}
	}
}

// 5. With no index ModTime, metadata alone is never trusted — hashing
//    always falls back to content. Inferable: no — assert SHAPE: the
//    stored index hash is not adopted; the content hash is produced.
func TestDetail05(t *testing.T) {
	dir := t.TempDir()
	fs := osfs.New(dir)
	if err := util.WriteFile(fs, "a.txt", []byte("real content"), 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := fs.Stat("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	idx := &index.Index{} // zero ModTime: in-memory index
	e, err := idx.Add("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	statFields(t, e, fi)
	e.Hash = sentinelHash

	f := findChild(t, NewRootNodeWithOptions(fs, nil, Options{Index: idx}), "a.txt")
	if bytes.Equal(f.Hash()[:20], sentinelHash.Bytes()) {
		t.Fatal("stored index hash trusted without an index ModTime")
	}
	if !bytes.Equal(f.Hash()[:20], blobHash(t, []byte("real content")).Bytes()) {
		t.Fatal("hash is not the content hash")
	}
}

// 6. With AutoCRLF a text file hashes its LF-normalized content; a binary
//    file is hashed raw. Inferable: partially — assert the observable
//    equivalence against computed blob hashes.
func TestDetail06(t *testing.T) {
	fs := memfs.New()
	text := []byte("line one\r\nline two\r\n")
	bin := []byte{'x', 0x00, 'y', '\r', '\n', 'z'}
	if err := util.WriteFile(fs, "text.txt", text, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := util.WriteFile(fs, "bin.dat", bin, 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRootNodeWithOptions(fs, nil, Options{AutoCRLF: true})
	textNode := findChild(t, root, "text.txt")
	binNode := findChild(t, root, "bin.dat")

	norm := blobHash(t, []byte("line one\nline two\n"))
	raw := blobHash(t, text)
	if !bytes.Equal(textNode.Hash()[:20], norm.Bytes()) {
		t.Fatalf("text file not hashed over LF-normalized content: %x", textNode.Hash()[:20])
	}
	if bytes.Equal(textNode.Hash()[:20], raw.Bytes()) {
		t.Fatal("text file hashed over raw CRLF content")
	}
	if !bytes.Equal(binNode.Hash()[:20], blobHash(t, bin).Bytes()) {
		t.Fatal("binary file content not hashed raw")
	}
}

// 7. A symlink hashes its target path bytes — the link is not followed.
//    Inferable: partially.
func TestDetail07(t *testing.T) {
	fs := memfs.New()
	if err := util.WriteFile(fs, "dir/target.txt", []byte("different content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fs.Symlink("dir/target.txt", "link"); err != nil {
		t.Fatal(err)
	}
	ln := findChild(t, NewRootNode(fs, nil), "link")
	if ln == nil {
		t.Fatal("link missing")
	}
	if !bytes.Equal(ln.Hash()[:20], blobHash(t, []byte("dir/target.txt")).Bytes()) {
		t.Fatalf("symlink not hashed over its target path: %x", ln.Hash()[:20])
	}
}

// 8. A submodule path reports the recorded commit hash and is not
//    descended. Inferable: partially — assert the recorded-hash
//    observable; mode encoding is not pinned.
func TestDetail08(t *testing.T) {
	fs := memfs.New()
	if err := util.WriteFile(fs, "vendor/lib/inner.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := plumbing.NewHash("1111111111111111111111111111111111111111")
	sub2 := plumbing.NewHash("2222222222222222222222222222222222222222")

	n := findChild(t, NewRootNode(fs, map[string]plumbing.Hash{"vendor/lib": sub}), "vendor")
	if n == nil {
		t.Fatal("vendor missing")
	}
	lib := findChild(t, n, "lib")
	if lib == nil {
		t.Fatal("lib missing")
	}
	if !bytes.Equal(lib.Hash()[:20], sub.Bytes()) {
		t.Fatalf("submodule hash %x is not the recorded commit", lib.Hash()[:20])
	}

	lib2 := findChild(t, findChild(t, NewRootNode(fs, map[string]plumbing.Hash{"vendor/lib": sub2}), "vendor"), "lib")
	if bytes.Equal(lib.Hash(), lib2.Hash()) {
		t.Fatal("different recorded submodule hashes produced identical node hashes")
	}
}

// 9. `.git` is excluded, sockets are skipped, and a directory that
//    disappears between listing and walk yields no children and no
//    error. Inferable: partially.
func TestDetail09(t *testing.T) {
	fs := memfs.New()
	if err := util.WriteFile(fs, ".git/HEAD", []byte("ref"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := util.WriteFile(fs, "plain.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	names := childNames(t, NewRootNode(fs, nil))
	if hasName(names, ".git") {
		t.Fatal(".git was listed as a child")
	}
	if !hasName(names, "plain.txt") {
		t.Fatal("plain.txt missing")
	}

	wrap := &entryInjectFS{
		Filesystem: memfs.New(),
		injectAt:   "/",
		inject: []iofs.DirEntry{
			fakeEntry{name: "sock", mode: os.ModeSocket | 0o777},
			fakeEntry{name: "gone", mode: os.ModeDir | 0o755},
		},
		missing: map[string]bool{"/gone": true},
	}
	root := NewRootNode(wrap, nil)
	names = childNames(t, root)
	if hasName(names, "sock") {
		t.Fatal("socket entry was not skipped")
	}
	gone := findChild(t, root, "gone")
	if gone == nil {
		t.Fatal("injected dir entry missing from root children")
	}
	ch, err := gone.Children()
	if err != nil {
		t.Fatalf("disappeared dir produced an error: %v", err)
	}
	if len(ch) != 0 {
		t.Fatalf("disappeared dir produced %d children", len(ch))
	}
}

// 10. Ignored entries are skipped only when the scope matches AND the
//     path (or, for dirs, any descendant) is absent from the index —
//     tracked entries still walk. Inferable: doc.
func TestDetail10(t *testing.T) {
	fs := memfs.New()
	must := func(p string) {
		t.Helper()
		if err := util.WriteFile(fs, p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("ignored.txt")
	must("tracked.txt")
	must("excl/inner.txt")
	must("excl2/keep.txt")
	must("plain.txt")

	idx := &index.Index{}
	for _, p := range []string{"tracked.txt", "excl2/keep.txt"} {
		if _, err := idx.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	scope := gitignore.NewScope([]gitignore.Pattern{
		gitignore.ParsePattern("ignored.txt", nil),
		gitignore.ParsePattern("tracked.txt", nil),
		gitignore.ParsePattern("excl", nil),
		gitignore.ParsePattern("excl2", nil),
	})
	root := NewRootNodeWithOptions(fs, nil, Options{Index: idx, IgnoreScope: scope})
	names := childNames(t, root)
	if hasName(names, "ignored.txt") {
		t.Fatal("ignored untracked file was not skipped")
	}
	if !hasName(names, "tracked.txt") {
		t.Fatal("tracked file matching an ignore rule was skipped")
	}
	if hasName(names, "excl") {
		t.Fatal("fully-untracked ignored dir was not skipped")
	}
	excl2 := findChild(t, root, "excl2")
	if excl2 == nil {
		t.Fatal("ignored dir containing a tracked file was skipped")
	}
	if !hasName(childNames(t, excl2), "keep.txt") {
		t.Fatal("tracked file inside ignored dir missing")
	}
}

// 11. Each directory's scope derives lazily from its own listing — a
//     .gitignore in a visited dir applies to its children, and an
//     excluded dir's .gitignore is never read. Inferable: partially —
//     assert the observable skip/no-resurrect behaviour.
func TestDetail11(t *testing.T) {
	fs := memfs.New()
	must := func(p, body string) {
		t.Helper()
		if err := util.WriteFile(fs, p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("sub/.gitignore", "inner.txt\n")
	must("sub/inner.txt", "x")
	must("sub/keep.txt", "x")
	must("excl/.gitignore", "!keep.txt\n")
	must("excl/keep.txt", "x")

	idx := &index.Index{}
	scope := gitignore.NewScope([]gitignore.Pattern{
		gitignore.ParsePattern("excl", nil),
	})
	root := NewRootNodeWithOptions(fs, nil, Options{Index: idx, IgnoreScope: scope})
	names := childNames(t, root)
	if hasName(names, "excl") {
		t.Fatal("excluded dir walked — a negation in its .gitignore resurrected it")
	}
	sub := findChild(t, root, "sub")
	if sub == nil {
		t.Fatal("sub missing")
	}
	subNames := childNames(t, sub)
	if hasName(subNames, "inner.txt") {
		t.Fatal("nested .gitignore pattern not applied to the dir's children")
	}
	if !hasName(subNames, "keep.txt") {
		t.Fatal("non-matching file in nested-gitignore dir was skipped")
	}
}

// 12. Hash is computed lazily on first Hash() call and cached — later
//     filesystem changes are invisible to that node. Inferable: doc.
func TestDetail12(t *testing.T) {
	fs := memfs.New()
	if err := util.WriteFile(fs, "a.txt", []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := NewRootNode(fs, nil)
	f := findChild(t, root, "a.txt")
	h1 := f.Hash()
	if err := util.WriteFile(fs, "a.txt", []byte("v2 different"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Hash(), h1) {
		t.Fatal("hash recomputed after first call")
	}
	f2 := findChild(t, NewRootNode(fs, nil), "a.txt")
	if bytes.Equal(f2.Hash(), h1) {
		t.Fatal("fresh node did not see updated content")
	}
}
