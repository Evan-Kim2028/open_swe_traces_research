# Contract — clusterpreds

Cluster and instance-group predicates over the `Cluster`/`ClusterSpec` API
types. Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **`IsEmpty` field checks.** `AuthenticationSpec.IsEmpty` checks Kopeio,
   AWS, and OIDC; `AuthorizationSpec.IsEmpty` checks RBAC and AlwaysAllow;
   `TargetSpec.IsEmpty` checks Terraform; `TerraformSpec.IsEmpty` checks
   ProviderExtraConfig and FilesProviderExtraConfig. Covered by
   `TestDetail01`.
2. **`FillDefaults`.** On a named cluster it fills the networking topology
   with public DNS and sets the channel to the package default; an unnamed
   cluster returns an error that names the offending field. Covered by
   `TestDetail02`.
3. **`SharedVPC`** is true iff the networking `NetworkID` is non-empty.
   Covered by `TestDetail03`.
4. **Version comparisons.** `IsKubernetesGTE`/`IsKubernetesLT` panic on
   unparseable input (either side) and strip Pre/Build components from the
   cluster version before comparing. Covered by `TestDetail04`.
5. **Azure naming.** `IsSharedAzureResourceGroup`/`AzureResourceGroupName`
   and `IsSharedAzureRouteTable`/`AzureRouteTableName` return the spec field
   when set, else the cluster name; `AzureNetworkSecurityGroupName` prefers
   the networking `NetworkID`, else the cluster name. Covered by
   `TestDetail05`.
6. **DNS predicates.** Nil topology, empty DNS type, or `Public` all mean
   `UsesPublicDNS`; `Private`/`None` are exact matches;
   `PublishesDNSRecords` is `!UsesNoneDNS` and
   `UsesLoadBalancerForKopsController` is `UsesNoneDNS`. Covered by
   `TestDetail06`.
7. **`InstallCNIAssets`.** With Cilium configured only `portmap` chaining
   counts; otherwise the cluster must have neither AmazonVPC nor Calico
   networking. Covered by `TestDetail07`.
8. **`HasImageVolumesSupport` (shape).** A Kubernetes-version gate exists:
   a very old version reports no support, a very new version reports
   support. The threshold itself is an implementation detail and is not
   pinned. Covered by `TestDetail08`.
9. **`APIInternalName` (shape).** Returns a derived dotted name that is not
   the bare cluster name but ends with it. The internal subdomain spelling
   is conventional and is not pinned. Covered by `TestDetail09`.
10. **IPv6/IPAM predicates.** `IsIPv6Only` and `IsKopsControllerIPAM` follow
    whether `NonMasqueradeCIDR` is an IPv6 CIDR; `IsCiliumENIIPAM` requires
    Cilium networking with the ENI IPAM mode. Covered by `TestDetail10`.
11. **`GetCloudProvider`.** The alpha cloud-provider label overrides; a
    single non-nil provider field yields its cloud id; no provider yields
    `""`. With multiple providers set, the returned value is one of the
    set providers (ordering is not pinned). Covered by `TestDetail11`.
12. **`WarmPoolSpec.IsEnabled`.** A nil spec is disabled; an explicit
    `MaxSize` of 0 disables; nil or non-zero `MaxSize` enables. Covered by
    `TestDetail12`.
13. **`WarmPoolSpec.ResolveDefaults`.** An instance group with no warm pool
    gets a disabled spec (`MaxSize` 0) when it is a control-plane/bastion
    group or when no cluster default exists, else the cluster default. An
    instance group with a warm pool wins per-field on node groups — unset
    `MaxSize`, `MinSize`, and `EnableLifecycleHook` inherit from the cluster
    default — and wins verbatim on control-plane/bastion groups. Covered by
    `TestDetail13`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — defaults are derivable; error asserted only as naming the field |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | yes |
| TestDetail08 | 8 | no — shape only (version gate: old=no, new=yes) |
| TestDetail09 | 9 | no — shape only (dotted name ending in cluster name) |
| TestDetail10 | 10 | yes |
| TestDetail11 | 11 | yes |
| TestDetail12 | 12 | yes |
| TestDetail13 | 13 | yes |
