# Bank expansion: client-go + goa — repair, materialise, preflight

2026-09-19/20 · branch `closure-S` · log `outputs/closure_S.log` · audit `.audit/bank-expansion-clientgo-goa.tsv`

Result: **all 20 authored units materialised and all 40 task dirs (L0 + L2)
preflight-PASS** (bare=fail on assertions, gold=pass, cheat=fail). Zero
exclusions. Bank grows from 30 → **50 units**.

## Materialise routes

Two blocking defects before any unit could package:

- **goa — rebrand.** Authored artifacts were written against the
  identity-pass tree (`example.internal/apikit/v3`, brand `Apikit`); the
  base tree is `example.internal/goa`. Pure naming drift — every hunk's
  context lines matched after rebranding. Fixed with an ordered
  word-boundary `rebrand:` map in `repos.yaml` (same mechanism K used for
  helm/kops), applied to excision/gold/cheat patches, hidden tests,
  `tests/test.sh`, `instruction.md`, plus a hard post-check that fails
  materialisation if any old-module token survives in a packaged artifact.
- **client-go — pinned `src:`.** The authored base is the identity-pass
  tree `example.internal/kvstore/v2` (dir renames tikv→kvclient,
  tikvrpc→wirerpc, mocktikv→mockkv *with* imports remapped).
  `prepare_repo`'s own client-go pass is unusable — it renames dirs
  without remapping imports, so nothing builds. `repos.yaml` now pins
  `src: experiments/pipeline/repos/client-go/src` and the image
  `ladder-base:client-go-obf` was built from that tree. No rebrand map —
  authored artifacts already match the base identity.

Patch-level repair (context drift, not naming): 5 hunks across 4 client-go
units overran EOF by one trailing blank line (authored files ended `}\n\n`,
base ends `}\n`). Fixed by `trim_trailing_blank_overruns()` +
`scripts/repair_authored_patches.py`, verified with `patch --fuzz=0` —
the closure-A route.

| repo | unit | materialise route |
|------|------|-------------------|
| goa | all 10 | rebrand |
| client-go | connarray, doactionbatches, memdbstaging, onregionerror, pdoracle, replicaselector | as-is |
| client-go | lockresolver, pessimisticlock, rangetask, regionstoresorted | trim (EOF blank overrun) |

## Packaging defects found by the gate, then fixed

Same classes K hit on helm/kops:

1. **Excision left imports unused (client-go, 7 units).** The excision
   removed the only uses of an import but kept the import line →
   `imported and not used`, classified `infra`. Repaired by blanking the
   import in the excision (`_ "path"`) and adding a mirrored *restore*
   hunk to gold (so gold re-enables the import); cheat's old-side context
   rewritten to match. Verified fuzz=0 before writing.
   30 import lines total: connarray 8, lockresolver 2, memdbstaging 1,
   onregionerror 2, pdoracle 2, pessimisticlock 10, rangetask 5.
   (`scripts/blank_unused_imports.py` — the prior version had an inverted
   blank/restore direction and a regex that missed the plain
   `"path" imported and not used` form.)
2. **Duplicate injected helper (goa httpxray).** The excision added
   `func _keepExcisedImports()` in three files of one package →
   redeclaration. Renamed per file (`_keepExcisedImportsWrapDoer`, …),
   mirrored in gold/cheat.
3. **pdoracle missed by the first repair batch** — caught by the full
   sweep re-showing `bare=infra`; repaired and re-verified.

## Per-unit preflight results

Gate = `scripts/preflight_task.py` over `openswe_traces.gate` (O's
consolidated package): bare ×2, gold ×2, cheat ×1, network-denied, cached
by (image, tests sha, tree sha). Every row below is a fresh execution —
`cached: false` in `validation.json`.

