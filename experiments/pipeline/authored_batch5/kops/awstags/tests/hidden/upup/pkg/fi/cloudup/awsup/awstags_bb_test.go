package awsup

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	autoscalingtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

// TestDetail01: the four Find*Tag helpers share semantics — first exact key
// match wins, absent key -> ("", false), nil key/value elements are tolerated.
func TestDetail01(t *testing.T) {
	ec2Tags := []ec2types.Tag{
		{Key: aws.String("k"), Value: aws.String("first")},
		{Key: nil, Value: aws.String("nokey")},
		{Key: aws.String("k"), Value: aws.String("second")},
	}
	if v, ok := FindEC2Tag(ec2Tags, "k"); !ok || v != "first" {
		t.Fatalf("FindEC2Tag first match = %q,%v", v, ok)
	}
	if v, ok := FindEC2Tag(ec2Tags, "absent"); ok || v != "" {
		t.Fatalf("FindEC2Tag absent = %q,%v", v, ok)
	}
	// nil Value on a matching key must not panic and yields the empty string.
	if v, ok := FindEC2Tag([]ec2types.Tag{{Key: aws.String("k")}}, "k"); !ok || v != "" {
		t.Fatalf("FindEC2Tag nil value = %q,%v", v, ok)
	}

	asgTags := []autoscalingtypes.TagDescription{
		{Key: aws.String("k"), Value: aws.String("v1")},
		{Key: aws.String("k"), Value: aws.String("v2")},
	}
	if v, ok := FindASGTag(asgTags, "k"); !ok || v != "v1" {
		t.Fatalf("FindASGTag = %q,%v", v, ok)
	}
	if _, ok := FindASGTag(asgTags, "zzz"); ok {
		t.Fatal("FindASGTag absent matched")
	}

	elbTags := []elbtypes.Tag{
		{Key: aws.String("k"), Value: aws.String("v1")},
		{Key: aws.String("k"), Value: aws.String("v2")},
	}
	if v, ok := FindELBTag(elbTags, "k"); !ok || v != "v1" {
		t.Fatalf("FindELBTag = %q,%v", v, ok)
	}
	if _, ok := FindELBTag(elbTags, "zzz"); ok {
		t.Fatal("FindELBTag absent matched")
	}

	elbv2Tags := []elbv2types.Tag{
		{Key: aws.String("k"), Value: aws.String("v1")},
		{Key: aws.String("k"), Value: aws.String("v2")},
	}
	if v, ok := FindELBV2Tag(elbv2Tags, "k"); !ok || v != "v1" {
		t.Fatalf("FindELBV2Tag = %q,%v", v, ok)
	}
	if _, ok := FindELBV2Tag(elbv2Tags, "zzz"); ok {
		t.Fatal("FindELBV2Tag absent matched")
	}
}

// TestDetail02: AWSErrorCode/AWSErrorMessage unwrap to a smithy.APIError via
// errors.As; non-API errors produce "" rather than the raw message.
func TestDetail02(t *testing.T) {
	apiErr := &smithy.GenericAPIError{Code: "NoSuchThing", Message: "it is not there"}
	if got := AWSErrorCode(apiErr); got != "NoSuchThing" {
		t.Fatalf("AWSErrorCode direct = %q", got)
	}
	if got := AWSErrorMessage(apiErr); got != "it is not there" {
		t.Fatalf("AWSErrorMessage direct = %q", got)
	}

	wrapped := fmt.Errorf("calling ec2: %w", apiErr)
	if got := AWSErrorCode(wrapped); got != "NoSuchThing" {
		t.Fatalf("AWSErrorCode wrapped = %q", got)
	}
	if got := AWSErrorMessage(wrapped); got != "it is not there" {
		t.Fatalf("AWSErrorMessage wrapped = %q", got)
	}

	plain := errors.New("boom")
	if got := AWSErrorCode(plain); got != "" {
		t.Fatalf("AWSErrorCode non-API = %q", got)
	}
	if got := AWSErrorMessage(plain); got != "" {
		t.Fatalf("AWSErrorMessage non-API = %q, want empty not %q", got, "boom")
	}
}

