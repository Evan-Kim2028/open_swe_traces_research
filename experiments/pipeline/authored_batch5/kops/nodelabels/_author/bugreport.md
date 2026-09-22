# Bug report

Node-label building is broken: `BuildNodeLabels` doesn't dispatch on instance-group role
(errors on valid roles or omits the role label), kubelet `NodeLabels` aren't merged from
cluster/IG config, the apiserver label misses its featureflag gate, mandatory control-plane
labels aren't applied, and user `Spec.NodeLabels` don't override.

Expected: role-ordered dispatch with error on unhandled role; ControlPlaneKubelet/Kubelet
merge with IG override; `node-role` labels for node/apiserver/control-plane IGs; mandatory
control-plane labels; spec labels last.

Reproduce with:

```
go test -count=1 ./pkg/nodelabels/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
