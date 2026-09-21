# Verifier report — nats-server batch-2 hidden suites

Date: 2026-09-20. Job: `VFnatsserver`. Role: VERIFIER AUTHOR only — no
`bugreport.md`, `contract.md`, or `gold.patch` was read; oracles were derived
from `api.md`, `DETAILS.md`, the excised source tree, and upstream-ancestor /
in-tree test files only.

## Result: 14/14 units packaged and preflighted, 0 excluded

195 hidden `TestDetailNN_*` property tests across 14 suites — exactly one test
per numbered `DETAILS.md` line. Seeded via `HIDDEN_SEED` (default 20260919;
audit seed 20260920 also verified). Every suite: bare excised tree FAILS with
assertion/panic (never setup/build), gold PASSes on both seeds, cheat FAILS.
Preflight ran in-image via closure-O `preflight_task` (`--network=none` per
task.toml); executed-tier rules A1/A3/A5/A8/A10 pass for every package.

## Preflight table (in-image, per package)

| task | bare | gold | cheat | s |
|---|---|---|---|---|
| archiveio-L0 | fail | pass | fail | 18 |
| archiveio-L2 | fail | pass | fail | 16 |
| confparse-L0 | fail | pass | fail | 18 |
| confparse-L2 | fail | pass | fail | 18 |
| cronparse-L0 | fail | pass | fail | 149 |
| cronparse-L2 | fail | pass | fail | 146 |
| gslsublist-L0 | fail | pass | fail | 19 |
| gslsublist-L2 | fail | pass | fail | 18 |
| hashwheel-L0 | fail | pass | fail | 19 |
| hashwheel-L2 | fail | pass | fail | 16 |
| ipqueue-L0 | fail | pass | fail | 159 |
| ipqueue-L2 | fail | pass | fail | 165 |
| jsversioning-L0 | fail | pass | fail | 173 |
| jsversioning-L2 | fail | pass | fail | 170 |
| jwtvalidate-L0 | fail | pass | fail | 164 |
| jwtvalidate-L2 | fail | pass | fail | 142 |
| ldapdn-L0 | fail | pass | fail | 20 |
| ldapdn-L2 | fail | pass | fail | 20 |
| proxyproto-L0 | fail | pass | fail | 155 |
| proxyproto-L2 | fail | pass | fail | 150 |
| seqset-L0 | fail | pass | fail | 20 |
| seqset-L2 | fail | pass | fail | 13 |
| stree-L0 | fail | pass | fail | 20 |
| stree-L2 | fail | pass | fail | 17 |
| subjecttransform-L0 | fail | pass | fail | 139 |
| subjecttransform-L2 | fail | pass | fail | 152 |
| utilparse-L0 | fail | pass | fail | 145 |
| utilparse-L2 | fail | pass | fail | 162 |

(Seconds include first-build image time where uncached; cached tasks ~15-20s.)

## Per-unit coverage

| unit | package | details | changed file(s) |
|---|---|---|---|
| ldapdn | internal/ldap | 10 | internal/ldap/dn.go |
| archiveio | server/archive | 11 | server/archive/archive.go |
| hashwheel | server/thw | 8 | server/thw/thw.go |
| seqset | server/avl | 17 | server/avl/seqset.go |
| gslsublist | server/gsl | 15 | server/gsl/gsl.go |
| stree | server/stree | 15 | server/stree/stree.go, parts.go |
| confparse | conf | 16 | conf/parse.go |
| subjecttransform | server | 17 | server/subject_transform.go |
| ipqueue | server | 12 | server/ipqueue.go |
| utilparse | server | 18 | server/util.go |
| proxyproto | server | 16 | server/client_proxyproto.go |
| jwtvalidate | server | 14 | server/jwt.go |
| jsversioning | server | 12 | server/jetstream_versioning.go |
| cronparse | server | 14 | server/cron.go |

Suites live in `src/openswe_traces/synth/testdata/batch2_natsserver/`; driver +
packager in `src/openswe_traces/synth/verifier_batch2_natsserver.py`. Packages:
`experiments/pipeline/tasks_batch2/nats-server/<unit>-L{0,2}` (L0=affordance
-2, L2=0), each with checksum-guarded `tests/test.sh`, hidden suite under
`tests/hidden/`, CURSOR allowlist in `task.toml`, gold+cheat patches.

## Oracle corrections made during iteration (black-box, from observed behavior)

- ldapdn: trailing-comma tolerance at EOF; `;` is a literal value char, not a
  separator; DER SET OF canonicalizes within-RDN attribute order (multiset
  compare); values must be hex-escaped when re-rendered for round-trip.
- archiveio: entry body is HeaderSize+PayloadSize as one stream; corrected
  wire-order construction, zero-payload and overflow expectations, lazy magic.
- hashwheel: `ExpireTasks` compares against wall clock — expirations generated
  around `time.Now()` spanning wheel wraparound.
- seqset: v2 tolerates zero-node/nonzero-size streams; kept documented bound.
- gslsublist: multi-char tokens are literal; bare `*` matches one token;
  duplicate inserts bump count but delivery dedups identical subject+value.
- stree: test-side off-by-one in expected values; `>` mid-filter is literal.
- confparse: `5kbb`/`5kibi` lex as nil, `7eibe` as string; simplified
  unsupported include/brace shapes.
- subjecttransform: errors wrap via `mappingDestinationErr` without Unwrap —
  message-text matching; `splitfromleft/splitfromright/partition` take wildcard
  index + position; `>` yields no numbered slot; `.` split emits verbatim.
- ipqueue: `popOne` vs `pop` in-progress accounting; `recycle` resets caller
  slice (pointer captured before recycle).
- utilparse: green first pass.
- proxyproto: v1 bound rejects missing-CRLF, not bad address; UNSPEC family
  requires STREAM proto; corrected version-nibble cases.
- jwtvalidate: sentinel must be a bearer token; `SetScoped(true)` needed for
  scoped claims; TrustedKeys vs TrustedOperators are mutually exclusive.
- jsversioning: `errorOnRequiredApiLevel` takes a full NATS header block
  (`NATS/1.0\r\nNats-Required-Api-Level: v\r\n\r\n`); non-integer and >5
  reject, ≤5/absent pass.
- cronparse: `n/step` single-bound semantics (`jul/1` = Jul..Dec step 1);
  empty/repeated terms tolerated.

## Incidents

- Concurrent job pruned `ladder-base:nats-server` mid-preflight: 3 tasks
  (proxyproto-L2, seqset-L0, seqset-L2) recorded `task image build failed`
  and the gate cached it in `validation.json`. Rebuilt the base image, re-ran
  with `force=True`: all three pass (bare=fail, gold=pass, cheat=fail).
- `pytest tests`: 25 pass, 1 pre-existing worktree-symlink failure in
  `test_cheap_difficulty.py` (`traces_data` resolves outside worktree root),
  unrelated to this job. `ruff check` clean on new code.

## Reproduce

```
uv run python -c "from openswe_traces.synth import verifier_batch2_natsserver as v; \
  [v.iterate_unit(u) for u in v.units()]"
# packages already written under experiments/pipeline/tasks_batch2/nats-server/
# per-task gate: (in oswt-closureO) uv run python scripts/preflight_task.py <task_dir>
```

Overlap check: `check_unit_overlap.py --extra <AU worktree>` — 14 new units vs
0 existing, CLEAN. Log: `outputs/VFnatsserver.log`. No commit made.
