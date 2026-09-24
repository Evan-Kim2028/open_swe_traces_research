package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
	"example.internal/gitkit/v6/plumbing/object"
	"example.internal/gitkit/v6/storage/memory"
)

func arcObj(t *testing.T, st *memory.Storage, typ plumbing.ObjectType, enc func(o plumbing.EncodedObject) error) plumbing.Hash {
	t.Helper()
	o := st.NewEncodedObject()
	o.SetType(typ)
	if err := enc(o); err != nil {
		t.Fatalf("encode %s: %v", typ, err)
	}
	h, err := st.SetEncodedObject(o)
	if err != nil {
		t.Fatalf("store %s: %v", typ, err)
	}
	return h
}

func arcBlob(t *testing.T, st *memory.Storage, body string) plumbing.Hash {
	return arcObj(t, st, plumbing.BlobObject, func(o plumbing.EncodedObject) error {
		w, err := o.Writer()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(body)); err != nil {
			return err
		}
		return w.Close()
	})
}

func arcTree(t *testing.T, st *memory.Storage, ents ...object.TreeEntry) plumbing.Hash {
	// git tree ordering: directories sort as "name/"
	key := func(e object.TreeEntry) string {
		if e.Mode == filemode.Dir {
			return e.Name + "/"
		}
		return e.Name
	}
	slices.SortFunc(ents, func(a, b object.TreeEntry) int {
		return strings.Compare(key(a), key(b))
	})
	return arcObj(t, st, plumbing.TreeObject, func(o plumbing.EncodedObject) error {
		return (&object.Tree{Entries: ents}).Encode(o)
	})
}

func arcCommit(t *testing.T, st *memory.Storage, treeH plumbing.Hash, when time.Time) plumbing.Hash {
	return arcObj(t, st, plumbing.CommitObject, func(o plumbing.EncodedObject) error {
		return (&object.Commit{
			Author:    object.Signature{Name: "a", Email: "a@b", When: when},
			Committer: object.Signature{Name: "a", Email: "a@b", When: when},
			Message:   "msg\n",
			TreeHash:  treeH,
		}).Encode(o)
	})
}

func arcTag(t *testing.T, st *memory.Storage, name string, targetType plumbing.ObjectType, target plumbing.Hash, when time.Time) plumbing.Hash {
	return arcObj(t, st, plumbing.TagObject, func(o plumbing.EncodedObject) error {
		return (&object.Tag{
			Name:       name,
			Tagger:     object.Signature{Name: "a", Email: "a@b", When: when},
			Message:    "tag\n",
			TargetType: targetType,
			Target:     target,
		}).Encode(o)
	})
}

// simple repo: README.md file, docs/ dir with guide.md, plus a commit and
// branch ref "main".
type arcRepo struct {
	st      *memory.Storage
	treeH   plumbing.Hash
	commitH plumbing.Hash
	when    time.Time
}

func arcFixture(t *testing.T) *arcRepo {
	st := memory.NewStorage()
	readme := arcBlob(t, st, "readme body\n")
	guide := arcBlob(t, st, "guide body\n")
	sub := arcTree(t, st, object.TreeEntry{Name: "guide.md", Mode: filemode.Regular, Hash: guide})
	when := time.Date(2020, 3, 4, 5, 6, 7, 0, time.UTC)
	root := arcTree(t, st,
		object.TreeEntry{Name: "README.md", Mode: filemode.Regular, Hash: readme},
		object.TreeEntry{Name: "docs", Mode: filemode.Dir, Hash: sub},
	)
	commit := arcCommit(t, st, root, when)
	if err := st.SetReference(plumbing.NewHashReference("refs/heads/main", commit)); err != nil {
		t.Fatalf("set ref: %v", err)
	}
	return &arcRepo{st: st, treeH: root, commitH: commit, when: when}
}

