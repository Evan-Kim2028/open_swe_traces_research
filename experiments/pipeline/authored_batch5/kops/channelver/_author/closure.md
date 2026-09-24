# Closure — channelver

Package: `pkg/apis/kops` (`example.internal/clustkit/pkg/apis/kops`).

Files: `pkg/apis/kops/channel.go` (12 funcs; `LoadChannel` stays — it is the VFS-bound caller).

Removed functions (bodies stubbed): `ResolveChannel`, `ParseChannel`,
`KubernetesVersionSpec.FindRecommendedUpgrade`, `KopsVersionSpec.FindRecommendedUpgrade`,
`KubernetesVersionSpec.IsUpgradeRequired`, `KopsVersionSpec.IsUpgradeRequired`,
`FindKubernetesVersionSpec`, `FindKopsVersionSpec`, `Channel.FindImage`,
`RecommendedKubernetesVersion`, `Channel.HasUpstreamImagePrefix`, `Channel.GetPackageVersion`.

Exported entry point(s): `ResolveChannel`/`ParseChannel`/`RecommendedKubernetesVersion` — used by
`kops create`/`upgrade` flows and by `LoadChannel` (kept).
