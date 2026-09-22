# Exported API — channelver

Package `pkg/apis/kops` (importable as `example.internal/clustkit/pkg/apis/kops`).

Channel resolution and version-recommendation logic over the `Channel` spec type.

- `func ResolveChannel(location string) (*url.URL, error)` — `"none"` returns `(nil, nil)`;
  a relative location resolves against `DefaultChannelBase`; absolute URLs pass through.
  Example: `ResolveChannel("stable")` → `https://raw.githubusercontent.com/kubernetes/clustkit/master/channels/stable`.
- `func ParseChannel(channelBytes []byte) (*Channel, error)` — YAML-decodes a `Channel`.
- `func (v *KubernetesVersionSpec) FindRecommendedUpgrade(version semver.Version) (*semver.Version, error)` —
  returns the parsed `RecommendedVersion` only when it is strictly greater than `version`;
  nil when empty or not newer. `KopsVersionSpec.FindRecommendedUpgrade` is the same shape.
- `func (v *KubernetesVersionSpec) IsUpgradeRequired(version semver.Version) (bool, error)` —
  true iff parsed `RequiredVersion` strictly exceeds `version`. Same for `KopsVersionSpec`.
- `func FindKubernetesVersionSpec(versions []KubernetesVersionSpec, version semver.Version) *KubernetesVersionSpec` —
  first entry whose `Range` matches (empty `Range` matches everything); nil if none.
  `FindKopsVersionSpec` is identical over `[]KopsVersionSpec`.
- `func (c *Channel) FindImage(provider CloudProviderID, kubernetesVersion semver.Version, architecture architectures.Architecture) *ChannelImageSpec` —
  filters `Spec.Images` by provider, by `ArchitectureID` (empty = wildcard) and by
  `KubernetesVersion` semver range (empty = wildcard); returns the first match, nil if none.
- `func RecommendedKubernetesVersion(c *Channel, kopsVersionString string) *semver.Version` —
  looks up the kops-version spec matching `kopsVersionString` and returns its parsed
  `KubernetesVersion`; nil on any miss.
- `func (c *Channel) HasUpstreamImagePrefix(image string) bool` — true when the image name
  starts with a known upstream prefix (`kope.io/k8s-`, the `099720109477` Ubuntu AMI paths,
  `cos-cloud/cos-stable-`, `ubuntu-os-cloud/...`, `Canonical:...`).
- `func (c *Channel) GetPackageVersion(name string, kubernetesVersion *semver.Version) (*util.Version, error)` —
  first `Spec.Packages` entry matching `name` and (when `kubernetesVersion` non-nil) its
  `KubernetesVersion` range; error when no package matches.
