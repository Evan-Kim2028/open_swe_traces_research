# Bug report

Cluster spec helpers are broken: spec emptiness checks report wrong results, `FillDefaults`
leaves topology/channel unset and never errors on a missing name, `IsKubernetesGTE`/`LT` compare
wrong (or never panic on bad versions), Azure resource-group/route-table/NSG names don't fall
back to the cluster name, DNS topology predicates misreport public/private/none, CNI asset
installation and image-volume support are decided incorrectly, `APIInternalName` is wrong, the
cloud provider is not derived from the spec (and the alpha `metal` label is ignored), and warm
pool enablement/default merging is wrong.

Expected: the predicates inspect the documented spec fields; `IsKubernetesGTE` panics on
unparseable input and ignores Pre/Build; `ResolveDefaults` merges IG-level warm pool fields over
cluster defaults with the control-plane/bastion carve-out.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