| repo | unit | level | bare | gold | cheat | sec |
|------|------|-------|------|------|-------|-----|
| goa | errloc | L0 | fail | pass | fail | 39.2 |
| goa | errloc | L2 | fail | pass | fail | 30.5 |
| goa | evalctx | L0 | fail | pass | fail | 35.0 |
| goa | evalctx | L2 | fail | pass | fail | 31.4 |
| goa | evalrun | L0 | fail | pass | fail | 35.9 |
| goa | evalrun | L2 | fail | pass | fail | 28.3 |
| goa | grpcerr | L0 | fail | pass | fail | 35.2 |
| goa | grpcerr | L2 | fail | pass | fail | 26.8 |
| goa | grpctrace | L0 | fail | pass | fail | 23.1 |
| goa | grpctrace | L2 | fail | pass | fail | 22.8 |
| goa | grpcxray | L0 | fail | pass | fail | 31.0 |
| goa | grpcxray | L2 | fail | pass | fail | 22.3 |
| goa | httpxray | L0 | fail | pass | fail | 26.7 |
| goa | httpxray | L2 | fail | pass | fail | 23.7 |
| goa | jsonrpcwire | L0 | fail | pass | fail | 25.6 |
| goa | jsonrpcwire | L2 | fail | pass | fail | 21.9 |
| goa | pkgvalidation | L0 | fail | pass | fail | 21.9 |
| goa | pkgvalidation | L2 | fail | pass | fail | 18.3 |
| goa | xrayseg | L0 | fail | pass | fail | 20.1 |
| goa | xrayseg | L2 | fail | pass | fail | 18.4 |
| client-go | connarray | L0 | fail | pass | fail | 50.1 |
| client-go | connarray | L2 | fail | pass | fail | 55.8 |
| client-go | doactionbatches | L0 | fail | pass | fail | 35.1 |
| client-go | doactionbatches | L2 | fail | pass | fail | 36.0 |
| client-go | lockresolver | L0 | fail | pass | fail | 116.1 |
| client-go | lockresolver | L2 | fail | pass | fail | 112.4 |
| client-go | memdbstaging | L0 | fail | pass | fail | 33.3 |
| client-go | memdbstaging | L2 | fail | pass | fail | 32.5 |
| client-go | onregionerror | L0 | fail | pass | fail | 123.3 |
| client-go | onregionerror | L2 | fail | pass | fail | 116.0 |
| client-go | pdoracle | L0 | fail | pass | fail | 45.9 |
| client-go | pdoracle | L2 | fail | pass | fail | 47.7 |
| client-go | pessimisticlock | L0 | fail | pass | fail | 162.7 |
| client-go | pessimisticlock | L2 | fail | pass | fail | 148.0 |
| client-go | rangetask | L0 | fail | pass | fail | 44.3 |
| client-go | rangetask | L2 | fail | pass | fail | 36.8 |
| client-go | regionstoresorted | L0 | fail | pass | fail | 27.7 |
| client-go | regionstoresorted | L2 | fail | pass | fail | 25.6 |
| client-go | replicaselector | L0 | fail | pass | fail | 114.2 |
| client-go | replicaselector | L2 | fail | pass | fail | 112.9 |

Bare failures are assertion-level in every row (`FAIL\t<pkg>` with test
assertion evidence in `validation.json`); no setup/build/import
(`infra`) verdicts remain.

## Excluded units

None. All 20 authored units reached preflight-PASS at both levels.

## New bank total (preflight-PASS units)

| repo | units |
|------|-------|
| helm | 10 |
| kops | 10 |
| gin | 10 |
| goa | 10 |
| client-go | 10 |
| **total** | **50** |

(L5/L6 task dirs were also materialised for both repos but are not part
of the gate count here.)

## What a third repo needs, in config terms

Given a repo authored under the same `_author/` layout
(`excised/excision.patch`, `gold.patch`, `cheat.patch`, `tests/`,
`instruction.md`):

1. `repos.yaml` entry with **`src:` pinned to the exact tree the authored
   artifacts were written against** (normally the identity-pass tree).
   Do not rely on `prepare_repo` output: verify `go build ./...` passes in
   that tree and that dir renames came with import remapping — client-go's
   generated pass failed exactly this.
2. **`base_image:`** prebuilt `ladder-base:<repo>` FROM that `src:` tree.
3. **`rebrand:`** ordered, word-boundary map whenever authored brand ≠
   base brand. Order longest-first: `module/vN` before the bare module,
   PascalCase brand before lowercase. The post-check then hard-fails if
   any old token survives in `tests/`, `patches/`, `instruction.md`,
   `task.toml`, `affordance.json`, `validation.json`.
4. **Patch hygiene the gate will enforce**: excision must not leave
   unused imports (blank them in the excision and mirror a restore hunk
   in gold); no trailing-blank EOF overruns (fixable via the trim route);
   injected package-level decls must be unique per package.

## Staged screening

`experiments/dose_response/sweep_L0_batch2/<repo>-<unit>-L0` — 20 dirs,
full task packages with the CURSOR host allowlist already in `task.toml`
(`cursor.com`, `*.cursor.com`, `*.cursor.sh`, `downloads.cursor.com`).
Staged only; screening not launched.

## Side fix

`src/openswe_traces/data.py:56` — `parse_shard` now resolves `root`
before `relative_to`, fixing 5 corpus tests broken by the symlinked
`traces_data` in this worktree. `uv run pytest tests` = 276 passed,
18 skipped.
