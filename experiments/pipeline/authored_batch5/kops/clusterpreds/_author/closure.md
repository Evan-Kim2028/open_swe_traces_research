# Closure — clusterpreds

Package: `pkg/apis/kops` (`example.internal/clustkit/pkg/apis/kops`).

Files: `pkg/apis/kops/cluster.go` (27 funcs).

Removed functions (bodies stubbed): `AuthenticationSpec.IsEmpty`, `AuthorizationSpec.IsEmpty`,
`TargetSpec.IsEmpty`, `TerraformSpec.IsEmpty`, `Cluster.FillDefaults`, `Cluster.SharedVPC`,
`Cluster.IsKubernetesGTE`, `Cluster.IsKubernetesLT`, `Cluster.IsSharedAzureResourceGroup`,
`Cluster.AzureResourceGroupName`, `Cluster.IsSharedAzureRouteTable`, `Cluster.AzureRouteTableName`,
`Cluster.AzureNetworkSecurityGroupName`, `Cluster.PublishesDNSRecords`, `Cluster.UsesPublicDNS`,
`Cluster.UsesPrivateDNS`, `Cluster.UsesNoneDNS`, `Cluster.UsesLoadBalancerForKopsController`,
`Cluster.InstallCNIAssets`, `Cluster.HasImageVolumesSupport`, `Cluster.APIInternalName`,
`Cluster.GetCloudProvider`, `ClusterSpec.IsIPv6Only`, `ClusterSpec.IsKopsControllerIPAM`,
`ClusterSpec.IsCiliumENIIPAM`, `WarmPoolSpec.IsEnabled`, `WarmPoolSpec.ResolveDefaults`.

Exported entry point(s): the `Cluster`/`ClusterSpec` predicate family — consumed throughout
`upup/pkg/fi/cloudup`, validation, and nodeup.

Test files removed in excision: `pkg/apis/kops/cluster_test.go` (covers WarmPoolSpec and
InstallCNIAssets).
