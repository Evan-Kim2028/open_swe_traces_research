package vfs

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/aws/smithy-go"
	"google.golang.org/api/googleapi"
)

// TestDetail01 — Path() formats: scheme://bucket/key (S3 honours the scheme field),
// gs://bucket/key, azureblob://account/container/key, k8s://host/key; String() == Path().
func TestDetail01(t *testing.T) {
	ctx := NewVFSContext()

	s3p := newS3Path(ctx.s3Context, "s3", "bkt", "k/ey", false, nil)
	if got := s3p.Path(); got != "s3://bkt/k/ey" {
		t.Errorf("S3Path.Path() = %q, want s3://bkt/k/ey", got)
	}
	if s3p.String() != s3p.Path() {
		t.Errorf("S3Path.String() = %q != Path() %q", s3p.String(), s3p.Path())
	}
	custom := newS3Path(ctx.s3Context, "custom", "bkt", "k", false, nil)
	if got := custom.Path(); got != "custom://bkt/k" {
		t.Errorf("custom scheme S3Path.Path() = %q, want custom://bkt/k", got)
	}

	gsp := NewGSPath(ctx, "bkt", "k/ey")
	if got := gsp.Path(); got != "gs://bkt/k/ey" {
		t.Errorf("GSPath.Path() = %q, want gs://bkt/k/ey", got)
	}
	if gsp.String() != gsp.Path() {
		t.Errorf("GSPath.String() = %q != Path() %q", gsp.String(), gsp.Path())
	}

	az := NewAzureBlobPath(ctx, "acct", "cont", "k/ey")
	if got := az.Path(); got != "azureblob://acct/cont/k/ey" {
		t.Errorf("AzureBlobPath.Path() = %q, want azureblob://acct/cont/k/ey", got)
	}
	if az.String() != az.Path() {
		t.Errorf("AzureBlobPath.String() = %q != Path() %q", az.String(), az.Path())
	}

	k8s := newKubernetesPath(ctx.k8sContext, "host", "k/ey")
	if got := k8s.Path(); got != "k8s://host/k/ey" {
		t.Errorf("KubernetesPath.Path() = %q, want k8s://host/k/ey", got)
	}
	if k8s.String() != k8s.Path() {
		t.Errorf("KubernetesPath.String() = %q != Path() %q", k8s.String(), k8s.Path())
	}
}

// TestDetail02 — Join prepends the existing key and path.Joins (normalizing ./..),
// keeping bucket/host/context fields unchanged.
func TestDetail02(t *testing.T) {
	ctx := NewVFSContext()

	s3p := newS3Path(ctx.s3Context, "s3", "bkt", "base/dir", false, nil)
	j := s3p.Join("a/./b", "../c")
	if got := j.Path(); got != "s3://bkt/base/dir/a/c" {
		t.Errorf("S3 Join normalized Path() = %q, want s3://bkt/base/dir/a/c", got)
	}
	if sj, ok := j.(*S3Path); !ok || sj.bucket != "bkt" || sj.scheme != "s3" || sj.s3Context != ctx.s3Context {
		t.Errorf("S3 Join did not preserve bucket/scheme/context: %+v", j)
	}

	gj := NewGSPath(ctx, "bkt", "x/y").Join("../z")
	if got := gj.Path(); got != "gs://bkt/x/z" {
		t.Errorf("GS Join normalized Path() = %q, want gs://bkt/x/z", got)
	}
	if gj.(*GSPath).vfsContext != ctx {
		t.Error("GS Join did not preserve vfsContext")
	}

	aj := NewAzureBlobPath(ctx, "acct", "cont", "d").Join("e/./f")
	if got := aj.Path(); got != "azureblob://acct/cont/d/e/f" {
		t.Errorf("Azure Join Path() = %q, want azureblob://acct/cont/d/e/f", got)
	}

	kj := newKubernetesPath(ctx.k8sContext, "host", "p").Join("q/./r")
	if got := kj.Path(); got != "k8s://host/p/q/r" {
		t.Errorf("k8s Join Path() = %q, want k8s://host/p/q/r", got)
	}
}

