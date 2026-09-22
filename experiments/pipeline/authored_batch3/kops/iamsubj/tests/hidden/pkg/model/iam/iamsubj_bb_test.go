// Package iam_test is the hidden black-box suite for iamsubj.
// One TestDetailNN per DETAILS.md commitment. Exported API only.
package iam_test

import (
	"strings"
	"testing"

	kops "example.internal/clustkit/pkg/apis/kops"
	iam "example.internal/clustkit/pkg/model/iam"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func awsCtx() *iam.IAMModelContext {
	return &iam.IAMModelContext{
		AWSAccountID: "123456789012",
		AWSPartition: "aws",
		Cluster: &kops.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "c1.test"},
			Spec: kops.ClusterSpec{
				CloudProvider: kops.CloudProviderSpec{AWS: &kops.AWSSpec{}},
			},
		},
	}
}

func gceCtx() *iam.IAMModelContext {
	return &iam.IAMModelContext{
		Cluster: &kops.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "c1.test"},
			Spec: kops.ClusterSpec{
				CloudProvider: kops.CloudProviderSpec{GCE: &kops.GCESpec{}},
			},
		},
	}
}

func podSpecWith(n int) *corev1.PodSpec {
	ps := &corev1.PodSpec{}
	for i := 0; i < n; i++ {
		ps.Containers = append(ps.Containers, corev1.Container{Name: "c"})
	}
	return ps
}

var saRole = &iam.GenericServiceAccount{
	NamespacedName: types.NamespacedName{Namespace: "ns", Name: "sa"},
	Policy:         &iam.Policy{},
}

// Detail 1 (Inferable: yes): node-role subjects return empty NamespacedName +
// false.
func TestDetail01(t *testing.T) {
	for _, s := range []iam.Subject{
		&iam.NodeRoleMaster{}, &iam.NodeRoleAPIServer{}, &iam.NodeRoleNode{}, &iam.NodeRoleBastion{},
	} {
		nn, ok := s.ServiceAccount()
		if ok || nn != (types.NamespacedName{}) {
			t.Fatalf("%T ServiceAccount = (%v,%v)", s, nn, ok)
		}
	}
}

// Detail 2 (Inferable: yes): GenericServiceAccount echoes its NamespacedName
// + true; BuildAWSPolicy returns the stored Policy verbatim.
func TestDetail02(t *testing.T) {
	nn, ok := saRole.ServiceAccount()
	if !ok || nn != saRole.NamespacedName {
		t.Fatalf("echo = (%v,%v)", nn, ok)
	}
	p, err := saRole.BuildAWSPolicy(nil)
	if err != nil {
		t.Fatalf("BuildAWSPolicy: %v", err)
	}
	if p != saRole.Policy {
		t.Fatal("BuildAWSPolicy did not return the stored Policy verbatim")
	}
}

// Detail 3 (Inferable: partially): BuildNodeRoleSubject maps the four node
// roles to their subject types; anything else errors.
func TestDetail03(t *testing.T) {
	cases := []struct {
		role kops.InstanceGroupRole
		want iam.Subject
	}{
		{kops.InstanceGroupRoleControlPlane, &iam.NodeRoleMaster{}},
		{kops.InstanceGroupRoleAPIServer, &iam.NodeRoleAPIServer{}},
		{kops.InstanceGroupRoleNode, &iam.NodeRoleNode{}},
		{kops.InstanceGroupRoleBastion, &iam.NodeRoleBastion{}},
	}
	for _, c := range cases {
		for _, flag := range []bool{false, true} {
			s, err := iam.BuildNodeRoleSubject(c.role, flag)
			if err != nil {
				t.Fatalf("%v flag=%v: %v", c.role, flag, err)
			}
			switch c.want.(type) {
			case *iam.NodeRoleMaster:
				if _, ok := s.(*iam.NodeRoleMaster); !ok {
					t.Fatalf("%v -> %T, want *NodeRoleMaster", c.role, s)
				}
			case *iam.NodeRoleAPIServer:
				if _, ok := s.(*iam.NodeRoleAPIServer); !ok {
					t.Fatalf("%v -> %T, want *NodeRoleAPIServer", c.role, s)
				}
			case *iam.NodeRoleNode:
				if _, ok := s.(*iam.NodeRoleNode); !ok {
					t.Fatalf("%v -> %T, want *NodeRoleNode", c.role, s)
				}
			case *iam.NodeRoleBastion:
				if _, ok := s.(*iam.NodeRoleBastion); !ok {
					t.Fatalf("%v -> %T, want *NodeRoleBastion", c.role, s)
				}
			}
		}
	}
	for _, bad := range []kops.InstanceGroupRole{"Bogus", kops.InstanceGroupRoleEtcd, ""} {
		if s, err := iam.BuildNodeRoleSubject(bad, false); err == nil {
			t.Fatalf("role %q mapped to %T without error", bad, s)
		}
	}
}

