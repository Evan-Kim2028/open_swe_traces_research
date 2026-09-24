# Details — channelver

1. `ResolveChannel` special-cases the literal `"none"` to `(nil, nil)`; other relative
   locations resolve against `DefaultChannelBase`, and unparseable input returns
   `invalid channel location: %q`. Inferable: partially — the `"none"` sentinel and the
   default base URL are arbitrary constants.
2. `ParseChannel` delegates to the kept `ParseRawYaml` and wraps the error as
   `error parsing channel %v`. Inferable: yes.
3. `FindRecommendedUpgrade`/`IsUpgradeRequired` return nil/false when the spec field is
   empty, and only fire when the parsed field is STRICTLY greater — an equal version is
   not an upgrade. Inferable: yes.
4. The Kubernetes specs parse versions with `util.ParseKubernetesVersion` (tolerates the
   `/v1.<n>.` URL form) while the Kops specs use `semver.ParseTolerant` — the two variants
   differ on inputs like `"https://.../v1.28.txt"`. Inferable: partially — the asymmetry
   is historical.
5. `FindKubernetesVersionSpec`/`FindKopsVersionSpec` return the FIRST entry whose `Range`
   matches; an empty `Range` matches any version; unparseable ranges are skipped with a
   warning, not an error. Inferable: yes.
6. `FindImage` treats empty `ArchitectureID` and empty `KubernetesVersion` as wildcards
   and returns the first match (warning, not error, on multiple). Inferable: yes.
7. `RecommendedKubernetesVersion` maps kops-version → spec → its `KubernetesVersion`
   field; it ignores `RecommendedVersion`/`Range` of that spec for the result. Inferable: yes.
8. `HasUpstreamImagePrefix` is a fixed prefix list including `kope.io/k8s-`,
   `099720109477/ubuntu/images/hvm-ssd*/ubuntu-{focal,jammy,noble}-*`, `cos-cloud/cos-stable-`,
   `ubuntu-os-cloud/ubuntu-*` and `Canonical:...` forms. Inferable: no — the exact prefix
   list is arbitrary.
9. `GetPackageVersion` requires name equality; a nil `kubernetesVersion` skips the range
   check; no match → non-nil error. Inferable: yes.
