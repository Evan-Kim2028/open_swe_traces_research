// Package vfs_test — hidden black-box property suite for memfs (seed 20260919).
//
// Contract coverage:
//   exclusive create vs overwrite write -> TestMemFsCreateWriteProperty
//   read missing -> TestMemFsCreateWriteProperty
//   ReadDir immediate children -> TestMemFsReadDirProperty
//   ReadTree leaves only -> TestMemFsReadTreeProperty
//   Remove / RemoveAll -> TestMemFsRemoveProperty
//   unseen random path ops -> TestMemFsUnseenRandomProperty
package vfs_test

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"example.internal/clustkit/util/pkg/vfs"
)

const bbSeed = 20260919
const bbCases = 10000

func bbCtx() context.Context { return context.Background() }

func bbRS(s string) io.ReadSeeker { return bytes.NewReader([]byte(s)) }

func bbNewRoot(t *testing.T) *vfs.MemFSPath {
	t.Helper()
	ctx := vfs.NewMemFSContext()
	return vfs.NewMemFSPath(ctx, "")
}

func bbJoin(t *testing.T, root *vfs.MemFSPath, parts ...string) vfs.Path {
	t.Helper()
	return root.Join(parts...)
}

func bbLoc(p vfs.Path) string {
	if m, ok := p.(*vfs.MemFSPath); ok {
		return m.Location()
	}
	return p.Path()
}

func bbLeavesUnder(t *testing.T, root *vfs.MemFSPath, prefix string) []string {
	t.Helper()
	ctx := bbCtx()
	all, err := root.ReadTree(ctx)
	if err != nil {
		t.Fatalf("ReadTree: %v", err)
	}
	var out []string
	for _, p := range all {
		loc := bbLoc(p)
		if prefix == "" || strings.HasPrefix(loc, prefix) {
			out = append(out, loc)
		}
	}
	sort.Strings(out)
	return out
}

func TestMemFsCreateWriteProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		root := bbNewRoot(t)
		rel := "f" + string(rune('a'+rng.Intn(26))) + rngString(rng, 3)
		p := bbJoin(t, root, rel)
		if err := p.CreateFile(bbCtx(), bbRS("a"), nil); err != nil {
			t.Fatalf("case %d create: %v", i, err)
		}
		if err := p.CreateFile(bbCtx(), bbRS("b"), nil); err != os.ErrExist {
			t.Fatalf("case %d second create want ErrExist got %v", i, err)
		}
		if err := p.WriteFile(bbCtx(), bbRS("c"), nil); err != nil {
			t.Fatalf("case %d write: %v", i, err)
		}
		b, err := p.ReadFile(bbCtx())
		if err != nil {
			t.Fatalf("case %d read: %v", i, err)
		}
		if string(b) != "c" {
			t.Fatalf("case %d read %q", i, b)
		}
		empty := bbJoin(t, root, "empty"+rngString(rng, 2))
		if _, err := empty.ReadFile(bbCtx()); err != os.ErrNotExist {
			t.Fatalf("case %d empty read want ErrNotExist got %v", i, err)
		}
	}
}

func TestMemFsReadDirProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		root := bbNewRoot(t)
		dir := bbJoin(t, root, "d"+rngString(rng, 2))
		nKids := 1 + rng.Intn(4)
		want := make([]string, 0, nKids)
		for k := 0; k < nKids; k++ {
			name := "k" + rngString(rng, 2) + string(rune('a'+k))
			want = append(want, name)
			child := dir.Join(name)
			if err := child.WriteFile(bbCtx(), bbRS("x"), nil); err != nil {
				t.Fatalf("case %d write child: %v", i, err)
			}
		}
		sort.Strings(want)
		kids, err := dir.ReadDir()
		if err != nil {
			t.Fatalf("case %d readdir: %v", i, err)
		}
		got := make([]string, 0, len(kids))
		for _, c := range kids {
			got = append(got, c.Base())
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("case %d readdir got %v want %v", i, got, want)
		}
	}
}

func TestMemFsReadTreeProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		root := bbNewRoot(t)
		depth := 1 + rng.Intn(3)
		leafPath := make([]string, 0, depth)
		for d := 0; d < depth; d++ {
			leafPath = append(leafPath, "p"+rngString(rng, 1))
		}
		leafPath = append(leafPath, "leaf.go")
		p := bbJoin(t, root, leafPath...)
		content := rngString(rng, 5+rng.Intn(8))
		if err := p.WriteFile(bbCtx(), bbRS(content), nil); err != nil {
			t.Fatalf("case %d write leaf: %v", i, err)
		}
		leaves := bbLeavesUnder(t, root, "")
		full := strings.Join(leafPath, "/")
		if len(leaves) != 1 || leaves[0] != full {
			t.Fatalf("case %d leaves %v want [%s]", i, leaves, full)
		}
	}
}

func TestMemFsRemoveProperty(t *testing.T) {
	for i := 0; i < bbCases; i++ {
		root := bbNewRoot(t)
		a := bbJoin(t, root, "a", "b", "c.txt")
		if err := a.WriteFile(bbCtx(), bbRS("z"), nil); err != nil {
			t.Fatalf("case %d write: %v", i, err)
		}
		if err := a.Remove(bbCtx()); err != nil {
			t.Fatalf("case %d remove: %v", i, err)
		}
		if _, err := a.ReadFile(bbCtx()); err != os.ErrNotExist {
			t.Fatalf("case %d after remove read: %v", i, err)
		}
		// RemoveAll clears subtree leaves
		root2 := bbNewRoot(t)
		p1 := bbJoin(t, root2, "x", "1")
		p2 := bbJoin(t, root2, "x", "2")
		_ = p1.WriteFile(bbCtx(), bbRS("1"), nil)
		_ = p2.WriteFile(bbCtx(), bbRS("2"), nil)
		parent := bbJoin(t, root2, "x")
		if err := parent.RemoveAll(bbCtx()); err != nil {
			t.Fatalf("case %d removeall: %v", i, err)
		}
		if _, err := p1.ReadFile(bbCtx()); err != os.ErrNotExist {
			t.Fatalf("case %d p1 after removeall: %v", i, err)
		}
		if _, err := p2.ReadFile(bbCtx()); err != os.ErrNotExist {
			t.Fatalf("case %d p2 after removeall: %v", i, err)
		}
	}
}

func TestMemFsUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 99))
	for i := 0; i < bbCases; i++ {
		root := bbNewRoot(t)
		ops := 1 + rng.Intn(6)
		written := map[string]string{}
		for o := 0; o < ops; o++ {
			depth := 1 + rng.Intn(3)
			segs := make([]string, depth)
			for d := 0; d < depth; d++ {
				segs[d] = rngString(rng, 1+rng.Intn(3))
			}
			key := strings.Join(segs, "/")
			p := bbJoin(t, root, segs...)
			val := rngString(rng, 4)
			switch rng.Intn(3) {
			case 0:
				if _, ok := written[key]; !ok {
					cerr := p.CreateFile(bbCtx(), bbRS(val), nil)
					if cerr != nil && cerr != os.ErrExist {
						t.Fatalf("case %d create %s: %v", i, key, cerr)
					}
					if cerr == nil {
						written[key] = val
					}
				}
			case 1:
				if err := p.WriteFile(bbCtx(), bbRS(val), nil); err != nil {
					t.Fatalf("case %d write %s: %v", i, key, err)
				}
				written[key] = val
			case 2:
				if _, ok := written[key]; ok {
					_ = p.Remove(bbCtx())
					delete(written, key)
				}
			}
		}
		for k, v := range written {
			p := bbJoin(t, root, strings.Split(k, "/")...)
			b, err := p.ReadFile(bbCtx())
			if err != nil {
				t.Fatalf("case %d verify %s: %v", i, k, err)
			}
			if string(b) != v {
				t.Fatalf("case %d verify %s got %q want %q", i, k, b, v)
			}
		}
	}
}

func rngString(rng *rand.Rand, n int) string {
	const alpha = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alpha[rng.Intn(len(alpha))]
	}
	return string(b)
}
