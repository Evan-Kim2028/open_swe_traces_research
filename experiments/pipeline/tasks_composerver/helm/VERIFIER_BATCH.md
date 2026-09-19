# VERIFIER_BATCH.md — composerver helm

Composer verifier batch for helm/chartkit feature-excision units.
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:helm`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_helm_batch.py --max-parallel 2
```

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `ignorerules` | 4 | 80.0 | yes | 0 | 0.1 | PASS |
| `strvalsparser` | 13 | 100.0 | yes | 0 | 0.1 | PASS |
| `kindsorter` | 4 | 100.0 | yes | 0 | 0.2 | PASS |
| `memorydriver` | 6 | 85.7 | yes | 0 | 0.4 | PASS |
| `storage` | 7 | 53.8 | yes | 0 | 0.1 | PASS |
| `coalesce` | 7 | 70.0 | yes | 0 | 0.1 | PASS |
| `chartloader` | 7 | 87.5 | yes | 0 | 0.2 | PASS |
| `depresolver` | 5 | 100.0 | yes | 0 | 0.2 | PASS |
| `repindex` | 7 | 93.3 | yes | 0 | 0.3 | PASS |
| `provenance` | 6 | 100.0 | yes | 0 | 0.2 | PASS |

**Batch totals:** 10/10 PASS

---

## `ignorerules`

- **Properties:** 4
- **Contract coverage:** 80.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/helm/ignorerules-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/ignorerules-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/ignorerules-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/ignorerules-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| rule file parsing | `TestIgnoreContractTableProperty` |
| malformed patterns error | `TestIgnoreParseAndDefaultsProperty` |
| load rules from file | `TestIgnoreOracleAgreementProperty` |
| match semantics incl. dir-only and ** | `TestIgnoreUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `strvalsparser`

- **Properties:** 13
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/helm/strvalsparser-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/strvalsparser-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/strvalsparser-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/strvalsparser-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| no panic on arbitrary input | `TestStrvalsContractTableProperty` |
| literal mode keeps strings | `TestStrvalsNoPanicProperty` |
| literal merge into dest | `TestStrvalsParseSimpleProperty` |
| nested level cap in literal mode | `TestStrvalsListGrowProperty` |
| list grow/index assignment | `TestStrvalsStringModeProperty` |
| set parsing incl. lists/escapes | `TestStrvalsLiteralProperty` |
| merge into existing map | `TestStrvalsParseIntoProperty` |
| string mode: no inference | `TestStrvalsLiteralIntoProperty` |
| JSON value mode | `TestStrvalsJSONProperty` |
| values via rune reader | `TestStrvalsFileModeProperty` |
| file mode into dest | `TestStrvalsToYAMLProperty` |
| yaml rendering | `TestStrvalsNestedLevelProperty` |
| nested level cap | `TestStrvalsUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `kindsorter`

- **Properties:** 4
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/helm/kindsorter-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/kindsorter-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/kindsorter-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/kindsorter-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| kind ordering applied | `TestKSContractTableProperty` |
| stable within kind | `TestKSContractTableProperty` |
| namespace vs unknown kinds | `TestKSContractTableProperty` |
| hooks weight+kind order | `TestKSReleaseSortsProperty` |
| partition hooks from manifests | `TestKSReleaseSortsProperty` |
| multi-doc split | `TestKSReleaseSortsProperty` |
| name sort with revision tiebreak | `TestKSOracleAgreementProperty` |
| date sort | `TestKSOracleAgreementProperty` |
| revision sort | `TestKSOracleAgreementProperty` |
| reverse name | `TestKSMalformedAndEmptyProperty` |
| reverse date | `TestKSMalformedAndEmptyProperty` |
| reverse revision | `TestKSMalformedAndEmptyProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `memorydriver`

- **Properties:** 6
- **Contract coverage:** 85.7%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.4

- **L0:** `experiments/pipeline/tasks_composerver/helm/memorydriver-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/memorydriver-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/memorydriver-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/memorydriver-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| driver reports its name | `TestMemoryDriverContractTableProperty` |
| create stores retrievable revision | `TestMemoryDriverContractTableProperty` |
| get by key or not-found | `TestMemoryDriverCRUDProperty` |
| list yields newest per name | `TestMemoryDriverCRUDProperty` |
| label match on newest revisions | `TestMemoryDriverListQueryProperty` |
| update replaces in place | `TestMemoryDriverListQueryProperty` |
| delete removes and returns | `TestMemoryDriverAdversarialProperty` |
| sorted insert + duplicate rejection | `TestMemoryDriverAdversarialProperty` |
| remove by key | `TestMemoryDriverUnseenRandomProperty` |
| remove by index | `TestMemoryDriverUnseenRandomProperty` |
| record lookup by key | `TestMemoryDriverConcurrentProperty` |
| index of key in sorted order | `TestMemoryDriverConcurrentProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `storage`

- **Properties:** 7
- **Contract coverage:** 53.8%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/helm/storage-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/storage-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/storage-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/storage-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| create persists under computed key | `TestStorageContractTableProperty` |
| update persists new revision | `TestStorageHistoryProperty` |
| delete removes version | `TestStorageMaxHistoryProperty` |
| list newest per name | `TestStorageDeployedFiltersProperty` |
| deployed lookup or error | `TestStorageAdversarialProperty` |
| corrupt entries skipped | `TestStorageUnseenRandomProperty` |
| all revisions of a name | `TestStoragePruneFailureProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `coalesce`