// 1. Without allowUnreachable, raw hashes are rejected with ErrOnlyRefNames
//    and relative expressions with ErrRelativeExpressions — only ref names
//    and ref:path are allowed.
func TestDetail01(t *testing.T) {
	r := arcFixture(t)

	if _, _, _, err := ResolveTreeish(r.st, r.commitH.String(), false); !errors.Is(err, ErrOnlyRefNames) {
		t.Fatalf("raw hash: got %v, want ErrOnlyRefNames", err)
	}
	for _, expr := range []string{"main^", "main~2", "main@{0}", "main^0"} {
		if _, _, _, err := ResolveTreeish(r.st, expr, false); !errors.Is(err, ErrRelativeExpressions) {
			t.Fatalf("%q: got %v, want ErrRelativeExpressions", expr, err)
		}
	}
	// same inputs allowed when unreachable is permitted
	if _, _, _, err := ResolveTreeish(r.st, r.commitH.String(), true); err != nil {
		t.Fatalf("raw hash with allowUnreachable: %v", err)
	}
}

// 2. `ref:path` strips at the FIRST colon — a path may itself contain
//    colons; the sub-path must resolve to a directory entry, not a file.
func TestDetail02(t *testing.T) {
	r := arcFixture(t)

	tree, _, _, err := ResolveTreeish(r.st, "main:docs", false)
	if err != nil {
		t.Fatalf("ref:path to dir: %v", err)
	}
	if _, err := tree.File("guide.md"); err != nil {
		t.Fatalf("sub-tree missing guide.md: %v", err)
	}

	// resolving to a file (not a dir) is an error naming the path
	if _, _, _, err := ResolveTreeish(r.st, "main:README.md", false); err == nil ||
		!strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("ref:file: got %v, want a not-a-directory error", err)
	}

	// path containing a colon: split at the FIRST colon means "a:b" is the path
	st2 := memory.NewStorage()
	inner := arcTree(t, st2)
	colonDir := arcTree(t, st2, object.TreeEntry{Name: "a:b", Mode: filemode.Dir, Hash: inner})
	c2 := arcCommit(t, st2, colonDir, r.when)
	if err := st2.SetReference(plumbing.NewHashReference("refs/heads/x", c2)); err != nil {
		t.Fatalf("set ref: %v", err)
	}
	if _, _, _, err := ResolveTreeish(st2, "x:a:b", false); err != nil {
		t.Fatalf("colon in path: %v", err)
	}
}

// 3. Ref resolution tries the bare name, then refs/heads/, then refs/tags/
//    in that order — a branch name beats a tag of the same name.
func TestDetail03(t *testing.T) {
	r := arcFixture(t)

	// tag with the same short name as the branch
	tagH := arcTag(t, r.st, "main", plumbing.CommitObject, r.commitH, r.when)
	if err := r.st.SetReference(plumbing.NewHashReference("refs/tags/main", tagH)); err != nil {
		t.Fatalf("set tag ref: %v", err)
	}

	h, err := ResolveRef(r.st, "main", false)
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if h != r.commitH {
		t.Fatalf("branch should beat tag: got %s, want %s", h, r.commitH)
	}

	full, err := ResolveRef(r.st, "refs/tags/main", false)
	if err != nil {
		t.Fatalf("full ref: %v", err)
	}
	if full != tagH {
		t.Fatalf("full ref: got %s, want tag %s", full, tagH)
	}
}

// 4. Annotated tags unwrap to their target repeatedly until a non-tag
//    object is reached — chains of tags work.
func TestDetail04(t *testing.T) {
	r := arcFixture(t)

	inner := arcTag(t, r.st, "inner", plumbing.CommitObject, r.commitH, r.when)
	outer := arcTag(t, r.st, "outer", plumbing.TagObject, inner, r.when)
	if err := r.st.SetReference(plumbing.NewHashReference("refs/tags/outer", outer)); err != nil {
		t.Fatalf("set ref: %v", err)
	}

	tree, commitH, _, err := ResolveTreeish(r.st, "outer", false)
	if err != nil {
		t.Fatalf("tag chain: %v", err)
	}
	if commitH == nil || *commitH != r.commitH {
		t.Fatalf("tag chain commit: got %v, want %s", commitH, r.commitH)
	}
	if tree.Hash != r.treeH {
		t.Fatalf("tag chain tree: got %s, want %s", tree.Hash, r.treeH)
	}
}

