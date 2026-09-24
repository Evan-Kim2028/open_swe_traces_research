package kops

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func bbIG(role InstanceGroupRole) *InstanceGroup {
	return &InstanceGroup{Spec: InstanceGroupSpec{Role: role}}
}

// TestDetail01: each HasX() is a plain equality check against its
// InstanceGroupRoleX constant — no aliasing or case folding.
func TestDetail01(t *testing.T) {
	pairs := []struct {
		role InstanceGroupRole
		has  func(InstanceGroupRole) bool
	}{
		{InstanceGroupRoleControlPlane, func(r InstanceGroupRole) bool { return r.HasControlPlane() }},
		{InstanceGroupRoleNode, func(r InstanceGroupRole) bool { return r.HasNode() }},
		{InstanceGroupRoleBastion, func(r InstanceGroupRole) bool { return r.HasBastion() }},
		{InstanceGroupRoleAPIServer, func(r InstanceGroupRole) bool { return r.HasAPIServer() }},
		{InstanceGroupRoleEtcd, func(r InstanceGroupRole) bool { return r.HasEtcd() }},
		{InstanceGroupRoleScheduler, func(r InstanceGroupRole) bool { return r.HasScheduler() }},
		{InstanceGroupRoleKubeControllerManager, func(r InstanceGroupRole) bool { return r.HasKubeControllerManager() }},
	}
	for _, p := range pairs {
		for _, other := range AllInstanceGroupRoles {
			got := p.has(other)
			want := other == p.role
			if got != want {
				t.Fatalf("Has(%s) on %s = %v, want %v", p.role, other, got, want)
			}
		}
	}
	// no case folding / aliasing
	if InstanceGroupRole("node").HasNode() || InstanceGroupRole("controlplane").HasControlPlane() {
		t.Fatal("case-insensitive or alias match")
	}
}

// TestDetail02: IsControlPlaneType covers ControlPlane, APIServer, Etcd,
// Scheduler, KubeControllerManager — not Node or Bastion.
func TestDetail02(t *testing.T) {
	for _, r := range []InstanceGroupRole{
		InstanceGroupRoleControlPlane, InstanceGroupRoleAPIServer, InstanceGroupRoleEtcd,
		InstanceGroupRoleScheduler, InstanceGroupRoleKubeControllerManager,
	} {
		if !r.IsControlPlaneType() {
			t.Fatalf("%s should be a control-plane type", r)
		}
	}
	for _, r := range []InstanceGroupRole{InstanceGroupRoleNode, InstanceGroupRoleBastion} {
		if r.IsControlPlaneType() {
			t.Fatalf("%s is not a control-plane type", r)
		}
	}
}

// TestDetail03: ToLowerString — "control-plane" for the control-plane role,
// otherwise the lowercased role string (no inserted separators).
func TestDetail03(t *testing.T) {
	cases := map[InstanceGroupRole]string{
		InstanceGroupRoleControlPlane:           "control-plane",
		InstanceGroupRoleNode:                   "node",
		InstanceGroupRoleBastion:                "bastion",
		InstanceGroupRoleAPIServer:              "apiserver",
		InstanceGroupRoleEtcd:                   "etcd",
		InstanceGroupRoleScheduler:              "scheduler",
		InstanceGroupRoleKubeControllerManager:  "kubecontrollermanager",
	}
	for r, want := range cases {
		if got := r.ToLowerString(); got != want {
			t.Fatalf("ToLowerString(%s) = %q, want %q", r, got, want)
		}
	}
}

// TestDetail04: RunsAPIServer/RunsEtcd/RunsScheduler/RunsKubeControllerManager
// are IsControlPlane() || IsXOnly() — control-plane groups run all four.
func TestDetail04(t *testing.T) {
	cp := bbIG(InstanceGroupRoleControlPlane)
	if !cp.RunsAPIServer() || !cp.RunsEtcd() || !cp.RunsScheduler() || !cp.RunsKubeControllerManager() {
		t.Fatal("control-plane group should run all control-plane daemons")
	}
	node := bbIG(InstanceGroupRoleNode)
	if node.RunsAPIServer() || node.RunsEtcd() || node.RunsScheduler() || node.RunsKubeControllerManager() {
		t.Fatal("node group runs control-plane daemons")
	}
	api := bbIG(InstanceGroupRoleAPIServer)
	if !api.RunsAPIServer() {
		t.Fatal("APIServer-only group does not run apiserver")
	}
	if api.RunsEtcd() || api.RunsScheduler() || api.RunsKubeControllerManager() {
		t.Fatal("APIServer-only group runs other daemons")
	}
	etc := bbIG(InstanceGroupRoleEtcd)
	if !etc.RunsEtcd() || etc.RunsAPIServer() {
		t.Fatal("Etcd-only group misreported")
	}
	bastion := bbIG(InstanceGroupRoleBastion)
	if bastion.RunsAPIServer() || bastion.RunsEtcd() {
		t.Fatal("bastion group runs daemons")
	}
}