- **Properties:** 7
- **Contract coverage:** 70.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.1

- **L0:** `experiments/pipeline/tasks_composerver/helm/coalesce-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/coalesce-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/coalesce-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/coalesce-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| coalesce user over defaults | `TestCoalesceContractTableProperty` |
| merge mode wholesale | `TestCoalesceTablesProperty` |
| table-level coalesce | `TestMergeTablesProperty` |
| table-level merge | `TestCoalesceValuesProperty` |
| type-mismatch warnings | `TestMergeValuesProperty` |
| nil cleanup in empty maps | `TestCoalesceAdversarialProperty` |
| subchart nil defaults cleaned | `TestCoalesceUnseenRandomProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `chartloader`

- **Properties:** 7
- **Contract coverage:** 87.5%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/helm/chartloader-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/chartloader-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/chartloader-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/chartloader-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| directory load produces a complete chart | `TestCLContractTable` |
| oversize charts rejected | `TestCLContractTable` |
| device files skipped | `TestCLMergeMapsProperty` |
| symlinks not followed | `TestCLMergeMapsProperty` |
| BOM handling on disk fixtures | `TestCLLoadValuesProperty` |
| BOM stripped in dir load | `TestCLLoadValuesProperty` |
| BOM stripped in archive load | `TestCLLoadFilesProperty` |
| single .tgz file load | `TestCLLoadFilesProperty` |
| explicit buffered-file list load | `TestCLLoaderEquivalence` |
| file order preserved | `TestCLLoaderEquivalence` |
| backslash paths rejected | `TestCLBOMProperty` |
| dependencies nested under charts/ | `TestCLBOMProperty` |
| truncated archive errors | `TestCLAdversarialPaths` |
| values stream parsed | `TestCLAdversarialPaths` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `depresolver`

- **Properties:** 5
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/helm/depresolver-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/depresolver-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/depresolver-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/depresolver-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| resolution across oci/file/repo/alias paths | `TestDRContractTable` |
| lock hash stability and difference | `TestDRResolveScenarios` |
| local path resolution and errors | `TestDRHashReqStable` |
| local path resolution and errors | `TestDRHashV2Stable` |
| local path resolution and errors | `TestDRGetLocalPathProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `repindex`

- **Properties:** 7
- **Contract coverage:** 93.3%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.3

- **L0:** `experiments/pipeline/tasks_composerver/helm/repindex-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/repindex-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/repindex-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/repindex-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| end-to-end index add/get/sort | `TestRIContractTable` |
| yaml index load | `TestRIContractTable` |
| duplicate entries deduped | `TestRIAddSort` |
| empty entry tolerated | `TestRIAddSort` |
| empty index tolerated | `TestRIGet` |
| annotations preserved | `TestRIGet` |
| entries sorted on load | `TestRIMerge` |
| indexes merge without dupes | `TestRIMerge` |
| directory scanned for archives | `TestRIIndexDirectory` |
| add joins url under base | `TestRIIndexDirectory` |
| yaml write round-trip | `TestRILoadRoundTrip` |
| json write round-trip | `TestRILoadRoundTrip` |
| nil-safe add | `TestRILoadMalformed` |
| range detection | `TestRILoadMalformed` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

## `provenance`

- **Properties:** 6
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.2

- **L0:** `experiments/pipeline/tasks_composerver/helm/provenance-L0`
- **L2:** `experiments/pipeline/tasks_composerver/helm/provenance-L2`
- **L5:** `experiments/pipeline/tasks_composerver/helm/provenance-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/helm/provenance-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| message block construction | `TestProvContractTable` |
| sums+metadata extracted | `TestProvContractTable` |
| secret key load | `TestProvContractTable` |
| keyring load | `TestProvDigestProperty` |
| keybox ring load | `TestProvDigestProperty` |
| mixed ring load | `TestProvDigestProperty` |
| armored ring load | `TestProvDigestFileProperty` |
| multi-block armor | `TestProvDigestFileProperty` |
| non-key blocks rejected | `TestProvDigestFileProperty` |
| stream digest | `TestProvParseMessageBlockProperty` |
| signatory from key files | `TestProvParseMessageBlockProperty` |
| file digest | `TestProvParseMessageBlockProperty` |
| passphrase decrypt | `TestProvSignVerifyRoundTrip` |
| clearsign output | `TestProvSignVerifyRoundTrip` |
| sign+verify round trip | `TestProvSignVerifyRoundTrip` |
| signing errors | `TestProvVerifyProperty` |
| verification status | `TestProvVerifyProperty` |
| verify via keybox | `TestProvVerifyProperty` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

