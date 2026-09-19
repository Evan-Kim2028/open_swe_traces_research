# VERIFIER_BATCH.md — composerver kops

Verifier batch for clustkit/kops feature-excision units.
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:kops`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_kops_batch.py --max-parallel 2
```

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `clustervalid` | 5 | 83.3 | yes | 0 | 0.5 | PASS |
| `assetsremap` | 5 | 90.9 | yes | 0 | 0.5 | PASS |
| `oidcdisc` | 5 | 100.0 | yes | 0 | 0.7 | PASS |
| `addonparse` | 5 | 90.9 | yes | 0 | 0.9 | PASS |
| `flagbuilder` | 5 | 100.0 | yes | 0 | 0.6 | PASS |
| `issuecert` | 5 | 100.0 | yes | 0 | 0.4 | PASS |
| `memfs` | 5 | 100.0 | yes | 0 | 0.8 | PASS |
| `osmetadata` | 4 | 88.9 | yes | 0 | 0.6 | PASS |
| `tomlwriter` | 6 | 60.0 | yes | 0 | 0.7 | PASS |
| `templater` | 8 | 100.0 | yes | 0 | 0.6 | PASS |

**Batch totals:** 10/10 PASS

---

## `clustervalid`

- **Properties:** 5
- **Contract coverage:** 83.3%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.5

- **L0:** `experiments/pipeline/tasks_composerver/kops/clustervalid-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/clustervalid-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/clustervalid-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/clustervalid-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| instance group absent from the cloud is a failure | `TestClusterValidContractTableProperty` |
| non-detached size below target is a failure | `TestClusterValidContractTableProperty` |
| detached members do not count toward size | `TestClusterValidConstructorProperty` |
| detached members are not required to join or be ready | `TestClusterValidConstructorProperty` |
| unready worker/node role is a failure | `TestClusterValidNodesProperty` |
| control-plane group undersized vs target | `TestClusterValidNodesProperty` |
| unready control-plane node is a failure | `TestClusterValidBastionProperty` |
| missing kube-apiserver/controller-manager/scheduler on a ready control-plane node | `TestClusterValidBastionProperty` |
| healthy critical pods produce no pod failures | `TestClusterValidToleratedUnreadyProperty` |
| pending/unknown/unready critical pods are failures | `TestClusterValidToleratedUnreadyProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `assetsremap`

- **Properties:** 5
- **Contract coverage:** 90.9%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.5

- **L0:** `experiments/pipeline/tasks_composerver/kops/assetsremap-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/assetsremap-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/assetsremap-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/assetsremap-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| hub image (no registry host) gets proxy prepended | `TestAssetsRemapImageContractTableProperty` |
| org/name hub form also prepends | `TestAssetsRemapImageContractTableProperty` |
| dotted first segment is a host and is replaced | `TestAssetsRemapRegistryConvergeProperty` |
| legacy k8s registry host is replaced | `TestAssetsRemapRegistryConvergeProperty` |
| tags survive proxy rewrite | `TestAssetsRemapFileContractTableProperty` |
| second registry pass does not double-prefix | `TestAssetsRemapFileContractTableProperty` |
| commas in file paths are `%2C` | `TestAssetsRemapManifestProperty` |
| empty YAML section does not panic | `TestAssetsRemapManifestProperty` |
| concurrent remap + sorted snapshot getters | `TestAssetsRemapSortedSnapshotsProperty` |
| successful hash downloads are cached by resolved URL | `TestAssetsRemapSortedSnapshotsProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `oidcdisc`

- **Properties:** 5
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.7

- **L0:** `experiments/pipeline/tasks_composerver/kops/oidcdisc-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/oidcdisc-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/oidcdisc-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/oidcdisc-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| objects in one universe are invisible in another | `TestOIDCMemoryStoreListProperty` |
| well-known document issuer/jwks_uri and 404 when no OIDC spec | `TestOIDCDiscoveryIsolationProperty` |
| JWKS merge by kid with LastSeen winner | `TestOIDCDiscoveryDocumentProperty` |
| apply/create identity and namespace checks; listed object is the applied one | `TestOIDCJWKSMergeProperty` |
| apply/create identity and namespace checks; listed object is the applied one | `TestOIDCDiscoveryAuthProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `addonparse`

- **Properties:** 5
- **Contract coverage:** 90.9%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.9