// 5. Tree-ish resolution timestamps differ by input kind: commits and tags
//    use the committer time, but a bare tree uses the current time.
func TestDetail05(t *testing.T) {
	r := arcFixture(t)

	_, _, ctime, err := ResolveTreeish(r.st, "main", false)
	if err != nil {
		t.Fatalf("commit resolve: %v", err)
	}
	if !ctime.Equal(r.when) {
		t.Fatalf("commit time %v, want committer time %v", ctime, r.when)
	}

	tagH := arcTag(t, r.st, "v1", plumbing.CommitObject, r.commitH, r.when)
	if err := r.st.SetReference(plumbing.NewHashReference("refs/tags/v1", tagH)); err != nil {
		t.Fatalf("set ref: %v", err)
	}
	_, _, ttime, err := ResolveTreeish(r.st, "v1", false)
	if err != nil {
		t.Fatalf("tag resolve: %v", err)
	}
	if !ttime.Equal(r.when) {
		t.Fatalf("tag time %v, want committer time %v", ttime, r.when)
	}

	// bare tree via raw hash: current time, not a stored timestamp
	before := time.Now()
	_, _, dtime, err := ResolveTreeish(r.st, r.treeH.String(), true)
	after := time.Now()
	if err != nil {
		t.Fatalf("bare tree resolve: %v", err)
	}
	if dtime.Before(before) || dtime.After(after) {
		t.Fatalf("bare tree time %v not between %v and %v", dtime, before, after)
	}
}

// 6. Tar output carries a PAX global header named `pax_global_header` whose
//    comment is the commit hash — `git get-tar-commit-id` reads it back.
func TestDetail06(t *testing.T) {
	r := arcFixture(t)
	tree, err := object.GetTree(r.st, r.treeH)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}

	var buf bytes.Buffer
	if err := WriteTarArchive(r.st, &buf, tree, &r.commitH, "", nil, r.when); err != nil {
		t.Fatalf("WriteTarArchive: %v", err)
	}

	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("first tar record: %v", err)
	}
	if hdr.Name != PAXGlobalHeader {
		t.Fatalf("first record %q, want %q", hdr.Name, PAXGlobalHeader)
	}

	id, err := GetTarCommitID(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("GetTarCommitID: %v", err)
	}
	if *id != r.commitH {
		t.Fatalf("commit id %s, want %s", id, r.commitH)
	}
}

// 7. Directories in tar get mode|0777 minus umask; regular files get
//    mode|0666 (or |0777 if executable) minus umask; symlinks are always
//    0777 and carry the link target as Linkname with size 0.
func TestDetail07(t *testing.T) {
	st := memory.NewStorage()
	reg := arcBlob(t, st, "regular\n")
	exe := arcBlob(t, st, "#!/bin/sh\n")
	link := arcBlob(t, st, "target.txt")
	sub := arcTree(t, st, object.TreeEntry{Name: "inner.txt", Mode: filemode.Regular, Hash: reg})
	root := arcTree(t, st,
		object.TreeEntry{Name: "dir", Mode: filemode.Dir, Hash: sub},
		object.TreeEntry{Name: "run.sh", Mode: filemode.Executable, Hash: exe},
		object.TreeEntry{Name: "sym", Mode: filemode.Symlink, Hash: link},
		object.TreeEntry{Name: "plain.txt", Mode: filemode.Regular, Hash: reg},
	)
	tree, err := object.GetTree(st, root)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}

	var buf bytes.Buffer
	if err := WriteTarArchive(st, &buf, tree, nil, "", nil, time.Now()); err != nil {
		t.Fatalf("WriteTarArchive: %v", err)
	}

	wantFile := 0o666 &^ DefaultUmask
	wantExec := 0o777 &^ DefaultUmask
	wantDir := 0o777 &^ DefaultUmask
	perm := func(h *tar.Header) int64 { return h.Mode & 0o777 }

	seen := map[string]*tar.Header{}
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		seen[hdr.Name] = hdr
	}

	if h := seen["plain.txt"]; h == nil || perm(h) != int64(wantFile) {
		t.Fatalf("regular file mode %v, want %o", seen["plain.txt"], wantFile)
	}
	if h := seen["run.sh"]; h == nil || perm(h) != int64(wantExec) {
		t.Fatalf("exec mode %v, want %o", seen["run.sh"], wantExec)
	}
	if h := seen["dir/"]; h != nil && perm(h) != int64(wantDir) {
		t.Fatalf("dir mode %o, want %o", h.Mode, wantDir)
	} else if h == nil {
		if h2 := seen["dir"]; h2 == nil || perm(h2) != int64(wantDir) {
			t.Fatalf("dir entry missing or wrong mode: %+v", h2)
		}
	}
	sym := seen["sym"]
	if sym == nil {
		t.Fatalf("symlink missing")
	}
	if sym.Linkname != "target.txt" || sym.Size != 0 {
		t.Fatalf("symlink fields: link=%q size=%d", sym.Linkname, sym.Size)
	}
	if perm := sym.Mode & 0o777; perm != 0o777 {
		t.Fatalf("symlink perm %o, want 777", perm)
	}
}

