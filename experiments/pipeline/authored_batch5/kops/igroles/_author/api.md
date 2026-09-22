# Exported API — igroles

Package `pkg/apis/kops` (importable as `example.internal/clustkit/pkg/apis/kops`).

`InstanceGroupRole` is a string enum (`ControlPlane`, `APIServer`, `Node`, `Bastion`, `Etcd`,
`Scheduler`, `KubeControllerManager`).

- `func (r InstanceGroupRole) HasControlPlane() bool` — true iff `r == InstanceGroupRoleControlPlane`.
  Same for `HasNode`/`HasBastion`/`HasAPIServer`/`HasEtcd`/`HasScheduler`/`HasKubeControllerManager`.
- `func (r InstanceGroupRole) IsControlPlaneType() bool` — true for ControlPlane, APIServer, Etcd,
  Scheduler or KubeControllerManager; false for Node and Bastion.
- `func (r InstanceGroupRole) ToLowerString() string` — `"control-plane"` for the control-plane role,
  otherwise the lowercased role string (`"node"`, `"apiserver"`, `"etcd"`, ...).
- `func (g *InstanceGroup) IsControlPlane() bool`, `IsAPIServerOnly`, `IsEtcdOnly`,
  `IsSchedulerOnly`, `IsKubeControllerManagerOnly`, `IsBastion` — role equality on `g.Spec.Role`.
- `func (g *InstanceGroup) RunsAPIServer() bool` — `IsControlPlane() || IsAPIServerOnly()`.
  Same shape for `RunsEtcd`, `RunsScheduler`, `RunsKubeControllerManager`.
- `func (g *InstanceGroup) HasGVisor() bool` — true only for Node role with
  `Spec.Containerd.GVisor.Enabled == true`.
- `func (g *InstanceGroup) IsKarpenterManaged() bool` — `Spec.Manager == InstanceManagerKarpenter`
  and role is Node.
- `func (g *InstanceGroup) AddInstanceGroupNodeLabel()` — sets
  `Spec.NodeLabels["kops.k8s.io/instancegroup"] = g.Name`, allocating the map if nil.

Worked examples: `InstanceGroupRoleControlPlane.ToLowerString()` → `"control-plane"`;
`InstanceGroupRoleNode.IsControlPlaneType()` → `false`; a ControlPlane group `RunsEtcd()` → `true`;
a Node group `RunsAPIServer()` → `false`.
