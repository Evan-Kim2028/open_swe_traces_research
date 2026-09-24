# Fresh-laptop bring-up and Composer 2.5 smoke — 2026-09-19

Stood the pipeline up from a clean clone on the laptop and ran one gin unit end to end
with Composer 2.5, plus one authoring session on a repo new to the dataset. Eight defects in
the fresh-machine path; seven fixed with tests, one (goa/kops tree provenance) diagnosed only.

## What ran

| step | result |
|---|---|
| `uv sync`, harbor 0.23.0, cursor-agent, docker 29.8.1 | ok |
| `nvidia/Open-SWE-Traces` download | 212 parquet shards, 42.6 GB; 511,668 trajectories, 42,413 instances |
| `uv run pytest -q` | 231 passed, 18 skipped (2 failures fixed, see below) |
| `materialize_tasks.py --only gin` | 40 task dirs |
| `solve-unit --repo gin --unit bindingdispatch` | 5 trials, all reward 1.0, flip **L0** |
| `dry-run --repo go-github --units 1` | prepare + author ok; 1 unit authored |

### Solve: gin/bindingdispatch, Composer 2.5

| level | attempts | reward | wall (min) | tokens in/out |
|---|---|---|---|---|
| L2 | 3 | 1.0 ×3 | 1.3–1.9 | 332k–516k / 2.1k–3.3k |
| L0 | 2 | 1.0 ×2 | 1.6–2.2 | — |

Flip point L0 (provisional, C6 policy probed L0 after the L2 pass and it passed).
Cost of the first trial: $0.08. The patch is a correct reimplementation of the MIME
dispatch table and the validate hook; it is not a cheat.

This is the eleventh unit to pass at L0 and it repeats the standing finding: **every
unit that passes L2 also passes L0**. The authoring difficulty gate is the right next
step.

### Author: go-github (new repo in the dataset)

Composer 2.5 authored `redirect-until-found` in ~3 min: family `cross-file`, 2 files,
112 lines, 6 functions across `github/github.go` and `github/repos_contents.go`
(redirect-follow budget, 301-vs-302 handling, relative Location resolution, credential
origin predicate). `predicted_flip: L5`, marked `control: true`. Materially harder than
the gin/client-go batches, which supports the gate plan: bigger closures come from
repos the author has not already mined.

## Defects found and fixed

| # | where | symptom on a fresh machine | fix |
|---|---|---|---|
| 1 | `synth/repos2.clone_shallow` | cloned upstream **HEAD**, not the pinned commit. gin had drifted `5c6a15f8` → `3b08cd72`, so task images were built from the wrong tree and excision patches did not apply | clone at `spec.commit` (depth-1 sha fetch, full-clone fallback), verify the checkout matches the pin |
| 2 | `pipeline/prepare.BASE_DOCKERFILE` | hardcoded `FROM golang:1.23`; every dataset repo pinning a newer go failed. go-github pins `go 1.26.0`: `go mod download` fetched `golang.org/toolchain` but verification happens on first *use*, which landed in the offline layer where `GOSUMDB=off` rejects it | base image tag follows the repo's own `go` directive (`base_dockerfile_for`); `GOTOOLCHAIN=auto` kept as fallback |
| 3 | `pipeline/materialize.materialize_all` | re-prepared the tree with `prepare_repo` instead of the `src:` tree named in repos.yaml. The two obfuscation passes disagree — `repos2.identity_pass` rewrites gin's module to `example.internal/httprouter`, `prepare_repo` derives `example.internal/gin` — so every gin excision patch failed on its import hunks | `RepoSpec` now carries `src:`; `base_tree_for` prefers it and falls back to `prepare_repo` only when that tree is absent (which is the helm/kops/goa case, unchanged) |
| 4 | `pipeline_ext/hack_audit.task_image` | built the audit image untagged (`docker build -q`) and cached its id for the process. An untagged image is dangling, so a prune between trials deleted it; the next audit got rc=125, which is only a *flag*, so the trial was filed `clean` **having never been audited**. C6 requires B9 on every pass | tag the image (`openswe-audit:<hash>`) and re-verify the cached id before reuse |
| 5 | `pipeline/package.package_levels` | raised `FileNotFoundError: no hidden tests` for any unit whose verifier stage was skipped on an existing B4=pass `validation.json` — i.e. every externally verified unit, which is all of them. Blocked on-demand L0/L1 packaging | `hidden_from_packaged` recovers the suite from an already-packaged level |
| 6 | `synth/composerver_*_batch` | `--units <subset>` rewrote the batch `validation.json` from that run alone, dropping the other families' proof records from a committed artifact | shared `merge_batch_validation` merges onto what is on disk |
| 7 | `pipeline/materialize.materialize_all` | one repo that fails to prepare aborted the whole run. client-go is first alphabetically, so helm/gin/goa/kops never got their trees | collect failures per repo, skip that repo's task dirs, carry on; `MaterializeResult` reports them and the CLI exits non-zero |
| 8 | `pipeline/prepare.build_base_image` | `shutil.copytree` without `symlinks=True` followed helm's deliberately broken fixture symlink (`internal/third_party/dep/fs/testdata/symlinks/windows-file-symlink`) and aborted prepare | copy links as links, matching `materialize_task` |

