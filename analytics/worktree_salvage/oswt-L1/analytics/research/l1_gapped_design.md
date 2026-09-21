# L1 gapped contract: per-detail causal test

Eight units failed 0/3 at L0 and passed at L2 (seven of them 3/3; assetsremap 2/3).
Every L0 miss was a single behavioural commitment. L1 holds the L2 contract
constant and deletes exactly one invariant, so the L0→L2 gain is no longer
"more prose helped" but "did *this* sentence matter?"

Packaged at
`/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_L1/`
(16 dirs: `<repo>-<unit>-L1binding` and `-L1other`). Harbor trials are not
launched from this session.

Reproduce:

```
uv run python scripts/build_l1_gapped.py --json
# preflight from the closure-K worktree (this branch does not ship pipeline/preflight.py):
cd /home/evan/Documents/oswt-closureK
uv run python scripts/preflight_task.py --json \
  /home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_L1/*-L1*
```

## How L1 is built

`build_affordance_levels(..., levels=(-1,), name_scheme="L")` is still the
packager. Two extensions in `synth/affordance.py`:

1. `omit_invariant(instruction, row_contains=..., prose=...)` deletes exactly
   one coverage-table row and/or a verbatim body sentence. A-1 previously took
   an arbitrary instruction string; it could not name *which* invariant to drop.
2. `variant=` suffixes the dest dir (`family-L1binding`). `rewrite_test_sh=False`
   keeps the L2 `tests/test.sh` so the hidden harness is not regenerated.

Source for each unit is the preflight-PASS L2 dir under
`oswt-closureK/experiments/pipeline/tasks_composerver/<repo>/<unit>-L2`.
The gapped instruction is that L2 `instruction.md` minus one commitment.
`task.toml` is the CURSOR allowlist (`cursor.com`, `*.cursor.com`, `*.cursor.sh`,
`downloads.cursor.com`). Gold/cheat patches and hidden tests are copies of L2.

## Hidden-test checksums

sha256 over sorted `(relpath, bytes)` under `tests/hidden/`. All four levels
match per unit. If they had not, the L0/L1/L2 comparison would be invalid.

| unit | sha256 | L0=L1=L2 |
|---|---|---|
| gin/jsonrenders | `ce3b3b7086a411ca3ae3d1b3f16d82daf1a1c1f03853f44ff3cddde94dc911cb` | yes |
| gin/streamrenders | `9776d43df2a59a5648678dc2c648ae67ebbce324c6681654286acc35d63b599f` | yes |
| kops/addonparse | `a131e62c9ceeb3237c8223362c142d4c024cdba7e0c1b089e17854a4e87aafb6` | yes |
| kops/issuecert | `ec55aaef93477b2760cadc7e435c41f815819e0102872b26aba7303754748c53` | yes |
| kops/memfs | `cbfb5435989ae54f45aa0efe7cb19c39cfbdf23c982042dacbf3fc3a3f70b6b9` | yes |
| kops/oidcdisc | `88573c3f394afb81d5743e467a8905cbc5ec43227eb9b9fe83eddf56d0919c62` | yes |
| kops/templater | `6ea4c49c1373603417a691dc4a0ca9fe91f616fb485680e6cdcf33b5ea37d83d` | yes |
| kops/assetsremap | `79ff2d8527564601aeb3e912c8c03af0a86a23c151660664e778edafa221eeb0` | yes |

`tests/test.sh` is byte-identical to the unit's L2 harness.

## Per unit

L0 evidence is from `experiments/dose_response/sweep_L0_audit.tsv` (three
attempts each). Binding = the commitment those attempts missed. Other = a
commitment the same attempts already got right.

### gin/jsonrenders

- L0 0/3, L2 3/3. Fail: `TestJsonpJSONCallbackProperty` / jsonp wrap mismatch (3/3).
- Binding deletes `| TestRenderJsonpJSON(+Error,+Error2,+Fail) | callback escaping, '(' json ');' wrapping, empty-callback passthrough, javascript content type |` and the JsonpJSON body sentence (content type, empty-callback passthrough, `callback(` json `);`).
- Other deletes `| TestRenderAsciiJSON(+Fail) | non-ASCII runes become \uXXXX escapes; charset-less content type |` and the AsciiJSON body sentence.

### gin/streamrenders

- L0 0/3, L2 3/3. Fail: `TestDataContentLengthProperty` / empty data must not set Content-Length (3/3).
- Binding deletes `| TestRenderData/TestRenderDataContentLength/TestRenderDataError | bytes written with Content-Length only when non-empty |` and "Data writes Content-Length only when the payload is non-empty and then writes the bytes."
- Other deletes `| TestRenderRedirect | status-code validation panic outside 3xx/201 and normal redirect otherwise |` and the Redirect body sentence.

### kops/addonparse

- L0 0/3, L2 3/3. Fail: `TestAddonParseEmptyInvalidProperty` line 191 `case 2` (3/3).
- Empty-input and invalid-YAML checks run *before* that loop and would have
  printed `empty %q` or `invalid yaml`. They did not. Go `rand` seed 20260919:
  i=0,1 have pad=0 (pass); i=2 has two leading spaces on a `kind: Addons` doc.
  Binding is "Parse trims the buffer", not the empty-list-for-empty-input row.
- Binding deletes `| TestParseAddons | kind Addons document parses |` (nearest
  coverage row; trim is body-only) and "Parse trims the buffer and loads YAML objects."
- Other deletes `| TestParseAddonsEmpty | empty input is not an error |` and
  "Empty/whitespace/comment-only input yields an empty addons list (not an error)."

