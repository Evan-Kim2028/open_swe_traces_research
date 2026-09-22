# Details — clusterpreds

1. The four `IsEmpty` methods check specific pointer/len fields (AuthenticationSpec:
   Kopeio, AWS, OIDC; AuthorizationSpec: RBAC, AlwaysAllow; TargetSpec: Terraform;
   TerraformSpec: both config maps). Inferable: partially — which fields count is a choice.
2. `FillDefaults` sets Topology to `{DNS: DNSTypePublic}` and Channel to `DefaultChannel`,
   and errors `cluster Name not set in FillDefaults` when `ObjectMeta.Name == ""`. Inferable:
   partially — the exact error string is arbitrary.
3. `SharedVPC` is `Networking.NetworkID != ""`. Inferable: yes.
4. `IsKubernetesGTE` parses via `util.ParseKubernetesVersion`, zeroes Pre/Build on the
   CLUSTER version only, and PANICS on unparseable input (no error return). Inferable:
   partially — panic-on-bad-input and Pre/Build stripping are choices.
5. Azure name helpers return the spec field when non-empty else `c.Name`; the NSG helper
   uses `Networking.NetworkID` first, then `c.Name`. Inferable: partially — the NSG→vnet
   fallback ordering is a policy choice.
6. `UsesPublicDNS` is true when Topology is nil OR DNS empty OR `DNSTypePublic`; Private/None
   require the exact DNSType. `PublishesDNSRecords`/`UsesLoadBalancerForKopsController` are
   negations/aliases of `UsesNoneDNS`. Inferable: yes.
7. `InstallCNIAssets`: Cilium → `ChainingMode == "portmap"` only; else
   `AmazonVPC == nil && Calico == nil`. Inferable: partially — the portmap special case.
8. `HasImageVolumesSupport` is `!IsKubernetesLT("1.32.0")`. Inferable: no — the 1.32.0
   threshold is arbitrary (doc comment explains it).
9. `APIInternalName` is `"api.internal." + name`. Inferable: partially — the
   `api.internal.` spelling is conventional but still a literal.
10. `IsIPv6Only` delegates to `utils.IsIPv6CIDR` on `NonMasqueradeCIDR`; `IsKopsControllerIPAM`
    aliases it; `IsCiliumENIIPAM` is `Cilium != nil && IPAM == "eni"`. Inferable: partially.
11. `GetCloudProvider`: the `alpha.clustkit.k8s.io/cloud == "metal"` label overrides, then the
    first non-nil provider in fixed order AWS→Azure→DO→GCE→Hetzner→Openstack→Scaleway→Linode,
    else `""`. Inferable: partially — the label key and ordering are arbitrary.
12. `WarmPoolSpec.IsEnabled` is `in != nil && (MaxSize == nil || *MaxSize != 0)` — an explicit
    MaxSize of 0 DISABLES. Inferable: yes.
13. `ResolveDefaults`: IG with no warm pool → `MaxSize:0` spec for control-plane/bastion or
    when `in == nil`, else `in`; IG with warm pool → IG wins per-field, unset fields
    (MaxSize, MinSize, EnableLifecycleHook) inherit from `in` — except CP/bastion which get
    the IG spec verbatim. Inferable: partially — the merge precedence is a choice.
