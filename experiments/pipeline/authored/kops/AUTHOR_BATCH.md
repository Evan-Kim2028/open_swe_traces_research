# AUTHOR_BATCH — kops (clustkit-obf) feature-excision units, hardest first

Base tree: `experiments/pipeline/repos/kops/src` (identity-obfuscated kops, module `example.internal/clustkit`). Base image `ladder-base:kops`. All units sit in packages whose tests pass OFFLINE per `repos/kops/test.jsonl` (`ok` lines).
Each unit lives at `experiments/pipeline/authored/kops/<unit>/_author/` with `closure.md`, `api.md`, `contract.md` (coverage table over the repo's own tests), `bugreport.md`, `gold.patch`, `cheat.patch`, `excised/` (tree + `excision.patch`), `difficulty.md`. Excision = function bodies replaced by `panic("excised: <name>")` stubs (signatures/doc comments kept, unused imports blanked, compiles clean, fails at runtime). Gold restores exact bodies. Cheat patches replace panics with zero-value returns. No tests written.

| # | unit | files | funcs | ~lines excised | hardness driver |
|---|------|-------|-------|---------------:|-----------------|
| 1 | `clustervalid` | 2 | 9 | ~402 | multi-pass cluster validation: detached/warm-pool/bastion exemptions, Ready∧¬NetworkUnavailable, max-unready applied to nodes and pods, control-plane static-pod set |
| 2 | `assetsremap` | 1 | 11 | ~318 | hub-vs-host proxy rewrite, idempotent registry flatten, concurrent mutex snapshots, success-only hash cache |
| 3 | `oidcdisc` | 3 | 12 | ~290 | per-universe isolation, OIDC document, JWKS kid-merge by LastSeen, SSA name/namespace vs client cert |
| 4 | `addonparse` | 3 | 8 | ~149 | YAML first-object wrap, location-hash synthetic name, semver filter, replace(id/hash/generation) |
| 5 | `flagbuilder` | 1 | 5 | ~200 | reflection over flag tags: repeat vs join, duration `0`→`0s`, quantity decimal, quote-if-`"` |
| 6 | `issuecert` | 4 | 5 | ~210 | type-alias issuance, self-signed vs keystore, SAN IP vs DNS, default serial/validity/usages, PEM parse |
| 7 | `memfs` | 1 | 11 | ~104 | exclusive create, leaf-only ReadTree, nested mutex Join |
| 8 | `osmetadata` | 1 | 6 | ~106 | ordered config-drive vs HTTP fallback, last error wins |
| 9 | `tomlwriter` | 1 | 9 | ~167 | byte-stable TOML: sorted scalars-before-tables, go-toml v1 scalar walk, quoting |
| 10 | `templater` | 2 | 5 | ~180 | sprig+indent+include+channel funcs; missingkey switch |

## Notes

- Packages confirmed `ok` offline: `pkg/validation`, `pkg/assets`, `discovery/pkg/discovery`, `channels/pkg/channels`, `pkg/flagbuilder`, `pkg/pki`, `util/pkg/vfs`, `upup/pkg/fi/cloudup/openstack/openstackmetadata`, `pkg/tomlwriter`, `pkg/util/templater`.
- `templater` is the batch control (`control: true` in its difficulty.md).
- predicted_flip: clustervalid L4, assetsremap L4, oidcdisc L4, addonparse L3, flagbuilder L3, issuecert L3, memfs L3, osmetadata L3, tomlwriter L2, templater L2.
- Gold/cheat/excision patches touch only non-`*_test.go` files (A12).