### kops/issuecert

- L0 0/3, L2 3/3. Fail: `TestIssuecertClientServerProperty` / IP SAN: [] (3/3).
- Binding deletes `| TestIssueCert (clientServer) | DNS + IP SANs; both client and server EKU |` and "; each alternate name is trimmed, IPs vs DNS split, empties skipped".
- Other deletes `| TestIssueCert (clientOneYear) | explicit validity window |` and "Optional Validity sets NotAfter from now (UTC)."

### kops/memfs

- L0 0/3, L2 3/3. Fail: `TestMemFsCreateWriteProperty` / create: file already exists (2/3); one attempt failed `TestMemFsReadDirProperty` instead.
- Binding is exclusive create (the majority miss). Deletes `| TestMemFsCreateFile | exclusive create vs overwrite write; exists on second create |` and "Create is exclusive: a node that already has contents returns exists; Write always replaces contents."
- Other deletes `| TestMemFsReadTree | recursive leaves only |` and the tree-listing sentence. Not ReadDir, because one L0 attempt failed that.

### kops/oidcdisc

- L0 0/3, L2 3/3. Fail: `TestOIDCMemoryStoreListProperty` / get missing (2/3); one attempt failed `TestOIDCDiscoveryDocumentProperty` (no oidc spec status 200).
- Binding is Get-missing / list-unknown-universe. The coverage table has no row
  for that; the hidden suite comment maps "List of an unknown universe is empty"
  onto this property. Body-only omit: "List of an unknown universe is empty, not
  an error. Get of a missing object returns nil, nil."
- Other deletes `| TestDiscoveryIsolation | objects in one universe are invisible in another |` and the universes-do-not-leak paragraph. Not the 404-when-no-OIDC-spec row (that failed 1/3 at L0).

### kops/templater

- L0 0/3, L2 3/3. Fail: `TestTemplaterSnippetNameProperty` / snippet named mainTemplate must be rejected (3/3).
- Binding is body-only: "Each snippet is parsed under its filename; colliding with `mainTemplate` is an error." The coverage table has no reserved-name row. The hidden suite maps "snippet mainTemplate rejected" onto this property.
- Other deletes `| TestRenderIndent | indent skips first/empty lines |` and the Indent body sentence.

### kops/assetsremap

- L0 0/3, L2 2/3. Fail: `TestAssetsRemapRegistryConvergeProperty` / case 0 oracle (3/3 at L0; the L2 miss was the same assertion).
- Binding deletes `| TestValidate_RemapImage_ContainerRegistry_MappingMultipleTimesConverges | second registry pass does not double-prefix |` and "A second pass must not double-prefix (spec assembly calls this until the cluster spec converges)."
- Other deletes `| TestRemapURLPathDelimiterEscaping | commas in file paths are %2C |` and "; commas in the escaped path become `%2C`".

## Preflight (in-image, closure-K)

Gate: bare FAIL with assertions, gold PASS, cheat FAIL. Ran from
`oswt-closureK` via `scripts/preflight_task.py`. Instruction text is not in
the image, so binding and other should share a verdict with L0/L2.

| dir | bare | gold | cheat | seconds | verdict |
|---|---|---|---|---|---|
| gin-jsonrenders-L1binding | fail | pass | fail | 16 | PASS |
| gin-jsonrenders-L1other | fail | pass | fail | 17 | PASS |
| gin-streamrenders-L1binding | fail | pass | fail | 15 | PASS |
| gin-streamrenders-L1other | fail | pass | fail | 17 | PASS |
| kops-addonparse-L1binding | fail | pass | fail | 78 | PASS |
| kops-addonparse-L1other | fail | pass | fail | 60 | PASS |
| kops-issuecert-L1binding | fail | pass | fail | 40 | PASS |
| kops-issuecert-L1other | fail | pass | fail | 25 | PASS |
| kops-memfs-L1binding | fail | pass | fail | 94 | PASS |
| kops-memfs-L1other | fail | pass | fail | 117 | PASS |
| kops-oidcdisc-L1binding | fail | pass | fail | 74 | PASS |
| kops-oidcdisc-L1other | fail | pass | fail | 39 | PASS |
| kops-templater-L1binding | fail | pass | fail | 76 | PASS |
| kops-templater-L1other | fail | pass | fail | 59 | PASS |
| kops-assetsremap-L1binding | fail | pass | fail | 79 | PASS |
| kops-assetsremap-L1other | fail | pass | fail | 75 | PASS |

16/16 PASS. 881 s wall. Binding and other share `tests_sha256` and `tree_sha256` per unit (instruction is not in the image).

## Pre-registered predictions

Fill `observed` after Composer trials (parent session). Do not peek.

| unit | L0 | L2 | pred binding | pred other | observed binding | observed other |
|---|---|---|---|---|---|---|
| gin/jsonrenders | 0/3 | 3/3 | FAIL | PASS |  |  |
| gin/streamrenders | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/addonparse | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/issuecert | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/memfs | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/oidcdisc | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/templater | 0/3 | 3/3 | FAIL | PASS |  |  |
| kops/assetsremap | 0/3 | 2/3 | FAIL | PASS |  |  |

If both variants pass, detail-completeness is not the mechanism and the L0→L2 gain is something else. If both fail, the contract as a whole is load-bearing rather than any single row.

Deleted text is also in each dir's `gap.json`. Mapping lives in
`src/openswe_traces/synth/l1_gapped.py` (`UNITS`).