// Detail 4 (Inferable: partially): AddServiceAccountRole dispatches on cloud
// provider — non-AWS errors, AWS proceeds.
func TestDetail04(t *testing.T) {
	if err := iam.AddServiceAccountRole(gceCtx(), podSpecWith(1), saRole); err == nil {
		t.Fatal("GCE cluster did not error")
	}
	if err := iam.AddServiceAccountRole(awsCtx(), podSpecWith(1), saRole); err != nil {
		t.Fatalf("AWS cluster errored: %v", err)
	}
}

// Detail 5 (Inferable: no): the AWS path adds exactly one projected volume
// containing a ServiceAccountToken projection (audience/path/expiry are
// unpinned literals — asserted for presence/shape only).
func TestDetail05(t *testing.T) {
	ps := podSpecWith(1)
	if err := iam.AddServiceAccountRole(awsCtx(), ps, saRole); err != nil {
		t.Fatal(err)
	}
	if len(ps.Volumes) != 1 {
		t.Fatalf("expected exactly 1 added volume, got %d", len(ps.Volumes))
	}
	v := ps.Volumes[0]
	if v.Projected == nil {
		t.Fatal("added volume is not a projected volume")
	}
	foundToken := false
	for _, src := range v.Projected.Sources {
		if src.ServiceAccountToken != nil {
			foundToken = true
			if src.ServiceAccountToken.Path == "" {
				t.Fatal("token projection has empty path")
			}
			if src.ServiceAccountToken.Audience == "" {
				t.Fatal("token projection has empty audience")
			}
			if src.ServiceAccountToken.ExpirationSeconds == nil || *src.ServiceAccountToken.ExpirationSeconds <= 0 {
				t.Fatal("token projection has no expiry")
			}
		}
	}
	if !foundToken {
		t.Fatal("no ServiceAccountToken projection added")
	}
}

// Detail 6 (Inferable: no): every container gains a read-only mount into the
// token volume plus env vars carrying the role ARN and token file path.
// Spellings unpinned; the mount<->volume and env<->token relations are
// asserted.
func TestDetail06(t *testing.T) {
	ps := podSpecWith(2)
	if err := iam.AddServiceAccountRole(awsCtx(), ps, saRole); err != nil {
		t.Fatal(err)
	}
	if len(ps.Volumes) == 0 {
		t.Fatal("no volume added")
	}
	volName := ps.Volumes[0].Name
	for i, c := range ps.Containers {
		var mount *corev1.VolumeMount
		for j := range c.VolumeMounts {
			if c.VolumeMounts[j].Name == volName {
				mount = &c.VolumeMounts[j]
			}
		}
		if mount == nil {
			t.Fatalf("container %d: no mount for the token volume", i)
		}
		if !mount.ReadOnly {
			t.Fatalf("container %d: token mount not read-only", i)
		}
		if mount.MountPath == "" {
			t.Fatalf("container %d: empty mount path", i)
		}
		if len(c.Env) < 2 {
			t.Fatalf("container %d: expected >=2 env vars, got %d", i, len(c.Env))
		}
		arn, tokenFile := false, false
		for _, e := range c.Env {
			if e.Name == "" {
				t.Fatalf("container %d: env var with empty name", i)
			}
			if strings.HasPrefix(e.Value, "arn:") {
				arn = true
			}
			if strings.HasPrefix(e.Value, mount.MountPath) {
				tokenFile = true
			}
		}
		if !arn {
			t.Fatalf("container %d: no env var carries an arn: value", i)
		}
		if !tokenFile {
			t.Fatalf("container %d: no env var points into the token mount", i)
		}
	}
}

// Detail 7 (Inferable: no): the pod SecurityContext gains an FSGroup —
// created when nil, left alone when already set.
func TestDetail07(t *testing.T) {
	ps := podSpecWith(1)
	if err := iam.AddServiceAccountRole(awsCtx(), ps, saRole); err != nil {
		t.Fatal(err)
	}
	if ps.SecurityContext == nil || ps.SecurityContext.FSGroup == nil {
		t.Fatal("SecurityContext/FSGroup not set on nil-securitycontext pod")
	}
	pre := int64(9999)
	ps2 := podSpecWith(1)
	ps2.SecurityContext = &corev1.PodSecurityContext{FSGroup: &pre}
	if err := iam.AddServiceAccountRole(awsCtx(), ps2, saRole); err != nil {
		t.Fatal(err)
	}
	if ps2.SecurityContext.FSGroup == nil || *ps2.SecurityContext.FSGroup != 9999 {
		t.Fatal("pre-set FSGroup was overwritten")
	}
}
