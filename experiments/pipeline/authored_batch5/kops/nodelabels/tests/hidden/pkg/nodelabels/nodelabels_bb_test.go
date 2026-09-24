package nodelabels

import (
	"strings"
	"testing"

	api "example.internal/clustkit/pkg/apis/kops"
	"example.internal/clustkit/pkg/featureflag"
)

func labelsFor(t *testing.T, cluster *api.Cluster, ig *api.InstanceGroup) map[string]string {
	t.Helper()
	labels, err := BuildNodeLabels(cluster, ig)
	if err != nil {
		t.Fatalf("BuildNodeLabels returned error: %v", err)
	}
	return labels
}

// TestDetail01 — role dispatch is exclusive-ordered
// (HasControlPlane > HasAPIServer > HasNode > HasBastion); unknown roles error.
func TestDetail01(t *testing.T) {
	cluster := &api.Cluster{}

	node := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleNode},
	})
	if _, ok := node[RoleLabelNode16]; !ok {
		t.Errorf("node IG missing node role label; got %v", node)
	}

	apiServer := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleAPIServer},
	})
	if _, ok := apiServer[RoleLabelAPIServer16]; !ok {
		t.Errorf("apiserver IG missing apiserver role label; got %v", apiServer)
	}
	if _, ok := apiServer[RoleLabelNode16]; ok {
		t.Errorf("apiserver IG unexpectedly got node role label; got %v", apiServer)
	}

	cp := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleControlPlane},
	})
	if _, ok := cp[RoleLabelControlPlane20]; !ok {
		t.Errorf("control-plane IG missing control-plane label; got %v", cp)
	}
	if _, ok := cp[RoleLabelNode16]; ok {
		t.Errorf("control-plane IG unexpectedly got node role label; got %v", cp)
	}

	bastion := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleBastion},
	})
	for _, key := range []string{RoleLabelNode16, RoleLabelAPIServer16, RoleLabelControlPlane20} {
		if _, ok := bastion[key]; ok {
			t.Errorf("bastion IG unexpectedly got role label %q; got %v", key, bastion)
		}
	}

	// Role matching is plain equality against the constants (see HasX predicates), so a
	// comma-list role matches none of them and is unhandled.
	if _, err := BuildNodeLabels(cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRole("APIServer,Node")},
	}); err == nil {
		t.Error("expected error for comma-list role (no single-role match), got nil")
	}

	// An IG with no matching role errors.
	if _, err := BuildNodeLabels(cluster, &api.InstanceGroup{}); err == nil {
		t.Error("expected error for IG with empty role, got nil")
	}
	if _, err := BuildNodeLabels(cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRole("Bogus")},
	}); err == nil {
		t.Error("expected error for IG with unknown role, got nil")
	}
}

// TestDetail02 — kubelet NodeLabels merge order: cluster-level (ControlPlaneKubelet for
// control-plane, else Kubelet) then IG-level Kubelet, IG overriding cluster.
func TestDetail02(t *testing.T) {
	cluster := &api.Cluster{
		Spec: api.ClusterSpec{
			Kubelet: &api.KubeletConfigSpec{
				NodeLabels: map[string]string{"a": "cluster", "shared": "cluster"},
			},
			ControlPlaneKubelet: &api.KubeletConfigSpec{
				NodeLabels: map[string]string{"cp": "x", "shared": "cp"},
			},
		},
	}

	node := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{
			Role:    api.InstanceGroupRoleNode,
			Kubelet: &api.KubeletConfigSpec{NodeLabels: map[string]string{"shared": "ig", "b": "ig"}},
		},
	})
	if node["a"] != "cluster" || node["b"] != "ig" {
		t.Errorf("node IG did not merge cluster+IG kubelet labels: %v", node)
	}
	if node["shared"] != "ig" {
		t.Errorf("IG kubelet labels should override cluster; shared=%q", node["shared"])
	}

	cp := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{
			Role:    api.InstanceGroupRoleControlPlane,
			Kubelet: &api.KubeletConfigSpec{NodeLabels: map[string]string{"shared": "ig"}},
		},
	})
	if cp["cp"] != "x" {
		t.Errorf("control-plane IG did not merge ControlPlaneKubelet labels: %v", cp)
	}
	if _, ok := cp["a"]; ok {
		t.Errorf("control-plane IG unexpectedly merged Spec.Kubelet when ControlPlaneKubelet set: %v", cp)
	}
	if cp["shared"] != "ig" {
		t.Errorf("IG kubelet labels should override cluster; shared=%q", cp["shared"])
	}

	// Control-plane IGs consult ControlPlaneKubelet, not Spec.Kubelet.
	cluster2 := &api.Cluster{
		Spec: api.ClusterSpec{
			Kubelet: &api.KubeletConfigSpec{NodeLabels: map[string]string{"a": "cluster"}},
		},
	}
	cp2 := labelsFor(t, cluster2, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleControlPlane},
	})
	if _, ok := cp2["a"]; ok {
		t.Errorf("control-plane IG should not merge Spec.Kubelet labels: %v", cp2)
	}
}