// TestDetail03 — Base() is path.Base(key): last element, or degenerate "." for empty.
func TestDetail03(t *testing.T) {
	ctx := NewVFSContext()
	if got := newS3Path(ctx.s3Context, "s3", "b", "a/b/c", false, nil).Base(); got != "c" {
		t.Errorf("Base(a/b/c) = %q, want c", got)
	}
	if got := newS3Path(ctx.s3Context, "s3", "b", "", false, nil).Base(); got != "." && got != "/" {
		t.Errorf("Base(empty key) = %q, want path.Base degenerate (. or /)", got)
	}
	if got := NewGSPath(ctx, "b", "d/e").Base(); got != "e" {
		t.Errorf("GS Base(d/e) = %q, want e", got)
	}
	if got := newKubernetesPath(ctx.k8sContext, "h", "m/n").Base(); got != "n" {
		t.Errorf("k8s Base(m/n) = %q, want n", got)
	}
	if got := NewAzureBlobPath(ctx, "a", "c", "u/v").Base(); got != "v" {
		t.Errorf("Azure Base(u/v) = %q, want v", got)
	}
}

// TestDetail04 — S3Path.GetHTTPsUrl resolves the bucket's region and builds a regional
// endpoint (dualstack-aware), appending the key — not the fixed s3.amazonaws.com form.
// The region probe short-circuits to S3_REGION when S3_ENDPOINT is set, so this is
// exercised without network access.
func TestDetail04(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "http://localhost:1")
	t.Setenv("S3_REGION", "eu-west-1")

	ctx := NewVFSContext()
	p, err := ctx.BuildVfsPath("s3://my-bucket/some/key")
	if err != nil {
		t.Fatalf("BuildVfsPath: %v", err)
	}
	s3p, ok := p.(*S3Path)
	if !ok {
		t.Fatalf("BuildVfsPath(s3://) returned %T, want *S3Path", p)
	}

	url, err := s3p.GetHTTPsUrl(false)
	if err != nil {
		t.Fatalf("GetHTTPsUrl: %v", err)
	}
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("GetHTTPsUrl = %q, want https URL", url)
	}
	if !strings.Contains(url, "eu-west-1") {
		t.Errorf("GetHTTPsUrl = %q should resolve the eu-west-1 regional endpoint", url)
	}
	if strings.Contains(url, "//my-bucket.s3.amazonaws.com") || url == "https://s3.amazonaws.com/my-bucket/some/key" {
		t.Errorf("GetHTTPsUrl = %q used the fixed non-regional endpoint", url)
	}
	if !strings.HasSuffix(url, "/some/key") {
		t.Errorf("GetHTTPsUrl = %q should append the key", url)
	}

	dual, err := s3p.GetHTTPsUrl(true)
	if err != nil {
		t.Fatalf("GetHTTPsUrl(dualstack): %v", err)
	}
	if !strings.Contains(dual, "dualstack") {
		t.Errorf("dualstack GetHTTPsUrl = %q, want a dualstack endpoint", dual)
	}
}

// TestDetail05 — GSPath.GetHTTPsUrl is the static storage.googleapis.com/bucket/key form
// with a trailing slash trimmed.
func TestDetail05(t *testing.T) {
	ctx := NewVFSContext()
	url, err := NewGSPath(ctx, "bkt", "k/ey").GetHTTPsUrl()
	if err != nil {
		t.Fatalf("GetHTTPsUrl: %v", err)
	}
	if url != "https://storage.googleapis.com/bkt/k/ey" {
		t.Errorf("GSPath.GetHTTPsUrl = %q, want https://storage.googleapis.com/bkt/k/ey", url)
	}
	url, err = NewGSPath(ctx, "bkt", "dir/").GetHTTPsUrl()
	if err != nil {
		t.Fatalf("GetHTTPsUrl: %v", err)
	}
	if url != "https://storage.googleapis.com/bkt/dir" {
		t.Errorf("trailing slash not trimmed: %q", url)
	}
}

