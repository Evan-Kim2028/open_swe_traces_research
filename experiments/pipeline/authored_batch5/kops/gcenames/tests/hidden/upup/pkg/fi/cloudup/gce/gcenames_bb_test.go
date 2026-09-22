package gce

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"
)

// TestDetail01: IsNotFound unwraps (errors.As) and checks Code==404;
// IsNotReady does NOT unwrap (direct type assertion) and scans
// Errors[].Reason for "resourceNotReady".
func TestDetail01(t *testing.T) {
	nf := &googleapi.Error{Code: 404}
	if !IsNotFound(nf) {
		t.Fatal("404 not recognised")
	}
	if !IsNotFound(fmt.Errorf("wrap: %w", nf)) {
		t.Fatal("wrapped 404 not recognised")
	}
	if IsNotFound(&googleapi.Error{Code: 403}) {
		t.Fatal("403 is not-found")
	}
	if IsNotFound(errors.New("x")) || IsNotFound(nil) {
		t.Fatal("non-googleapi error is not-found")
	}

	nr := &googleapi.Error{Errors: []googleapi.ErrorItem{{Reason: "resourceNotReady"}}}
	if !IsNotReady(nr) {
		t.Fatal("resourceNotReady not recognised")
	}
	if IsNotReady(fmt.Errorf("wrap: %w", nr)) {
		t.Fatal("IsNotReady unwrapped a wrapped error")
	}
	if IsNotReady(&googleapi.Error{Code: 404}) {
		t.Fatal("404 without Errors[] is not-ready")
	}
	if IsNotReady(&googleapi.Error{Errors: []googleapi.ErrorItem{{Reason: "other"}}}) {
		t.Fatal("unrelated reason is not-ready")
	}
	if IsNotReady(errors.New("x")) || IsNotReady(nil) {
		t.Fatal("non-googleapi error is not-ready")
	}
}

// TestDetail02: name helpers replace "." with "-" in the cluster name.
func TestDetail02(t *testing.T) {
	if got := SafeClusterName("a.b.c.example.com"); got != "a-b-c-example-com" {
		t.Fatalf("SafeClusterName = %q", got)
	}
	if got := SafeObjectName("obj", "a.b"); got != "obj-a-b" {
		t.Fatalf("SafeObjectName = %q", got)
	}
	if strings.Contains(SafeClusterName("a.b"), ".") {
		t.Fatal("dot survived")
	}
}

// TestDetail03: ClusterPrefixedName / ClusterSuffixedName compose
// safeCluster-objectName / objectName-safeCluster truncated to maxLength,
// using a hash suffix when the cluster part must shrink. (The 10-char floor
// and the hash length are not pinned.)
func TestDetail03(t *testing.T) {
	if os.Getenv("BB_FATAL_CHILD") == "1" {
		// objectName alone exceeds maxLength: no scheme can fit a cluster part
		ClusterPrefixedName("averyverylongobjectname", "cluster", 20)
		os.Exit(0)
	}

	if got := ClusterPrefixedName("https", "cluster.example.com", 38); got != "cluster-example-com-https" {
		t.Fatalf("ClusterPrefixedName = %q", got)
	}
	if got := ClusterSuffixedName("obj", "my.cluster", 30); got != "obj-my-cluster" {
		t.Fatalf("ClusterSuffixedName = %q", got)
	}

	// truncation: length respected, object suffix kept, deterministic,
	// input-sensitive (hash component differs across cluster names)
	long1 := ClusterPrefixedName("obj", "averylongclustername.example.com", 20)
	if len(long1) != 20 || !strings.HasSuffix(long1, "-obj") {
		t.Fatalf("prefixed truncation = %q", long1)
	}
	long2 := ClusterPrefixedName("obj", "anotherlongclustername.example.com", 20)
	if long1 == long2 {
		t.Fatal("truncated names not input-sensitive")
	}
	if ClusterPrefixedName("obj", "averylongclustername.example.com", 20) != long1 {
		t.Fatal("truncation not deterministic")
	}
	suf := ClusterSuffixedName("obj", "averylongclustername.example.com", 20)
	if len(suf) != 20 || !strings.HasPrefix(suf, "obj-") {
		t.Fatalf("suffixed truncation = %q", suf)
	}

	// when the fixed part leaves too little room for the cluster portion the
	// helper fails fatally (shape — process death; the exact floor is an
	// implementation detail)
	cmd := exec.Command(os.Args[0], "-test.run=^TestDetail03$")
	cmd.Env = append(os.Environ(), "BB_FATAL_CHILD=1")
	if err := cmd.Run(); err == nil {
		t.Fatal("impossible-length composition silently succeeded")
	}
}

