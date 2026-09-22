package kops

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func bbCluster(name string) *Cluster {
	return &Cluster{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func bbPanicked(f func()) (panicked bool) {
	panicked = true
	defer func() {
		if recover() == nil {
			panicked = false
		}
	}()
	f()
	return false
}

// TestDetail01: IsEmpty methods check the listed pointer/map fields.
func TestDetail01(t *testing.T) {
	if !(&AuthenticationSpec{}).IsEmpty() {
		t.Fatal("empty AuthenticationSpec not empty")
	}
	if (&AuthenticationSpec{Kopeio: &KopeioAuthenticationSpec{}}).IsEmpty() {
		t.Fatal("Kopeio set but IsEmpty")
	}
	if (&AuthenticationSpec{AWS: &AWSAuthenticationSpec{}}).IsEmpty() {
		t.Fatal("AWS set but IsEmpty")
	}
	if (&AuthenticationSpec{OIDC: &OIDCAuthenticationSpec{}}).IsEmpty() {
		t.Fatal("OIDC set but IsEmpty")
	}

	if !(&AuthorizationSpec{}).IsEmpty() {
		t.Fatal("empty AuthorizationSpec not empty")
	}
	if (&AuthorizationSpec{RBAC: &RBACAuthorizationSpec{}}).IsEmpty() {
		t.Fatal("RBAC set but IsEmpty")
	}
	if (&AuthorizationSpec{AlwaysAllow: &AlwaysAllowAuthorizationSpec{}}).IsEmpty() {
		t.Fatal("AlwaysAllow set but IsEmpty")
	}

	if !(&TargetSpec{}).IsEmpty() {
		t.Fatal("empty TargetSpec not empty")
	}
	if (&TargetSpec{Terraform: &TerraformSpec{}}).IsEmpty() {
		t.Fatal("Terraform set but IsEmpty")
	}

	if !(&TerraformSpec{}).IsEmpty() {
		t.Fatal("empty TerraformSpec not empty")
	}
	if (&TerraformSpec{ProviderExtraConfig: map[string]string{"a": "b"}}).IsEmpty() {
		t.Fatal("ProviderExtraConfig set but IsEmpty")
	}
	if (&TerraformSpec{FilesProviderExtraConfig: map[string]string{"a": "b"}}).IsEmpty() {
		t.Fatal("FilesProviderExtraConfig set but IsEmpty")
	}
}

// TestDetail02: FillDefaults fills Topology (public DNS) and Channel, and
// fails when the cluster has no name (the error names the field).
func TestDetail02(t *testing.T) {
	c := bbCluster("my.cluster.example.com")
	if err := c.FillDefaults(); err != nil {
		t.Fatalf("FillDefaults: %v", err)
	}
	if c.Spec.Networking.Topology == nil || c.Spec.Networking.Topology.DNS != DNSTypePublic {
		t.Fatalf("topology = %+v, want public DNS", c.Spec.Networking.Topology)
	}
	if c.Spec.Channel != DefaultChannel {
		t.Fatalf("channel = %q, want %q", c.Spec.Channel, DefaultChannel)
	}

	err := bbCluster("").FillDefaults()
	if err == nil {
		t.Fatal("empty cluster name accepted")
	} else if !strings.Contains(err.Error(), "Name") {
		t.Fatalf("error does not name the offending field: %v", err)
	}
}

// TestDetail03: SharedVPC is NetworkID != "".
func TestDetail03(t *testing.T) {
	c := bbCluster("c")
	if c.SharedVPC() {
		t.Fatal("empty NetworkID is shared")
	}
	c.Spec.Networking.NetworkID = "vpc-123"
	if !c.SharedVPC() {
		t.Fatal("NetworkID set but not shared")
	}
}

// TestDetail04: IsKubernetesGTE panics on unparseable input and ignores the
// cluster version's Pre/Build components.
func TestDetail04(t *testing.T) {
	c := bbCluster("c")
	c.Spec.KubernetesVersion = "1.30.0-alpha.1"
	// Pre is stripped from the cluster version, so 1.30.0-alpha.1 satisfies >=1.30.0
	if !c.IsKubernetesGTE("1.30.0") {
		t.Fatal("cluster pre-release not stripped")
	}
	if c.IsKubernetesLT("1.30.0") {
		t.Fatal("IsKubernetesLT is not !IsKubernetesGTE")
	}
	if !c.IsKubernetesLT("1.31.0") {
		t.Fatal("1.30.0 not < 1.31.0")
	}
	if !bbPanicked(func() { c.IsKubernetesGTE("not-a-version") }) {
		t.Fatal("unparseable argument did not panic")
	}
	bad := bbCluster("c")
	bad.Spec.KubernetesVersion = "not-a-version"
	if !bbPanicked(func() { bad.IsKubernetesGTE("1.30.0") }) {
		t.Fatal("unparseable cluster version did not panic")
	}
}

// TestDetail05: Azure name helpers return the spec field when set, else the
// cluster name; the NSG helper prefers Networking.NetworkID.
func TestDetail05(t *testing.T) {
	c := bbCluster("clust.example.com")
	c.Spec.CloudProvider.Azure = &AzureSpec{}

	if c.IsSharedAzureResourceGroup() {
		t.Fatal("empty ResourceGroupName reported shared")
	}
	if got := c.AzureResourceGroupName(); got != "clust.example.com" {
		t.Fatalf("resource group fallback = %q", got)
	}
	c.Spec.CloudProvider.Azure.ResourceGroupName = "rg1"
	if !c.IsSharedAzureResourceGroup() || c.AzureResourceGroupName() != "rg1" {
		t.Fatal("ResourceGroupName not honored")
	}

	if c.IsSharedAzureRouteTable() {
		t.Fatal("empty RouteTableName reported shared")
	}
	if got := c.AzureRouteTableName(); got != "clust.example.com" {
		t.Fatalf("route table fallback = %q", got)
	}
	c.Spec.CloudProvider.Azure.RouteTableName = "rt1"
	if !c.IsSharedAzureRouteTable() || c.AzureRouteTableName() != "rt1" {
		t.Fatal("RouteTableName not honored")
	}

	c.Spec.Networking.NetworkID = "vnet-1"
	if got := c.AzureNetworkSecurityGroupName(); got != "vnet-1" {
		t.Fatalf("NSG should follow the vnet name: %q", got)
	}
	c.Spec.Networking.NetworkID = ""
	if got := c.AzureNetworkSecurityGroupName(); got != "clust.example.com" {
		t.Fatalf("NSG fallback = %q", got)
	}
}

// TestDetail06: DNS topology predicates — nil/empty/public topology means
// public DNS; PublishesDNSRecords and UsesLoadBalancerForKopsController are
// the negation and alias of UsesNoneDNS.
func TestDetail06(t *testing.T) {
	c := bbCluster("c")

	// nil topology
	if !c.UsesPublicDNS() || c.UsesPrivateDNS() || c.UsesNoneDNS() {
		t.Fatal("nil topology should be public")
	}
	c.Spec.Networking.Topology = &TopologySpec{}
	if !c.UsesPublicDNS() {
		t.Fatal("empty DNS type should be public")
	}
	c.Spec.Networking.Topology.DNS = DNSTypePrivate
	if !c.UsesPrivateDNS() || c.UsesPublicDNS() {
		t.Fatal("private DNS misclassified")
	}
	if !c.PublishesDNSRecords() {
		t.Fatal("private DNS should still publish records")
	}
	if c.UsesLoadBalancerForKopsController() {
		t.Fatal("private DNS should not need the LB path")
	}
	c.Spec.Networking.Topology.DNS = DNSTypeNone
	if !c.UsesNoneDNS() {
		t.Fatal("none DNS misclassified")
	}
	if c.PublishesDNSRecords() {
		t.Fatal("none DNS publishes records")
	}
	if !c.UsesLoadBalancerForKopsController() {
		t.Fatal("none DNS should use the LB path")
	}
}

// TestDetail07: InstallCNIAssets — with Cilium only "portmap" chaining counts;
// otherwise it requires no AmazonVPC and no Calico.
func TestDetail07(t *testing.T) {
	c := bbCluster("c")
	if !c.InstallCNIAssets() {
		t.Fatal("plain cluster should install CNI assets")
	}
	c.Spec.Networking.AmazonVPC = &AmazonVPCNetworkingSpec{}
	if c.InstallCNIAssets() {
		t.Fatal("AmazonVPC networking still installs CNI assets")
	}
	c.Spec.Networking.AmazonVPC = nil
	c.Spec.Networking.Calico = &CalicoNetworkingSpec{}
	if c.InstallCNIAssets() {
		t.Fatal("Calico networking still installs CNI assets")
	}
	c.Spec.Networking.Calico = nil
	c.Spec.Networking.Cilium = &CiliumNetworkingSpec{}
	if c.InstallCNIAssets() {
		t.Fatal("Cilium without portmap chaining installs CNI assets")
	}
	c.Spec.Networking.Cilium.ChainingMode = "portmap"
	if !c.InstallCNIAssets() {
		t.Fatal("Cilium portmap chaining should install CNI assets")
	}
	// portmap wins even when a VPC CNI is also configured
	c.Spec.Networking.AmazonVPC = &AmazonVPCNetworkingSpec{}
	if !c.InstallCNIAssets() {
		t.Fatal("Cilium portmap + AmazonVPC should still install")
	}
}

// TestDetail08: HasImageVolumesSupport is a version gate (shape only — the
// threshold is an implementation detail): very old clusters lack it, very new
// clusters have it.
func TestDetail08(t *testing.T) {
	old := bbCluster("c")
	old.Spec.KubernetesVersion = "1.0.0"
	if old.HasImageVolumesSupport() {
		t.Fatal("ancient cluster claims image volume support")
	}
	new := bbCluster("c")
	new.Spec.KubernetesVersion = "9.99.0"
	if !new.HasImageVolumesSupport() {
		t.Fatal("future cluster lacks image volume support")
	}
}

// TestDetail09: APIInternalName is a derived dotted name ending in the cluster
// name (shape only — the internal subdomain spelling is conventional).
func TestDetail09(t *testing.T) {
	c := bbCluster("my.cluster.example.com")
	got := c.APIInternalName()
	if got == "" || got == "my.cluster.example.com" {
		t.Fatalf("APIInternalName = %q", got)
	}
	if !strings.HasSuffix(got, "my.cluster.example.com") {
		t.Fatalf("APIInternalName %q does not end with the cluster name", got)
	}
	if !strings.Contains(got, ".") {
		t.Fatalf("APIInternalName %q is not a dotted name", got)
	}
}

// TestDetail10: IsIPv6Only keys on NonMasqueradeCIDR; IsKopsControllerIPAM
// aliases it; IsCiliumENIIPAM needs Cilium with the ENI IPAM mode.
func TestDetail10(t *testing.T) {
	s := &ClusterSpec{}
	s.Networking.NonMasqueradeCIDR = "fd00:10::/64"
	if !s.IsIPv6Only() || !s.IsKopsControllerIPAM() {
		t.Fatal("v6 NonMasqueradeCIDR not recognised")
	}
	s.Networking.NonMasqueradeCIDR = "100.64.0.0/10"
	if s.IsIPv6Only() || s.IsKopsControllerIPAM() {
		t.Fatal("v4 NonMasqueradeCIDR is IPv6")
	}

	if s.IsCiliumENIIPAM() {
		t.Fatal("no Cilium but ENI IPAM")
	}
	s.Networking.Cilium = &CiliumNetworkingSpec{}
	if s.IsCiliumENIIPAM() {
		t.Fatal("Cilium without IPAM=eni")
	}
	s.Networking.Cilium.IPAM = CiliumIpamEni
	if !s.IsCiliumENIIPAM() {
		t.Fatal("Cilium eni IPAM not detected")
	}
	s.Networking.Cilium.IPAM = "hostscope"
	if s.IsCiliumENIIPAM() {
		t.Fatal("Cilium hostscope is ENI")
	}
}

// TestDetail11: GetCloudProvider — the alpha cloud label overrides, a single
// non-nil provider field yields its id, and none yields "".
func TestDetail11(t *testing.T) {
	c := bbCluster("c")
	if got := c.GetCloudProvider(); got != "" {
		t.Fatalf("empty provider = %q", got)
	}
	c.Spec.CloudProvider.GCE = &GCESpec{}
	if got := c.GetCloudProvider(); got != CloudProviderGCE {
		t.Fatalf("gce provider = %q", got)
	}
	c.Spec.CloudProvider.AWS = &AWSSpec{}
	if got := c.GetCloudProvider(); got != CloudProviderAWS && got != CloudProviderGCE {
		t.Fatalf("multi-provider returned a third provider: %q", got)
	}
	c.Labels = map[string]string{AlphaLabelCloudProvider: "metal"}
	if got := c.GetCloudProvider(); got != CloudProviderMetal {
		t.Fatalf("label override = %q, want metal", got)
	}
}

// TestDetail12: WarmPoolSpec.IsEnabled — nil spec is disabled; an explicit
// MaxSize of 0 disables; nil MaxSize or non-zero enables.
func TestDetail12(t *testing.T) {
	var in *WarmPoolSpec
	if in.IsEnabled() {
		t.Fatal("nil warm pool enabled")
	}
	if !(&WarmPoolSpec{}).IsEnabled() {
		t.Fatal("unset MaxSize should be enabled")
	}
	var zero, five int64 = 0, 5
	if (&WarmPoolSpec{MaxSize: &zero}).IsEnabled() {
		t.Fatal("MaxSize 0 should disable")
	}
	if !(&WarmPoolSpec{MaxSize: &five}).IsEnabled() {
		t.Fatal("MaxSize 5 should enable")
	}
}

// TestDetail13: ResolveDefaults — an IG with no warm pool gets a disabled
// (MaxSize 0) spec for control-plane/bastion roles or when no cluster default
// exists, else the cluster default; an IG with a warm pool wins per-field on
// node roles (unset MaxSize/MinSize/EnableLifecycleHook inherit), and wins
// verbatim on control-plane/bastion.
func TestDetail13(t *testing.T) {
	var five, nine int64 = 5, 9

	nodeIG := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleNode}}
	cpIG := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleControlPlane}}
	bastionIG := &InstanceGroup{Spec: InstanceGroupSpec{Role: InstanceGroupRoleBastion}}

	// no cluster default + no IG warm pool -> disabled spec
	var in *WarmPoolSpec
	got := in.ResolveDefaults(nodeIG)
	if got == nil || got.MaxSize == nil || *got.MaxSize != 0 {
		t.Fatalf("nil default + node no-wp = %+v, want MaxSize 0", got)
	}

	in = &WarmPoolSpec{MaxSize: &five, MinSize: 2}
	got = in.ResolveDefaults(nodeIG)
	if got == nil || got.MaxSize == nil || *got.MaxSize != 5 || got.MinSize != 2 {
		t.Fatalf("node without warm pool should inherit the cluster default: %+v", got)
	}
	got = in.ResolveDefaults(cpIG)
	if got == nil || got.MaxSize == nil || *got.MaxSize != 0 {
		t.Fatalf("control-plane without warm pool = %+v, want MaxSize 0", got)
	}
	got = in.ResolveDefaults(bastionIG)
	if got == nil || got.MaxSize == nil || *got.MaxSize != 0 {
		t.Fatalf("bastion without warm pool = %+v, want MaxSize 0", got)
	}

	// node IG with a warm pool: set fields win, unset fields inherit
	nodeIG.Spec.WarmPool = &WarmPoolSpec{MaxSize: &nine}
	got = in.ResolveDefaults(nodeIG)
	if got == nil || got.MaxSize == nil || *got.MaxSize != 9 {
		t.Fatalf("IG MaxSize should win: %+v", got)
	}
	if got.MinSize != 2 {
		t.Fatalf("MinSize should inherit from cluster default: %+v", got)
	}

	// control-plane IG with a warm pool: verbatim, no merge
	cpIG.Spec.WarmPool = &WarmPoolSpec{MinSize: 7}
	got = in.ResolveDefaults(cpIG)
	if got == nil || got.MaxSize != nil || got.MinSize != 7 {
		t.Fatalf("control-plane warm pool should be verbatim: %+v", got)
	}
}
