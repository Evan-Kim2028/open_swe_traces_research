// Hidden black-box suite for sympath. Exported API only: Walk, IsSymlink.
package sympath_test

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/helm/internal/sympath"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func TestDetail01_WalkLexicalOrderIncludesRoot(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		root := t.TempDir()
		names := []string{"z-last", "a-first", "m-mid"}
		for _, n := range names {
			if err := os.WriteFile(filepath.Join(root, n), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		sub := filepath.Join(root, "d-dir")
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		child := "c" + strconv.Itoa(rng.Intn(9))
		if err := os.WriteFile(filepath.Join(sub, child), []byte("y"), 0o644); err != nil {
			t.Fatal(err)
		}
		var visited []string
		err := sympath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			visited = append(visited, p)
			return nil
		})
		if err != nil {
			t.Fatalf("i=%d Walk: %v", i, err)
		}
		if len(visited) == 0 || visited[0] != root {
			t.Fatalf("i=%d root not first: %v", i, visited)
		}
		// children of root in lexical order among siblings
		var kids []string
		for _, p := range visited[1:] {
			if filepath.Dir(p) == root {
				kids = append(kids, filepath.Base(p))
			}
		}
		if len(kids) < 4 {
			t.Fatalf("i=%d missing kids %v", i, kids)
		}
		for j := 1; j < len(kids); j++ {
			if kids[j-1] > kids[j] {
				t.Fatalf("i=%d kids not lexical: %v", i, kids)
			}
		}
	}
}

func TestDetail02_SymlinkVisitAtLinkPathResolvedInfo(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inside"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "thelink")
	if err := os.Symlink(targetDir, link); err != nil {
		t.Fatal(err)
	}
	var sawLink os.FileInfo
	err := sympath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p == link {
			sawLink = info
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sawLink == nil {
		t.Fatal("walkFn not invoked at the link path")
	}
	if !sawLink.IsDir() {
		t.Fatal("resolved info at link path must describe the directory target")
	}

	dang := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "no-such-target"), dang); err != nil {
		t.Fatal(err)
	}
	err = sympath.Walk(dang, func(p string, info os.FileInfo, e error) error { return e })
	if err == nil {
		t.Fatal("eval-symlink error must surface")
	}
	if !strings.Contains(err.Error(), "error evaluating symlink") {
		t.Fatalf("want wrapped eval error, got %v", err)
	}
	if !strings.Contains(err.Error(), dang) {
		t.Fatalf("eval error must name the link path, got %v", err)
	}
}

func TestDetail03_SkipDirSwallowedOnResolvedLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "tdir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "n"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "ldir")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	err := sympath.Walk(root, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if p == link || strings.HasPrefix(p, link+string(os.PathSeparator)) {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SkipDir inside resolved-link subtree must be swallowed, got %v", err)
	}
}

func TestDetail04_SkipDirOnPlainFilePropagates(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "plain.txt")
	sib := filepath.Join(root, "zz-sib.txt")
	if err := os.WriteFile(f, []byte("p"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sib, []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	var sawSib bool
	err := sympath.Walk(root, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if p == sib {
			sawSib = true
		}
		if p == f {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SkipDir on a non-dir must not fail Walk, got %v", err)
	}
	if sawSib {
		t.Fatal("SkipDir from a plain file skips the rest of the directory (later sibling not visited)")
	}
}

func TestDetail05_RootStatErrorCallbackSkipDirNil(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope", "missing")
	var called bool
	var gotErr error
	err := sympath.Walk(missing, func(p string, info os.FileInfo, e error) error {
		called = true
		if p != missing {
			t.Fatalf("root callback path %q want %q", p, missing)
		}
		if info != nil {
			t.Fatal("root-stat-error must pass nil info")
		}
		gotErr = e
		return filepath.SkipDir
	})
	if !called {
		t.Fatal("walkFn must be invoked on root Lstat error")
	}
	if gotErr == nil {
		t.Fatal("root Lstat error must be passed to walkFn")
	}
	if err != nil {
		t.Fatalf("SkipDir at root-stat-error must yield nil, got %v", err)
	}
}

func TestDetail06_ReadDirFailureGoesToCallback(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "secret")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "x"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	if _, rerr := os.ReadDir(blocked); rerr == nil {
		t.Skip("process can readdir a 000 directory (likely root); cannot force readDirNames failure")
	}

	var saw error
	err := sympath.Walk(root, func(p string, info os.FileInfo, e error) error {
		if p == blocked && e != nil {
			saw = e
			return e
		}
		return e
	})
	if saw == nil {
		t.Fatal("readdir failure must be delivered to walkFn")
	}
	if err == nil {
		t.Fatal("readdir failure returned from walkFn must surface")
	}
}

func TestDetail07_IsSymlinkModeBit(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "f")
	if err := os.WriteFile(reg, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(reg)
	if err != nil {
		t.Fatal(err)
	}
	if sympath.IsSymlink(fi) {
		t.Fatal("regular file must not be a symlink")
	}
	link := filepath.Join(dir, "l")
	if err := os.Symlink(reg, link); err != nil {
		t.Fatal(err)
	}
	lfi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if !sympath.IsSymlink(lfi) {
		t.Fatal("symlink FileInfo must report IsSymlink")
	}
	if lfi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("precondition: Lstat mode bit")
	}
}