// 8. A prefix ending in `/` emits an explicit directory entry before the
//    files; the prefix is prepended to every member name.
func TestDetail08(t *testing.T) {
	r := arcFixture(t)
	tree, err := object.GetTree(r.st, r.treeH)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}

	var buf bytes.Buffer
	if err := WriteTarArchive(r.st, &buf, tree, nil, "pfx/", nil, r.when); err != nil {
		t.Fatalf("WriteTarArchive: %v", err)
	}
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	var names []string
	dirSeen := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		names = append(names, hdr.Name)
		if hdr.Name == "pfx/" || hdr.Name == "pfx" {
			dirSeen = true
		}
	}
	if !dirSeen {
		t.Fatalf("no prefix dir entry in %v", names)
	}
	for _, n := range names {
		if n == PAXGlobalHeader {
			continue
		}
		if !strings.HasPrefix(n, "pfx/") && n != "pfx" {
			t.Fatalf("unprefixed member %q", n)
		}
	}
	if !strings.Contains(strings.Join(names, " "), "pfx/README.md") {
		t.Fatalf("prefixed file missing: %v", names)
	}
}

// 9. A symlink target larger than 64KiB is a hard error — the tar format
//    cannot carry it. Inferable: no — assert SHAPE: an error naming the
//    symlink-target limit is returned.
func TestDetail09(t *testing.T) {
	st := memory.NewStorage()
	big := arcBlob(t, st, strings.Repeat("x", maxTarSymlinkTargetSize+1))
	root := arcTree(t, st,
		object.TreeEntry{Name: "sym", Mode: filemode.Symlink, Hash: big},
	)
	tree, err := object.GetTree(st, root)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	var buf bytes.Buffer
	err = WriteTarArchive(st, &buf, tree, nil, "", nil, time.Now())
	if !errors.Is(err, ErrSymlinkTargetTooLarge) {
		t.Fatalf("oversize symlink: got %v, want ErrSymlinkTargetTooLarge", err)
	}
}