// TestDetail03 — apiserver role label: unconditional for apiserver IGs, gated on
// featureflag.APIServerNodes for control-plane IGs.
func TestDetail03(t *testing.T) {
	cluster := &api.Cluster{}
	featureflag.ParseFlags("-APIServerNodes")
	defer featureflag.ParseFlags("-APIServerNodes")

	apiServer := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleAPIServer},
	})
	if _, ok := apiServer[RoleLabelAPIServer16]; !ok {
		t.Errorf("apiserver IG missing apiserver label even with flag off; got %v", apiServer)
	}

	cpOff := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleControlPlane},
	})
	if _, ok := cpOff[RoleLabelAPIServer16]; ok {
		t.Errorf("control-plane IG got apiserver label with APIServerNodes disabled; got %v", cpOff)
	}

	featureflag.ParseFlags("+APIServerNodes")
	cpOn := labelsFor(t, cluster, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleControlPlane},
	})
	if _, ok := cpOn[RoleLabelAPIServer16]; !ok {
		t.Errorf("control-plane IG missing apiserver label with APIServerNodes enabled; got %v", cpOn)
	}
}

// TestDetail04 — role labels carry the empty-string value (documented in the API example).
func TestDetail04(t *testing.T) {
	node := labelsFor(t, &api.Cluster{}, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleNode},
	})
	if v, ok := node[RoleLabelNode16]; !ok || v != "" {
		t.Errorf("node role label = %q (present=%v); expected empty string", v, ok)
	}
}

// TestDetail05 — Spec.NodeLabels overlay last; user labels can clobber role labels.
func TestDetail05(t *testing.T) {
	labels := labelsFor(t, &api.Cluster{}, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{
			Role:       api.InstanceGroupRoleNode,
			NodeLabels: map[string]string{RoleLabelNode16: "custom", "extra": "e"},
		},
	})
	if labels[RoleLabelNode16] != "custom" {
		t.Errorf("user NodeLabels did not clobber role label: %v", labels)
	}
	if labels["extra"] != "e" {
		t.Errorf("user NodeLabels missing from result: %v", labels)
	}
}

// TestDetail06 — all-empty input returns nil, not an empty map.
func TestDetail06(t *testing.T) {
	labels := labelsFor(t, &api.Cluster{}, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleBastion},
	})
	if labels != nil {
		t.Errorf("expected nil map for all-empty bastion IG, got %v", labels)
	}

	node := labelsFor(t, &api.Cluster{}, &api.InstanceGroup{
		Spec: api.InstanceGroupSpec{Role: api.InstanceGroupRoleNode},
	})
	if node == nil {
		t.Error("expected non-nil map for node IG")
	}
}

// TestDetail07 — BuildMandatoryControlPlaneLabels mutates and returns the passed map.
func TestDetail07(t *testing.T) {
	m := map[string]string{"keep": "me"}
	got := BuildMandatoryControlPlaneLabels(m)

	if got["keep"] != "me" {
		t.Errorf("returned map lost existing entry: %v", got)
	}
	got["identity-probe"] = "1"
	if m["identity-probe"] != "1" {
		t.Error("returned map is not the same map as the argument")
	}
	if _, ok := m[RoleLabelControlPlane20]; !ok {
		t.Errorf("input map missing control-plane label after call: %v", m)
	}
	if _, ok := m["node.kubernetes.io/exclude-from-external-load-balancers"]; !ok {
		t.Errorf("input map missing exclude-from-external-load-balancers label: %v", m)
	}
	// api.md commits to exactly three labels added: control-plane role, controller-PKI,
	// exclude-from-external-load-balancers.
	if len(m) != 1+3+1 {
		t.Errorf("expected 3 mandatory labels added, got map %v", m)
	}
}

// TestDetail08 — label key spellings are fixed strings (shape only: non-empty k8s-style
// qualified keys).
func TestDetail08(t *testing.T) {
	m := BuildMandatoryControlPlaneLabels(map[string]string{})
	for k := range m {
		if !strings.Contains(k, "/") {
			t.Errorf("mandatory label key %q lacks qualified-key shape (no '/')", k)
		}
	}
}
