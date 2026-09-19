# Fresh-laptop bring-up and Composer 2.5 smoke — 2026-09-19

Stood the pipeline up from a clean clone on the laptop and ran one gin unit end to end
with Composer 2.5, plus one authoring session on a repo new to the bank. Six defects in
the fresh-machine path; all fixed, each with a test.

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

### Author: go-github (new repo in the bank)

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
| 2 | `pipeline/prepare.BASE_DOCKERFILE` | hardcoded `FROM golang:1.23`; every bank repo pinning a newer go failed. go-github pins `go 1.26.0`: `go mod download` fetched `golang.org/toolchain` but verification happens on first *use*, which landed in the offline layer where `GOSUMDB=off` rejects it | base image tag follows the repo's own `go` directive (`base_dockerfile_for`); `GOTOOLCHAIN=auto` kept as fallback |
| 3 | `pipeline/materialize.materialize_all` | re-prepared the tree with `prepare_repo` instead of the `src:` tree named in repos.yaml. The two obfuscation passes disagree — `repos2.identity_pass` rewrites gin's module to `example.internal/httprouter`, `prepare_repo` derives `example.internal/gin` — so every gin excision patch failed on its import hunks | `RepoSpec` now carries `src:`; `base_tree_for` prefers it and falls back to `prepare_repo` only when that tree is absent (which is the helm/kops/goa case, unchanged) |
| 4 | `pipeline_ext/hack_audit.task_image` | built the audit image untagged (`docker build -q`) and cached its id for the process. An untagged image is dangling, so a prune between trials deleted it; the next audit got rc=125, which is only a *flag*, so the trial was filed `clean` **having never been audited**. C6 requires B9 on every pass | tag the image (`openswe-audit:<hash>`) and re-verify the cached id before reuse |
| 5 | `pipeline/package.package_levels` | raised `FileNotFoundError: no hidden tests` for any unit whose verifier stage was skipped on an existing B4=pass `validation.json` — i.e. every externally verified unit, which is all of them. Blocked on-demand L0/L1 packaging | `hidden_from_packaged` recovers the suite from an already-packaged level |
| 6 | `synth/composerver_*_batch` | `--units <subset>` rewrote the batch `validation.json` from that run alone, dropping the other families' proof records from a committed artifact | shared `merge_batch_validation` merges onto what is on disk |

| 7 | `pipeline/materialize.materialize_all` | one repo that fails to prepare aborted the whole run. client-go is first alphabetically, so helm/gin/goa/kops never got their trees | collect failures per repo, skip that repo's task dirs, carry on; `MaterializeResult` reports them and the CLI exits non-zero |

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

## Test suite on a fresh clone

Two failures, neither a code regression:

- `test_obfuscate.py::test_obfuscate_tiny_go_module` — `gopls` was not installed.
  It is a hard requirement of the obfuscation path; **add it to the setup steps**
  (`go install golang.org/x/tools/gopls@latest`).
- `test_ablation2.py::test_valid_units_match_judge` — asserts excision patches exist under
  `experiments/ablation_graph/repos/`, which is gitignored and absent on a fresh clone.
  Now skips when the round-1 tree is not local.

## B9 note

The `hacked` verdicts in this run's first trials came from the pre-merge audit (the
`git apply` fallback landed in 1448b97). On merged code, gold passes the hidden suite
under **both** seeds 20260919 and 20260920 — rule A1 holds and the suite is seed-robust
(B5). Defect 4 above is separate and still live before this change: an audit that cannot
run is recorded as a pass. Worth deciding whether a class-(d) audit failure should keep
the trial out of the flip calculation rather than marking it `clean`; that is a policy
call, not changed here.

## Setup steps that were missing from HANDOFF.md

```bash
go install golang.org/x/tools/gopls@latest   # required by the obfuscation pass
docker pull golang:1.23 golang:1.26          # base images per repo go directive
```