// TestDetail03: EC2TagSpecification yields nothing for an empty tag map, and a
// single spec carrying every pair under the given resource type otherwise.
func TestDetail03(t *testing.T) {
	if got := EC2TagSpecification(ec2types.ResourceTypeInstance, map[string]string{}); len(got) != 0 {
		t.Fatalf("empty map -> %d specs, want 0", len(got))
	}
	if got := EC2TagSpecification(ec2types.ResourceTypeInstance, nil); len(got) != 0 {
		t.Fatalf("nil map -> %d specs, want 0", len(got))
	}

	got := EC2TagSpecification(ec2types.ResourceTypeInstance, map[string]string{"a": "1", "b": "2"})
	if len(got) != 1 {
		t.Fatalf("expected a single TagSpecification, got %d", len(got))
	}
	if got[0].ResourceType != ec2types.ResourceTypeInstance {
		t.Fatalf("ResourceType = %v", got[0].ResourceType)
	}
	seen := map[string]string{}
	for _, tag := range got[0].Tags {
		seen[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	if len(seen) != 2 || seen["a"] != "1" || seen["b"] != "2" {
		t.Fatalf("tags = %v", seen)
	}
}

// TestDetail04: the name helpers return bounded, deterministic names (shape
// only — the exact hash policy is an implementation detail).
func TestDetail04(t *testing.T) {
	for _, in := range []string{"my.cluster.example.com", "short", strings.Repeat("a", 100)} {
		got := GetClusterName40(in)
		if len(got) > 40 {
			t.Fatalf("GetClusterName40(%q) length %d > 40", in, len(got))
		}
		if got == "" {
			t.Fatalf("GetClusterName40(%q) returned empty", in)
		}
		if got2 := GetClusterName40(in); got2 != got {
			t.Fatalf("GetClusterName40 not deterministic: %q vs %q", got, got2)
		}
	}
	for _, in := range []string{"my.cluster.example.com", "short", strings.Repeat("b", 100)} {
		got := GetResourceName32(in, "lb")
		if len(got) > 32 {
			t.Fatalf("GetResourceName32(%q) length %d > 32", in, len(got))
		}
		if !strings.HasPrefix(got, "lb") {
			t.Fatalf("GetResourceName32(%q) = %q, does not carry prefix", in, got)
		}
		if got2 := GetResourceName32(in, "lb"); got2 != got {
			t.Fatalf("GetResourceName32 not deterministic: %q vs %q", got, got2)
		}
	}
}

// TestDetail05: GetResourceName32 sanitizes the cluster part — dots in the
// cluster name may not survive into the result.
func TestDetail05(t *testing.T) {
	got := GetResourceName32("a.b.c", "lb")
	if strings.Contains(got, ".") {
		t.Fatalf("GetResourceName32 kept dots: %q", got)
	}
	if !strings.Contains(got, "a-b-c") {
		t.Fatalf("GetResourceName32 = %q, want the dot-to-dash cluster part", got)
	}
}

// TestDetail06: NameForExternalTargetGroup returns the middle segment of a
// targetgroup/NAME/id ARN resource and rejects malformed ARNs.
func TestDetail06(t *testing.T) {
	arn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/0123456789abcdef"
	got, err := NameForExternalTargetGroup(arn)
	if err != nil {
		t.Fatalf("valid ARN: %v", err)
	}
	if got != "my-tg" {
		t.Fatalf("name = %q, want the middle segment my-tg", got)
	}

	for _, bad := range []string{
		"not-an-arn",
		"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/0123", // not targetgroup
		"arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/onlyname",       // missing third part
		"arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/a/b/c",          // too many parts
		"",
	} {
		if got, err := NameForExternalTargetGroup(bad); err == nil {
			t.Fatalf("malformed ARN %q accepted as %q", bad, got)
		}
	}
}

// TestDetail07: IsIAMNoSuchEntityException unwraps to the typed exception; nil
// is false.
func TestDetail07(t *testing.T) {
	if IsIAMNoSuchEntityException(nil) {
		t.Fatal("nil error reported as NoSuchEntity")
	}
	if IsIAMNoSuchEntityException(errors.New("nope")) {
		t.Fatal("plain error reported as NoSuchEntity")
	}
	typed := &iamtypes.NoSuchEntityException{Message: aws.String("gone")}
	if !IsIAMNoSuchEntityException(typed) {
		t.Fatal("typed NoSuchEntityException not detected")
	}
	if !IsIAMNoSuchEntityException(fmt.Errorf("deleting role: %w", typed)) {
		t.Fatal("wrapped NoSuchEntityException not detected")
	}
}
