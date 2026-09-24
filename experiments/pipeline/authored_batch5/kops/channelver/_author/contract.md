# Contract — channelver

Channel resolution and version recommendation over the `Channel` spec type.
Every commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Channel resolution.** `ResolveChannel` maps the sentinel `"none"` to
   `(nil, nil)`, resolves a relative location under the package's declared
   channel base, passes absolute URLs through unchanged, and returns an error
   that mentions the channel location for unparseable input. Covered by
   `TestDetail01`.
2. **Channel parse.** `ParseChannel` decodes a YAML `Channel` and wraps a
   malformed document in an "error parsing channel" error. Covered by
   `TestDetail02`.
3. **Upgrade gates.** `FindRecommendedUpgrade` and `IsUpgradeRequired` treat
   empty spec fields as "no upgrade" and fire only when the spec version is
   strictly greater than the current one — an equal version is not an
   upgrade. This holds for both the Kubernetes and the kops spec types.
   Covered by `TestDetail03`.
4. **Parser asymmetry.** The Kubernetes-version spec fields accept the
   `/v1.<minor>.` URL form (yielding version `1.<minor>.0`); the kops spec
   fields do not. Covered by `TestDetail04`.
5. **Range lookup.** `FindKubernetesVersionSpec`/`FindKopsVersionSpec` return
   the first spec whose range matches; an empty range matches any version;
   unparseable ranges are skipped rather than fatal. Covered by
   `TestDetail05`.
6. **Image select.** `FindImage` filters by provider and treats empty
   architecture and empty version-range fields as wildcards; the first match
   wins and no match yields nil. Covered by `TestDetail06`.
7. **Recommended version.** `RecommendedKubernetesVersion` follows the kops
   version to the matching spec and returns that spec's `KubernetesVersion`
   field — not its `RecommendedVersion` — and nil on any miss. Covered by
   `TestDetail07`.
8. **Upstream prefix (shape).** `HasUpstreamImagePrefix` answers from the
   image string alone — independent of channel contents — and reports false
   for the empty string and for arbitrary custom-registry images. The exact
   prefix list is an implementation detail and is not pinned. Covered by
   `TestDetail08`.
9. **Package lookup.** `GetPackageVersion` requires exact name equality,
   honors a package's `KubernetesVersion` range when a cluster version is
   supplied, and errors when nothing matches. Covered by `TestDetail09`.

   Note: the source row also claims a nil `kubernetesVersion` skips the range
   check; the reference implementation dereferences the version for ranged
   packages, so that sub-case is intentionally not asserted (flagged in the
   verification writeup).

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — sentinel documented in the source comment; base via the exported var |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially — asserts k8s spec accepts the URL form and kops spec does not |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | yes |
| TestDetail08 | 8 | no — shape only (channel-independent, false for custom images) |
| TestDetail09 | 9 | yes — nil+ranged sub-case not asserted (see note) |
