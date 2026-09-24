# Pipeline generalisation beyond Go — Python unit end to end (2026-09-19)

One Python repo (`pallets/itsdangerous`), one authored unit (`signer`), packaged L0+L2 with
`affordance.py`, materialised through the same `materialize.py` path as the Go repos, and proven
by the in-image preflight gate: bare fails with pytest assertions, gold passes, cheat fails.
This is the first non-Go measurement of the pipeline; the question is which abstractions hold and
which were Go-shaped. Branch `closure-P`, worktree `oswt-closureP`.

## 1. Repo pick

| candidate | verdict |
|---|---|
| `psf/cachecontrol` | heavier deps (urllib3, msgpack, filelock); cache backends need filesystem fixtures |
| `pallets/itsdangerous` | **picked** — BSD-3-Clause, pip-installable (flit), 4 source files / ~1200 LOC, 297 in-tree tests, builds in seconds in `python:3.12-slim` |
| `encode/httpx` core | bigger surface (httpcore, anyio, certifi); slow install |
| `python-jsonschema/jsonschema` | large dep tree (attrs, referencing, rpds, jsonschema-specifications) |
| `kurtmckee/feedparser` | sax-heavy, larger; test suite weaker |

Pinned: tag 2.2.0, commit `096c8d42545d3b68ea21a4f890fb2b2d8979c0bd`, in `repos.yaml` with the
four non-Go keys: `image`, `install_cmd`, `test_cmd` (`{files}` template), `test_glob`.

## 2. Unit: signer