// TestDetail06 — isGCSNotFound accepts the typed SDK sentinels and googleapi 404s.
func TestDetail06(t *testing.T) {
	if !isGCSNotFound(storage.ErrObjectNotExist) {
		t.Error("storage.ErrObjectNotExist should be not-found")
	}
	if !isGCSNotFound(storage.ErrBucketNotExist) {
		t.Error("storage.ErrBucketNotExist should be not-found")
	}
	if !isGCSNotFound(fmt.Errorf("wrap: %w", storage.ErrObjectNotExist)) {
		t.Error("wrapped sentinel should be not-found")
	}
	if !isGCSNotFound(&googleapi.Error{Code: 404}) {
		t.Error("googleapi 404 should be not-found")
	}
	if isGCSNotFound(&googleapi.Error{Code: 403}) {
		t.Error("googleapi 403 should not be not-found")
	}
	if isGCSNotFound(errors.New("boom")) {
		t.Error("generic error should not be not-found")
	}
	if isGCSNotFound(nil) {
		t.Error("nil should not be not-found")
	}
}

// TestDetail07 — GSAcl.String renders entries with %+v inside braces (shape only).
func TestDetail07(t *testing.T) {
	r1 := storage.ACLRule{Entity: "user-one@example.com", Role: storage.RoleReader}
	r2 := storage.ACLRule{Entity: "user-two@example.com", Role: storage.RoleOwner}
	s := (&GSAcl{Acl: []storage.ACLRule{r1, r2}}).String()
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		t.Errorf("GSAcl.String() = %q, want brace-wrapped", s)
	}
	if !strings.Contains(s, fmt.Sprintf("%+v", r1)) || !strings.Contains(s, fmt.Sprintf("%+v", r2)) {
		t.Errorf("GSAcl.String() = %q, want each entry rendered via %%+v", s)
	}
	empty := (&GSAcl{}).String()
	if !strings.HasPrefix(empty, "{") || !strings.HasSuffix(empty, "}") {
		t.Errorf("empty GSAcl.String() = %q, want brace shape", empty)
	}
}

// TestDetail08 — GSPath.TerraformLink emits a google_storage_bucket_object literal
// referencing the given name (shape only).
func TestDetail08(t *testing.T) {
	ctx := NewVFSContext()
	lit := NewGSPath(ctx, "bkt", "k").TerraformLink("myobject")
	if lit == nil {
		t.Fatal("TerraformLink returned nil")
	}
	if !strings.Contains(lit.String, "google_storage_bucket_object") {
		t.Errorf("TerraformLink literal %q should reference google_storage_bucket_object", lit.String)
	}
	if !strings.Contains(lit.String, "myobject") {
		t.Errorf("TerraformLink literal %q should reference the object name", lit.String)
	}
}

// TestDetail09 — AWSErrorCode unwraps smithy.APIError (including wrapped errors).
func TestDetail09(t *testing.T) {
	if got := AWSErrorCode(&smithy.GenericAPIError{Code: "NoSuchBucket", Message: "m"}); got != "NoSuchBucket" {
		t.Errorf("AWSErrorCode = %q, want NoSuchBucket", got)
	}
	if got := AWSErrorCode(fmt.Errorf("outer: %w", &smithy.GenericAPIError{Code: "AccessDenied"})); got != "AccessDenied" {
		t.Errorf("AWSErrorCode should unwrap, got %q", got)
	}
	if got := AWSErrorCode(errors.New("plain")); got != "" {
		t.Errorf("AWSErrorCode(non-API error) = %q, want empty", got)
	}
	if got := AWSErrorCode(nil); got != "" {
		t.Errorf("AWSErrorCode(nil) = %q, want empty", got)
	}
}