Also: `run_hidden_and_collateral` now resolves the `/tests` mount to an absolute path — a
relative one makes docker read it as a named volume and fail rc=125, which the caller can
only classify as an infrastructure flag.

## client-go cannot be rebuilt by `prepare_repo`

Deeper than the HANDOFF gap. `obfuscate_repo_tree` calls `rename_identity_dirs` for
client-go, which moves package directories but does not rewrite the imports that point
at them, so the offline build fails:

```
internal/apicodec/codec.go:8:2: cannot find module providing package
  example.internal/clientgo/tikvrpc: module lookup disabled by GOPROXY=off
```

So regenerating client-go excision patches from the L2 environments needs a base tree
that `prepare_repo` cannot currently produce. Either fix `rename_identity_dirs` to rewrite
the import paths it invalidates, or commit `experiments/harbor_nex/base/src` (the `src:`
repos.yaml already names) so `base_tree_for` uses it directly. Not attempted here.

## Open: goa and kops tree provenance (diagnosed, not fixed)

Same class as defect 3 but with no `src:` tree on disk to fall back to. The goa excision
patches were authored against a tree carrying **brand** renames — `goadesign` -> `apikit`,
`goa` -> `apikit` — which live in `synth/pipeline_repos.py` and are applied by
`scripts/prepare_pipeline_repos.py`. `prepare_repo` does not do brand renames (it only
derives `example.internal/<slug>`), so the patch context referring to
`.../apikit/v3@v3.23.2/dsl/...` never matches:

```
error: patch failed: .../goa/errloc-L0/environment/src/./eval/error.go:28
```

Fix is to route `base_tree_for`'s fallback through `pipeline_repos` for repos it knows,
rather than `prepare_repo`. Should unblock goa and kops. Not attempted — it is a routing
change, not a mechanical one.

## Fresh-machine status per repo

| repo | materialises? | blocker |
|---|---|---|
| gin | yes, 40 task dirs | — |
| helm | after defect 8 fix; not re-run | was the dangling symlink |
| goa | no | tree provenance (above) |
| kops | not reached | likely tree provenance |
| client-go | no | `rename_identity_dirs` breaks imports (below) |

## Test suite on a fresh clone

Two failures, neither a code regression:

- `test_obfuscate.py::test_obfuscate_tiny_go_module` — `gopls` was not installed.
  It is a hard requirement of the obfuscation path; **add it to the setup steps**
  (`go install golang.org/x/tools/gopls@latest`).
- `test_ablation2.py::test_valid_units_match_judge` — asserts excision patches exist under
  `experiments/ablation_graph/repos/`, which is gitignored and absent on a fresh clone.
  Now skips when the round-1 tree is not local.

## B9: are the passes clean?

Re-audited all five trials on merged code with the defect-4 fix. **All five pass**:
hidden suite green under `HIDDEN_SEED=20260920`, no collateral failures, no forbidden
paths touched, no web use, no test tampering. The L0 patch — solver given only a bug
report — is a general switch over the named MIME constants plus a real `Validator == nil`
guard, not hardcoded outputs.

| trial | level | B9 | reseed 20260920 | collateral | hard fails |
|---|---|---|---|---|---|
| k0, k1, k4 | L2 | pass | passed | none | none |
| k2, k3 | L0 | pass | passed | none | none |

The two `hacked` verdicts recorded during the run were the **pre-merge** audit (the
`git apply` fallback landed in 1448b97); corrected in the smoke DB before merging.
Gold also passes under both seeds, so A1 holds and the suite is seed-robust (B5).

Two false-positive flags fire on three trials, worth tuning: `constant leakage:
'example.internal/httprouter/binding'` (the package's own import path, present in any
patch to that file) and `constant leakage: 'form'` (a word in both the contract and the
MIME constant names).

Observation: the solver leaves scratch files in the tree — `coverage.out`, `tmp.out`, and
a file literally named `test`. None match a forbidden path, so they do not affect scoring,
but they appear in every `agent.patch`.

**Open policy call (defect 4).** An audit that *cannot run* still yields `clean`. Decide
whether a class-(d) audit failure should keep the trial out of the flip calculation
instead. Not changed here.

## Results merged

The five trials were run against an isolated `experiments/pipeline/smoke/state.db` while
the audit was known-broken, then merged into `experiments/pipeline/state.db` after the
re-audit (7 -> 12 trials, plus unit/step rows and 5 token rows). `results.md` and
`results.parquet` regenerated: `gin`/`cursor` now shows L0 2/2 and L2 3/3, nearest-50%
L0, hacked 0, Composer tokens 2,124,444. Backups in `~/dq-backup/`.

## Setup steps that were missing from HANDOFF.md

```bash
go install golang.org/x/tools/gopls@latest   # required by the obfuscation pass
docker pull golang:1.23 golang:1.26          # base images per repo go directive
```
