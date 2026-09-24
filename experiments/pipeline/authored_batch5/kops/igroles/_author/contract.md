# Contract — igroles

`InstanceGroupRole` predicates and `InstanceGroup` role helpers in
`pkg/apis/kops`. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Role predicates.** Each `HasX()` is plain equality against its
   `InstanceGroupRoleX` constant — no aliasing, no case folding. Covered by
   `TestDetail01`.
2. **`IsControlPlaneType`** covers ControlPlane, APIServer, Etcd, Scheduler,
   and KubeControllerManager — not Node or Bastion. Covered by
   `TestDetail02`.
3. **`ToLowerString`** yields `control-plane` for the control-plane role and
   the lowercased role string otherwise (`apiserver`, `node`, `etcd`,
   `scheduler`, `kubecontrollermanager`). Covered by `TestDetail03`.
4. **Daemon predicates.** `RunsAPIServer`/`RunsEtcd`/`RunsScheduler`/
   `RunsKubeControllerManager` are `IsControlPlane() || IsXOnly()` — a
   control-plane group runs all four; only groups are limited to their own
   daemon. Covered by `TestDetail04`.
5. **Single-role equality.** `IsControlPlane`/`IsAPIServerOnly`/`IsBastion`
   are exact matches; an empty role matches nothing. Covered by
   `TestDetail05`.
6. **`HasGVisor`** requires role Node AND `Spec.Containerd.GVisor.Enabled`
   non-nil and true; every missing level is false, not a panic. Covered by
   `TestDetail06`.
7. **`IsKarpenterManaged`** requires `Spec.Manager == InstanceManagerKarpenter`
   AND role Node — a Karpenter control-plane group is false. Covered by
   `TestDetail07`.
8. **`AddInstanceGroupNodeLabel`** writes the instancegroup label key with
   the group's name, allocating `Spec.NodeLabels` when nil and preserving
   existing labels. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — membership asserted per api.md |
| TestDetail03 | 3 | partially — control-plane special case per api.md |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | partially — conjunction asserted per api.md |
| TestDetail07 | 7 | partially — conjunction asserted per api.md |
| TestDetail08 | 8 | no — key asserted per api.md literal; shape = alloc + value |
