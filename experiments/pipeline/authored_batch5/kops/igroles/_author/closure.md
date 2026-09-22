# Closure — igroles

Package: `pkg/apis/kops` (`example.internal/clustkit/pkg/apis/kops`).

Files: `pkg/apis/kops/instancegroup.go` (23 funcs).

Removed functions (bodies stubbed): `InstanceGroupRole.HasControlPlane`, `HasNode`, `HasBastion`,
`HasAPIServer`, `HasEtcd`, `HasScheduler`, `HasKubeControllerManager`, `IsControlPlaneType`,
`ToLowerString`; `InstanceGroup.IsControlPlane`, `IsAPIServerOnly`, `RunsAPIServer`, `IsEtcdOnly`,
`RunsEtcd`, `IsSchedulerOnly`, `RunsScheduler`, `IsKubeControllerManagerOnly`,
`RunsKubeControllerManager`, `HasGVisor`, `IsBastion`, `IsKarpenterManaged`,
`AddInstanceGroupNodeLabel`.

Exported entry point(s): the role predicate family — consumed by validation, nodeup builders and
`pkg/nodelabels`; `ParseInstanceGroupRole` (kept, in `parse.go`) calls `ToLowerString`.

Test files removed in excision: `pkg/apis/kops/cluster_test.go` (reaches `HasControlPlane` via
`WarmPoolSpec.ResolveDefaults`), `pkg/apis/kops/parse_test.go` (reaches `ToLowerString` via
`ParseInstanceGroupRole`).
