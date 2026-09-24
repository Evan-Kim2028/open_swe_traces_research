// Package vfs_test is the hidden black-box suite for vfspaths.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// BuildVfsPath, RetryWithBackoff, NewVFSContext, NewTestingVFSContext.
package vfs_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"example.internal/clustkit/util/pkg/vfs"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Detail 1 (Inferable: yes): bare path / file:// -> FSPath; each cloud scheme
// routes to its builder; anything else errors.
func TestDetail01(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "https://s3.example.com")
	os.Unsetenv("AZURE_STORAGE_ACCOUNT") // t.Setenv can't unset; be explicit
	c := vfs.NewVFSContext()

	for _, u := range []string{"rel/path", "/abs/path", "file:///abs/file"} {
		p, err := c.BuildVfsPath(u)
		if err != nil {
			t.Fatalf("%q: %v", u, err)
		}
		if _, ok := p.(*vfs.FSPath); !ok {
			t.Fatalf("%q -> %T, want *FSPath", u, p)
		}
	}
	cloud := map[string]string{
		"s3://bucket/key":           "*vfs.S3Path",
		"memfs://cluster/x":         "*vfs.MemFSPath",
		"gs://bucket/key":           "*vfs.GSPath",
		"k8s://whatever":            "*vfs.KubernetesPath",
		"swift://cont/key":          "*vfs.SwiftPath",
		"azureblob://acct/cont/key": "*vfs.AzureBlobPath",
	}
	// memfs needs an initialized context
	tc := vfs.NewTestingVFSContext()
	for u, want := range cloud {
		ctx := c
		if u[:6] == "memfs:" {
			ctx = tc
		}
		p, err := ctx.BuildVfsPath(u)
		if err != nil {
			t.Fatalf("%q: %v", u, err)
		}
		if got := typeName(p); got != want {
			t.Fatalf("%q -> %s, want %s", u, got, want)
		}
	}
	for _, u := range []string{"bogus://x", "http://x"} {
		if _, err := c.BuildVfsPath(u); err == nil {
			t.Fatalf("%q did not error", u)
		}
	}
}

func typeName(p vfs.Path) string {
	switch p.(type) {
	case *vfs.FSPath:
		return "*vfs.FSPath"
	case *vfs.S3Path:
		return "*vfs.S3Path"
	case *vfs.MemFSPath:
		return "*vfs.MemFSPath"
	case *vfs.GSPath:
		return "*vfs.GSPath"
	case *vfs.KubernetesPath:
		return "*vfs.KubernetesPath"
	case *vfs.SwiftPath:
		return "*vfs.SwiftPath"
	case *vfs.AzureBlobPath:
		return "*vfs.AzureBlobPath"
	}
	return "unknown"
}

// Detail 2 (Inferable: partially): builders parse the URL, reject a
// mismatched scheme implicitly via the dispatcher, take the bucket from
// u.Host — an empty bucket is an error.
func TestDetail02(t *testing.T) {
	c := vfs.NewVFSContext()
	if _, err := c.BuildVfsPath("s3:///key"); err == nil {
		t.Fatal("empty s3 bucket accepted")
	}
	p, err := c.BuildVfsPath("s3://bucket/key")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("s3://bucket/key returned nil")
	}
}

// Detail 3 (Inferable: no): s3 is the only S3-family builder that works
// without S3_ENDPOINT; do/linode/hos/scw require it.
func TestDetail03(t *testing.T) {
	os.Unsetenv("S3_ENDPOINT")
	c := vfs.NewVFSContext()
	if _, err := c.BuildVfsPath("s3://bucket/key"); err != nil {
		t.Fatalf("s3 without S3_ENDPOINT: %v", err)
	}
	for _, u := range []string{"do://bucket/key", "linode://bucket/key", "hos://bucket/key", "scw://bucket/key"} {
		if _, err := c.BuildVfsPath(u); err == nil {
			t.Fatalf("%q worked without S3_ENDPOINT", u)
		}
	}
	t.Setenv("S3_ENDPOINT", "https://s3.example.com")
	for _, u := range []string{"do://bucket/key", "linode://bucket/key", "hos://bucket/key", "scw://bucket/key"} {
		p, err := c.BuildVfsPath(u)
		if err != nil {
			t.Fatalf("%q with S3_ENDPOINT: %v", u, err)
		}
		if _, ok := p.(*vfs.S3Path); !ok {
			t.Fatalf("%q -> %T, want *S3Path", u, p)
		}
	}
}