// 10. Path filters match exact names, `name/` prefixes, parent dirs of
//     `name`, and glob patterns — and if any filter list is supplied but
//     nothing matched, the write fails with ErrPathspecNoMatch.
func TestDetail10(t *testing.T) {
	cases := []struct {
		name    string
		filters []string
		want    bool
	}{
		{"README.md", []string{"README.md"}, true},      // exact
		{"docs/guide.md", []string{"docs"}, true},       // bare name prefixes children
		{"docs", []string{"docs/"}, true},               // slashed filter matches the dir entry
		{"docs/", []string{"docs/"}, true},              // dir path, slashed filter
		{"docs", []string{"docs/guide.md"}, true},       // parent dir of a filter
		{"main.go", []string{"*.go"}, true},             // glob
		{"x.txt", []string{"*.go"}, false},              // glob negative
		{"docs/guide.md", []string{"other"}, false},     // unrelated
		{"docs/guide.md", nil, false},                   // empty list: function returns false
	}
	for _, c := range cases {
		if got := MatchesPathFilter(c.name, c.filters); got != c.want {
			t.Fatalf("MatchesPathFilter(%q,%v)=%v want %v", c.name, c.filters, got, c.want)
		}
	}

	r := arcFixture(t)
	tree, err := object.GetTree(r.st, r.treeH)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteTarArchive(r.st, &buf, tree, nil, "", []string{"*.zzz"}, r.when); !errors.Is(err, ErrPathspecNoMatch) {
		t.Fatalf("no-match filter: got %v, want ErrPathspecNoMatch", err)
	}

	buf.Reset()
	if err := WriteTarArchive(r.st, &buf, tree, nil, "", []string{"docs"}, r.when); err != nil {
		t.Fatalf("matching filter: %v", err)
	}
	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	var sawGuide bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		if hdr.Name == PAXGlobalHeader {
			continue
		}
		if hdr.Name != "docs" && hdr.Name != "docs/" && !strings.HasPrefix(hdr.Name, "docs/") {
			t.Fatalf("filter leaked member %q", hdr.Name)
		}
		if hdr.Name == "docs/guide.md" {
			sawGuide = true
		}
	}
	if !sawGuide {
		t.Fatalf("filtered member docs/guide.md missing")
	}
}

// 11. Prefixes containing `..` path segments or a leading `/` or `\` are
//     rejected before any output is written.
func TestDetail11(t *testing.T) {
	for _, p := range []string{"../evil/", "a/../b/", "/abs/", "\\win\\", ".."} {
		if !HasInvalidPrefix(p) {
			t.Fatalf("HasInvalidPrefix(%q)=false", p)
		}
	}
	for _, p := range []string{"", "ok/", "a.b/c/", "a..b/"} {
		if HasInvalidPrefix(p) {
			t.Fatalf("HasInvalidPrefix(%q)=true", p)
		}
	}

	r := arcFixture(t)
	tree, err := object.GetTree(r.st, r.treeH)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteArchive(r.st, &buf, tree, &r.commitH, r.when, "tar", "../x/", nil); !errors.Is(err, ErrInvalidPrefix) {
		t.Fatalf("traversal prefix: got %v, want ErrInvalidPrefix", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("output written before prefix rejection")
	}
}

// 12. Zip output records the commit hash as the archive comment, uses
//     Deflate, and maps symlink mode to the 0o120000 bit — directories are
//     not emitted as members.
func TestDetail12(t *testing.T) {
	st := memory.NewStorage()
	reg := arcBlob(t, st, "regular\n")
	link := arcBlob(t, st, "target.txt")
	sub := arcTree(t, st, object.TreeEntry{Name: "inner.txt", Mode: filemode.Regular, Hash: reg})
	root := arcTree(t, st,
		object.TreeEntry{Name: "dir", Mode: filemode.Dir, Hash: sub},
		object.TreeEntry{Name: "sym", Mode: filemode.Symlink, Hash: link},
		object.TreeEntry{Name: "plain.txt", Mode: filemode.Regular, Hash: reg},
	)
	tree, err := object.GetTree(st, root)
	if err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	commit := arcCommit(t, st, root, time.Now())

	var buf bytes.Buffer
	if err := WriteZipArchive(st, &buf, tree, &commit, "", nil, time.Now()); err != nil {
		t.Fatalf("WriteZipArchive: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if zr.Comment != commit.String() {
		t.Fatalf("zip comment %q, want %q", zr.Comment, commit.String())
	}
	var symFound bool
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") || f.Name == "dir" {
			t.Fatalf("directory emitted as zip member: %q", f.Name)
		}
		if f.Method != zip.Deflate {
			t.Fatalf("%q method %d, want Deflate", f.Name, f.Method)
		}
		if f.Name == "sym" {
			symFound = true
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open sym: %v", err)
			}
			body, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("read sym: %v", err)
			}
			if string(body) != "target.txt" {
				t.Fatalf("sym body %q, want link target", body)
			}
		}
	}
	if !symFound {
		t.Fatalf("symlink member missing")
	}
}
