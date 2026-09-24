# Bug report

Instance group role checks are broken: control-plane instance groups no longer report that they
run the apiserver, etcd, scheduler or kube-controller-manager, `ToLowerString` no longer produces
`control-plane`, gVisor node groups are not detected, Karpenter-managed groups are not detected,
and the `kops.k8s.io/instancegroup` node label is never added.

Expected: role predicates compare `Spec.Role` against the matching constant; control-plane groups
report running all four control-plane daemons; `ToLowerString` yields `control-plane`,
`apiserver`, `node`, etc.; `HasGVisor` is true only for Node groups with
`containerd.gVisor.enabled: true`; `IsKarpenterManaged` requires a Karpenter manager on a Node
group; `AddInstanceGroupNodeLabel` writes the group's name under its well-known label key.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