// Detail 4 (Inferable: no): S3_ENDPOINT consumers configure endpoint,
// path-style and checksum behavior. Asserted shape: construction succeeds and
// yields S3Path instances whose key/bucket survived — endpoint internals are
// not exported, so presence of a working path is the observable part.
func TestDetail04(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "https://s3.example.com")
	c := vfs.NewVFSContext()
	for _, u := range []string{"do://b/k", "linode://b/k", "hos://b/k", "scw://b/k"} {
		p, err := c.BuildVfsPath(u)
		if err != nil || p == nil {
			t.Fatalf("%q: %v", u, err)
		}
	}
}

// Detail 5 (Inferable: no): AZURE_STORAGE_ACCOUNT set -> error (account must
// come from the URL); host=account, first path segment=container; missing
// container is an error.
func TestDetail05(t *testing.T) {
	c := vfs.NewVFSContext()
	os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	p, err := c.BuildVfsPath("azureblob://acct/cont/key")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*vfs.AzureBlobPath); !ok {
		t.Fatalf("%T", p)
	}
	if _, err := c.BuildVfsPath("azureblob://acct"); err == nil {
		t.Fatal("missing container accepted")
	}
	t.Setenv("AZURE_STORAGE_ACCOUNT", "acct")
	if _, err := c.BuildVfsPath("azureblob://acct/cont/key"); err == nil {
		t.Fatal("AZURE_STORAGE_ACCOUNT set was accepted")
	}
}

// Detail 6 (Inferable: partially): memfs:// errors when the memfs context is
// uninitialized; NewTestingVFSContext/ResetMemfsContext initialize it.
func TestDetail06(t *testing.T) {
	c := vfs.NewVFSContext()
	if _, err := c.BuildVfsPath("memfs://cluster/x"); err == nil {
		t.Fatal("memfs worked without initialized context")
	}
	tc := vfs.NewTestingVFSContext()
	if _, err := tc.BuildVfsPath("memfs://cluster/x"); err != nil {
		t.Fatalf("testing ctx: %v", err)
	}
	c.ResetMemfsContext(false)
	if _, err := c.BuildVfsPath("memfs://cluster/x"); err != nil {
		t.Fatalf("after ResetMemfsContext: %v", err)
	}
}

// Detail 7 (Inferable: no): RetryWithBackoff runs the condition BEFORE the
// first sleep, counts attempts against Steps, and returns the condition's
// (done, err) on exhaustion — not a timeout error.
func TestDetail07(t *testing.T) {
	sentinel := errors.New("cond-failed")
	calls := 0
	start := time.Now()
	done, err := vfs.RetryWithBackoff(wait.Backoff{Duration: time.Hour, Steps: 1},
		func() (bool, error) { calls++; return false, sentinel })
	if calls != 1 || done || !errors.Is(err, sentinel) {
		t.Fatalf("calls=%d done=%v err=%v", calls, done, err)
	}
	if time.Since(start) > time.Minute {
		t.Fatal("slept before first attempt")
	}

	calls = 0
	done, err = vfs.RetryWithBackoff(wait.Backoff{Duration: time.Millisecond, Steps: 3},
		func() (bool, error) { calls++; return false, sentinel })
	if calls != 3 || done || !errors.Is(err, sentinel) {
		t.Fatalf("calls=%d done=%v err=%v", calls, done, err)
	}

	// success path
	calls = 0
	done, err = vfs.RetryWithBackoff(wait.Backoff{Duration: time.Millisecond, Steps: 5},
		func() (bool, error) { calls++; return calls >= 2, nil })
	if calls != 2 || !done || err != nil {
		t.Fatalf("calls=%d done=%v err=%v", calls, done, err)
	}
}

// Detail 8 (Inferable: partially): backoff multiplies by Factor then clamps
// to Cap only when Cap > 0. Asserted shape: a capped steep backoff completes
// quickly where the uncapped growth would blow the budget.
func TestDetail08(t *testing.T) {
	start := time.Now()
	calls := 0
	vfs.RetryWithBackoff(
		wait.Backoff{Duration: 10 * time.Millisecond, Factor: 1000, Cap: 20 * time.Millisecond, Steps: 4},
		func() (bool, error) { calls++; return false, nil })
	// uncapped this would sleep ~10s; capped it is ~60ms
	if time.Since(start) > 3*time.Second {
		t.Fatal("Cap not applied — backoff grew without bound")
	}
	if calls != 4 {
		t.Fatalf("calls=%d", calls)
	}
}
