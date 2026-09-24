# Exported API — clusterpreds

Package `pkg/apis/kops` (importable as `example.internal/clustkit/pkg/apis/kops`).

Predicates and small derivations over `Cluster`/`ClusterSpec` and nested spec types.

- `func (s *AuthenticationSpec) IsEmpty() bool` — all of Kopeio/AWS/OIDC nil.
- `func (s *AuthorizationSpec) IsEmpty() bool` — RBAC and AlwaysAllow nil.
- `func (t *TargetSpec) IsEmpty() bool` — Terraform nil.
- `func (t *TerraformSpec) IsEmpty() bool` — both ProviderExtraConfig and
  FilesProviderExtraConfig empty.
- `func (c *Cluster) FillDefaults() error` — fills `Spec.Networking.Topology` (public DNS)
  and `Spec.Channel` (DefaultChannel) when unset; errors when `ObjectMeta.Name` is empty.
- `func (c *Cluster) SharedVPC() bool` — `Spec.Networking.NetworkID != ""`.
- `func (c *Cluster) IsKubernetesGTE(version string) bool` / `IsKubernetesLT` — parses the
  cluster's `Spec.KubernetesVersion` and the argument with `util.ParseKubernetesVersion`,
  ignores Pre/Build, compares; panics on unparseable input. `LT` is `!GTE`.
- `func (c *Cluster) IsSharedAzureResourceGroup() bool` / `AzureResourceGroupName()` /
  `IsSharedAzureRouteTable()` / `AzureRouteTableName()` / `AzureNetworkSecurityGroupName()` —
  non-empty spec field → shared; the `Name` methods fall back to `c.Name` (NSG falls back to
  `Networking.NetworkID`, then `c.Name`).
- `func (c *Cluster) UsesPublicDNS() bool` — topology nil, empty or `DNSTypePublic`;
  `UsesPrivateDNS`/`UsesNoneDNS` check the matching `DNSType`;
  `PublishesDNSRecords = !UsesNoneDNS`; `UsesLoadBalancerForKopsController = UsesNoneDNS`.
- `func (c *Cluster) InstallCNIAssets() bool` — with Cilium: `ChainingMode == "portmap"`;
  otherwise `AmazonVPC == nil && Calico == nil`.
- `func (c *Cluster) HasImageVolumesSupport() bool` — `!IsKubernetesLT("1.32.0")`.
- `func (c *Cluster) APIInternalName() string` — `"api.internal." + ObjectMeta.Name`.
- `func (c *ClusterSpec) IsIPv6Only() bool` — `NonMasqueradeCIDR` parses as an IPv6 CIDR;
  `IsKopsControllerIPAM = IsIPv6Only`;
  `IsCiliumENIIPAM` = Cilium set and `IPAM == "eni"`.
- `func (c *Cluster) GetCloudProvider() CloudProviderID` — label
  `alpha.clustkit.k8s.io/cloud == "metal"` wins; else first non-nil provider field in the
  order AWS, Azure, DO, GCE, Hetzner, Openstack, Scaleway, Linode; `""` if none.
- `func (in *WarmPoolSpec) IsEnabled() bool` — non-nil and (MaxSize nil or non-zero).
- `func (in *WarmPoolSpec) ResolveDefaults(ig *InstanceGroup) *WarmPoolSpec` — merges the
  cluster-level default `in` with `ig.Spec.WarmPool`; when the IG has no warm pool it returns
  `&WarmPoolSpec{MaxSize: &zero}` for control-plane/bastion roles or no-cluster-default,
  else the cluster default; when the IG has one, its fields win and unset fields
  (MaxSize, MinSize, EnableLifecycleHook) inherit from `in` — except on control-plane/bastion
  IGs where the IG spec is returned unmerged.