// TestDetail04: LabelForCluster returns a fixed, non-empty label key with the
// safe cluster name as the value.
func TestDetail04(t *testing.T) {
	l := LabelForCluster("my.cluster.example.com")
	if l.Key == "" {
		t.Fatal("empty label key")
	}
	if l.Key != gceLabelNameKubernetesCluster {
		t.Fatalf("label key = %q, want the package cluster-label constant", l.Key)
	}
	if l.Value != "my-cluster-example-com" {
		t.Fatalf("label value = %q", l.Value)
	}
}

// TestDetail05: ServiceAccountName is ClusterSuffixedName at the documented
// length cap.
func TestDetail05(t *testing.T) {
	if got := ServiceAccountName("myrole", "my.cluster"); got != ClusterSuffixedName("myrole", "my.cluster", 30) {
		t.Fatalf("ServiceAccountName = %q", got)
	}
	if got := ServiceAccountName("r", "averyveryverylongclustername.example.com"); len(got) > 30 {
		t.Fatalf("ServiceAccountName len = %d", len(got))
	}
}

// TestDetail06: LastComponent returns the substring after the last "/", or
// the whole string when there is none.
func TestDetail06(t *testing.T) {
	cases := map[string]string{
		"a/b/c":     "c",
		"noslash":   "noslash",
		"a/b/c/d-e": "d-e",
		"a/b/":      "",
	}
	for in, want := range cases {
		if got := LastComponent(in); got != want {
			t.Fatalf("LastComponent(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDetail07: SSHUsernameForImage matches a case-insensitive "ubuntu"
// prefix on the last path component; anything else yields the primary SSH
// username ("admin").
func TestDetail07(t *testing.T) {
	for _, img := range []string{"ubuntu-2204-lts", "UBUNTU-2404", "projects/p/global/images/ubuntu-server", "ubuntufoo"} {
		if got := SSHUsernameForImage(img); got != "ubuntu" {
			t.Fatalf("SSHUsernameForImage(%q) = %q, want ubuntu", img, got)
		}
	}
	for _, img := range []string{"debian-11", "notubuntu", "projects/p/images/debian", "x/notubuntu-y"} {
		if got := SSHUsernameForImage(img); got != "admin" {
			t.Fatalf("SSHUsernameForImage(%q) = %q, want admin", img, got)
		}
	}
}

// TestDetail08: ZoneToRegion drops the last dash-segment; zones with <=2
// segments error.
func TestDetail08(t *testing.T) {
	got, err := ZoneToRegion("us-central1-a")
	if err != nil || got != "us-central1" {
		t.Fatalf("ZoneToRegion(us-central1-a) = %q %v", got, err)
	}
	got, err = ZoneToRegion("europe-west1-b")
	if err != nil || got != "europe-west1" {
		t.Fatalf("ZoneToRegion(europe-west1-b) = %q %v", got, err)
	}
	if _, err := ZoneToRegion("us-central1"); err == nil {
		t.Fatal("two-segment zone accepted")
	}
	if _, err := ZoneToRegion("nohyphen"); err == nil {
		t.Fatal("one-segment zone accepted")
	}
}