Closure: `src/itsdangerous/signer.py` — `Signer.derive_key` (returns `b"broken-key"`, ignores
salt + derivation scheme), `Signer.get_signature` (empty signature), `Signer.verify_signature`
(always `False`). `sign`/`unsign`/`validate` are the unchanged callers, so every public entry
point is broken while the module keeps its exact API shape and imports cleanly (Python has no
panic; "wrong constant" is the language's stub idiom, supplied as patch data).

Hidden suite (`tests/test_itsdangerous/test_signer_bb.py`, black-box, B4): seeded-random
round-trip / signature-format / tamper / wrong-key / rotation-scan-order properties
(`random.Random(HIDDEN_SEED or 20260919)`), plus the contract's worked examples and edges.
Local counts:

| tree | hidden suite |
|---|---|
| base (gold) | 18 passed; also 18 passed under `HIDDEN_SEED=20260920` |
| bare excised | 16 failed, 2 passed (the 2 green test genuinely unchanged code: separator validation, abstract algorithm) |
| cheat (hardcodes the 4 worked examples) | 14 failed, 4 passed (worked examples + the same 2) |

Authored artifacts: `experiments/pipeline/authored/itsdangerous/signer/_author/` (api.md,
bugreport.md, contract.md, closure.md, difficulty.md, hidden/, excision/gold/cheat patches).
Builder: `src/openswe_traces/synth/python_unit.py` + `scripts/build_python_unit.py` — unit
content only; it drives the existing `materialize.materialize_task` and
`affordance.build_affordance_levels` (the equivalent Go builders are the per-repo
`synth/composerver_*_batch.py` scripts).

## 3. Preflight results (in-image, task's own test.sh, no solver)

| repo | unit | level | bare | gold | cheat | seconds |
|---|---|---|---|---|---|---|
| itsdangerous | signer | L0 | fail (pytest `FAILURES`, `AssertionError`) | pass | fail | 7.6 |
| itsdangerous | signer | L2 | fail (pytest `FAILURES`, `AssertionError`) | pass | fail | 8.3 |

Gate verdict PASS for both dirs; `rule_verdicts` A8/A1/A3 recorded; no `[setup failed]`, no
collection error, no `ModuleNotFoundError` on any of the six runs.

## 4. Where the pipeline assumed Go — config vs code, honestly

**Config only (no pipeline code touched):**

- `repos.yaml`: the `image` / `install_cmd` / `test_cmd` / `test_glob` keys (added during K's
  preflight work, exercised end to end here). `prepare.base_dockerfile_for` and
  `affordance.render_test_cmd` already consumed them.
- The whole unit: patches, hidden suite, bugreport/contract are data.

**Code/data edits required (each one a real Go-shaped assumption):**

| # | file | change | why |
|---|---|---|---|
| 1 | `pipeline/preflight.py` | `INFRA_SIGNATURES["python"]` += `ERROR collecting`, `(?m)^=+ ERRORS =+$`, `INTERNALERROR`, `Fatal Python error`, `ImportError while importing` | pytest's actual "runner never ran" shapes: collection-time import failure, setup-error section, internal crash. Data rows, no new logic. |
| 2 | `pipeline/preflight.py` | `LANGUAGE_SUFFIXES` += `test_*.py` (fnmatch glob; literal rows still `endswith`) | pytest's default collection pattern was invisible to the suffix fallback (only `*_test.py` matched). |
| 3 | `pyproject.toml` | `[tool.pytest.ini_options] testpaths = ["tests"]` | the repo's own suite is now scoped: `uv run pytest` was recursively collecting the vendored second-language suites under `experiments/` (their deps are not dev deps) and dying at collection — the Python analogue of the Go "no tests to run masks a real fail" class, one level earlier (collection vs masking). |
| 4 | `tests/test_pipeline.py` | `test_yaml_config_and_repos_load` no longer asserts `all(r.language == "go")` | the test itself encoded the Go-only repo bank. |
| 5 | `src/openswe_traces/synth/python_unit.py` (new) | unit builder | no missing pipeline abstraction — the Go batches are also per-repo scripts in `synth/`. |

B6/B7 self-checks passed on this unit (`symptom=True repro=True`, no leaks) — the bugreport
uses explicit `Expected:` / `Actual:` lines, which the Go bugreports in the bank mostly do not;
that is a language-neutral lesson, not a Python one.

**Job O:** `src/openswe_traces/gate/` exists in closureO (rule-engine consolidation). This
worktree extends the preflight *signature tables* (data) in its own `preflight.py`; no parallel
rule engine was built here.

## 5. What held vs leaked

**Held without change:** `materialize.py` (base-tree copy + excision patch application, `patch
-p1` semantics identical), `affordance.build_affordance_levels` (L0/L2 derivation, negative
levels, instruction rendering, B4 packaging gate, `test_cmd` rendering into test.sh),
`preflight` gate flow + verdict caching, `rules.check_b4`'s Python branch, B5 markers, the
"stub body" concept (panic in Go, wrong constant in Python — both just patch data), and the
soft-infra protection (`collected 0 items` / `no tests ran` never masks a real `FAILED`).

**Leaked (needed edits):** the language signature rows (data), the suffix table (data), the
repo's own pytest scoping (repo config), one Go-only test assertion. No pipeline *code*
abstraction was missing; the leak surface was exactly the two tables the README already calls
"the extension point for new languages".

## 6. What a THIRD language would need (TypeScript, Rust) — derived from what this hit

| step | TypeScript (node) | Rust |
|---|---|---|
| repos.yaml | `image: node:22-slim`, `install_cmd: npm ci`, `test_cmd: npm test -- {files}` or `vitest run {files}`, `test_glob: *.test.ts` | `image: rust:1.8x-slim`, `install_cmd: cargo build --offline` (bake deps at image build), `test_cmd: cargo test --offline`, `test_glob: *_test.rs` |
| INFRA_SIGNATURES | node row exists; ADD `error TS\d+` (tsc), `ts-node`, `ERR_PNPM`, and verify-time `npm` fetch patterns (no-network runs print DNS failures — like go's `go: downloading`) | rust row exists (`error[E\d+]`, `could not compile`, `cannot find`, `failed to resolve`); verify `warning: unused` never classifies infra; `error: no tests to run` soft row exists |
| LANGUAGE_MARKERS / SUFFIXES | package.json marker exists; `.test.ts`/`.spec.ts` rows exist | Cargo.toml marker exists; `_test.rs` row exists |
| seed contract | env `HIDDEN_SEED` or module constant (mirror the Python pattern) | `rand::rngs::StdRng::seed_from_u64` from env/const; audit seed rewrite is currently Go-only in `hack_audit` — needs a non-Go path |
| stubs | `throw new Error("excised")` / wrong constants | `todo!()` / wrong constants (compiles, fails assertions) |
| fixtures | `tests/test_preflight_node.py` mirroring `test_preflight_python.py` | `tests/test_preflight_rust.py` |
| repo config | same `testpaths` scoping already in place | same |

Open follow-ups, not done here (no solver trials were run): `hack_audit`'s seed-rewrite and
touched-file allowlist are Go-shaped (`*_test.go`, `NewSource` rewrite); the Python B9 audit
path needs the env-seed contract wired into the audit runner.