- **L0:** `experiments/pipeline/tasks_composerver/kops/addonparse-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/addonparse-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/addonparse-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/addonparse-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| kind Addons document parses | `TestAddonParseContractTableProperty` |
| first-object Addons wins in a multi-doc stream | `TestAddonParseContractTableProperty` |
| non-Addons YAML is wrapped as a synthetic addon | `TestAddonParseEmptyInvalidProperty` |
| synthetic name is manifest- plus 12 hex of location hash | `TestAddonParseEmptyInvalidProperty` |
| invalid YAML is an error | `TestAddonGetCurrentVersionProperty` |
| empty input is not an error | `TestAddonGetCurrentVersionProperty` |
| kubernetes version range selects the matching addon | `TestAddonReplacementProperty` |
| id/hash/generation decide which duplicate wins | `TestAddonReplacementProperty` |
| nil vs non-nil update from existing version + replace | `TestAddonRequiredUpdatesProperty` |
| (uses GetCurrent/version) rolling when replace is true | `TestAddonRequiredUpdatesProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `flagbuilder`

- **Properties:** 5
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.6

- **L0:** `experiments/pipeline/tasks_composerver/kops/flagbuilder-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/flagbuilder-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/flagbuilder-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/flagbuilder-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| durations, ints, nested structs, omit-empty | `TestFlagbuilderKCMProperty` |
| kubelet tags including maps/slices | `TestFlagbuilderKubeletProperty` |
| admission slices and repeat vs join | `TestFlagbuilderAPIServerProperty` |
| quote only when the joined-string form sees `"` | `TestFlagbuilderQuotingProperty` |
| quote only when the joined-string form sees `"` | `TestFlagbuilderSortedRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `issuecert`

- **Properties:** 5
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.4

- **L0:** `experiments/pipeline/tasks_composerver/kops/issuecert-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/issuecert-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/issuecert-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/issuecert-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| PEM private key parse + signed cert from existing key | `TestIssuecertCAProperty` |
| certificate PEM round-trip | `TestIssuecertClientProperty` |
| private key PEM round-trip | `TestIssuecertClientServerProperty` |
| private key PEM round-trip | `TestIssuecertServerProperty` |
| private key PEM round-trip | `TestIssuecertPEMRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `memfs`

- **Properties:** 5
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.8

- **L0:** `experiments/pipeline/tasks_composerver/kops/memfs-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/memfs-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/memfs-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/memfs-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| exclusive create vs overwrite write; exists on second create | `TestMemFsCreateWriteProperty` |
| immediate children after joins/writes | `TestMemFsReadDirProperty` |
| recursive leaves only | `TestMemFsReadTreeProperty` |
| recursive leaves only | `TestMemFsRemoveProperty` |
| recursive leaves only | `TestMemFsUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `osmetadata`

- **Properties:** 4
- **Contract coverage:** 88.9%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.6

- **L0:** `experiments/pipeline/tasks_composerver/kops/osmetadata-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/osmetadata-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/osmetadata-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/osmetadata-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| non-200 HTTP is an error | `TestOsmetadataDefaultSearchOrderProperty` |
| 200 body decodes | `TestOsmetadataDefaultSearchOrderProperty` |
| missing config-2 device/blkid fails | `TestOsmetadataGetLocalProperty` |
| successful mount+read | `TestOsmetadataGetLocalProperty` |
| last error after all sources fail | `TestOsmetadataJSONRoundTripProperty` |
| first source wins | `TestOsmetadataJSONRoundTripProperty` |
| fallback to HTTP | `TestOsmetadataJSONRandomProperty` |
| HTTP-first order | `TestOsmetadataJSONRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `tomlwriter`

- **Properties:** 6
- **Contract coverage:** 60.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.7

- **L0:** `experiments/pipeline/tasks_composerver/kops/tomlwriter-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/tomlwriter-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/tomlwriter-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/tomlwriter-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| empty document is empty string | `TestTomlEmptyAndRootTableProperty` |
| scalars before subtables, sorted keys | `TestTomlQuotingContractProperty` |
| table header at root | `TestTomlSetPathBehaviorProperty` |
| quoting and escape sequences | `TestTomlSetPathUnsupportedTypeProperty` |
| already-quoted keys not re-escaped | `TestTomlScalarsOrderProperty` |
| empty key is `""` | `TestTomlUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `templater`

- **Properties:** 8
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.6

- **L0:** `experiments/pipeline/tasks_composerver/kops/templater-L0`
- **L2:** `experiments/pipeline/tasks_composerver/kops/templater-L2`
- **L5:** `experiments/pipeline/tasks_composerver/kops/templater-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/kops/templater-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| basic interpolation | `TestTemplaterRenderContextProperty` |
| failOnMissing errors | `TestTemplaterFailOnMissingProperty` |
| failOnMissing false allows missing | `TestTemplaterAllowMissingProperty` |
| indent skips first/empty lines | `TestTemplaterIndentProperty` |
| include named snippet | `TestTemplaterIncludeProperty` |
| context map fields | `TestTemplaterSnippetNameProperty` |
| channel recommended version/image funcs | `TestTemplaterIncludeMissingProperty` |
| combined snippets + funcs | `TestTemplaterUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