// TestDetail05: IsControlPlane / IsAPIServerOnly / IsBastion are single-role
// equality — no wildcard or empty-role fallback.
func TestDetail05(t *testing.T) {
	if !bbIG(InstanceGroupRoleControlPlane).IsControlPlane() {
		t.Fatal("ControlPlane not IsControlPlane")
	}
	if bbIG(InstanceGroupRoleAPIServer).IsControlPlane() {
		t.Fatal("APIServer is IsControlPlane")
	}
	if bbIG(InstanceGroupRoleNode).IsControlPlane() {
		t.Fatal("Node is IsControlPlane")
	}
	if !bbIG(InstanceGroupRoleAPIServer).IsAPIServerOnly() {
		t.Fatal("APIServer not IsAPIServerOnly")
	}
	if bbIG(InstanceGroupRoleControlPlane).IsAPIServerOnly() {
		t.Fatal("ControlPlane is IsAPIServerOnly")
	}
	if !bbIG(InstanceGroupRoleBastion).IsBastion() {
		t.Fatal("Bastion not IsBastion")
	}
	if bbIG(InstanceGroupRoleNode).IsBastion() {
		t.Fatal("Node is IsBastion")
	}
	empty := bbIG("")
	if empty.IsControlPlane() || empty.IsAPIServerOnly() || empty.IsBastion() {
		t.Fatal("empty role matched a predicate")
	}
}

// TestDetail06: HasGVisor requires role Node AND Spec.Containerd.GVisor
// .Enabled == true; every missing level is false, not a panic.
func TestDetail06(t *testing.T) {
	tru := true
	node := bbIG(InstanceGroupRoleNode)
	node.Spec.Containerd = &ContainerdConfig{GVisor: &GVisorConfig{Enabled: &tru}}
	if !node.HasGVisor() {
		t.Fatal("node with gvisor enabled not detected")
	}

	fls := false
	for _, g := range []*InstanceGroup{
		bbIG(InstanceGroupRoleNode),                                                          // no Containerd
		{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode, Containerd: &ContainerdConfig{}}}, // no GVisor
		{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode, Containerd: &ContainerdConfig{GVisor: &GVisorConfig{}}}},                 // Enabled nil
		{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode, Containerd: &ContainerdConfig{GVisor: &GVisorConfig{Enabled: &fls}}}},    // false
		{Spec: InstanceGroupSpec{Role: InstanceGroupRoleControlPlane, Containerd: &ContainerdConfig{GVisor: &GVisorConfig{Enabled: &tru}}}}, // wrong role
	} {
		if g.HasGVisor() {
			t.Fatalf("HasGVisor true for %+v", g.Spec.Role)
		}
	}
}

// TestDetail07: IsKarpenterManaged requires Manager == Karpenter AND
// role Node.
func TestDetail07(t *testing.T) {
	km := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode, Manager: InstanceManagerKarpenter}}
	if !km.IsKarpenterManaged() {
		t.Fatal("karpenter node group not detected")
	}
	kcp := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleControlPlane, Manager: InstanceManagerKarpenter}}
	if kcp.IsKarpenterManaged() {
		t.Fatal("karpenter control-plane group reported managed")
	}
	cg := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode, Manager: InstanceManagerCloudGroup}}
	if cg.IsKarpenterManaged() {
		t.Fatal("cloudgroup node group reported managed")
	}
	none := bbIG(InstanceGroupRoleNode)
	if none.IsKarpenterManaged() {
		t.Fatal("unmanaged node group reported managed")
	}
}

// TestDetail08: AddInstanceGroupNodeLabel writes the instancegroup label
// (per api.md: "kops.k8s.io/instancegroup") with the group name, allocating
// NodeLabels when nil.
func TestDetail08(t *testing.T) {
	ig := &InstanceGroup{ObjectMeta: metav1.ObjectMeta{Name: "nodes-1"}}
	ig.AddInstanceGroupNodeLabel()
	if ig.Spec.NodeLabels == nil {
		t.Fatal("NodeLabels not allocated")
	}
	if got := ig.Spec.NodeLabels["kops.k8s.io/instancegroup"]; got != "nodes-1" {
		t.Fatalf("label value = %q, want nodes-1", got)
	}
	// existing labels preserved
	ig2 := &InstanceGroup{ObjectMeta: metav1.ObjectMeta{Name: "ig2"}}
	ig2.Spec.NodeLabels = map[string]string{"existing": "label"}
	ig2.AddInstanceGroupNodeLabel()
	if ig2.Spec.NodeLabels["existing"] != "label" || ig2.Spec.NodeLabels["kops.k8s.io/instancegroup"] != "ig2" {
		t.Fatalf("labels = %v", ig2.Spec.NodeLabels)
	}
}
