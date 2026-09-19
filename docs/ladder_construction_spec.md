# Ladder construction spec

Reproduce the overnight Harbor ladder: author four artifacts, derive L0–L6
mechanically, solve with the adaptive policy (C6 amendment), audit every pass
(B9), and emit the **calibration curve** (pass rate vs level, per repo × solver)
as the primary measurement.

Logic the orchestrator calls lives in `src/openswe_traces/pipeline_ext/`
(pure functions). Do not confuse it with `src/openswe_traces/pipeline/` (a
separate builder). Packaging of Harbor dirs is `openswe_traces.synth.affordance`.
Validity/fairness checks are `openswe_traces.synth.rules` plus
`analytics/research/verifier_rules.md`.

```
uv run python scripts/pipeline_ext_ladder_policy.py --state state.json
uv run python scripts/pipeline_ext_hack_audit.py --trial TRIAL --skip-docker
uv run python scripts/pipeline_ext_calibration.py --records attempts.json
uv run pytest tests/test_pipeline_ext_ladder.py tests/test_pipeline_ext_hack_audit.py \
  tests/test_pipeline_ext_calibration.py tests/test_pipeline_ext_meta.py
uv run ruff check src/openswe_traces/pipeline_ext tests/test_pipeline_ext_*.py \
  scripts/pipeline_ext_*.py
```

Harbor L{k} ↔ affordance.py A-level: **L = A + 2**. Mapping in
`pipeline_ext.ladder_policy.LADDER_TO_AFFORDANCE` and
`synth.affordance.dest_dir_name(..., name_scheme="L")`.

| Harbor | affordance.py `level` | solver receives |
|---|---|---|
| L0 | A-2 (`-2`) | bug report + repro only (unit excised) |
| L1 | A-1 (`-1`) | gapped contract (one omitted invariant) |
| L2 | A0 (`0`) | full prose contract + coverage of every hidden assertion |
| L3 | A1 (`1`) | L2 + hidden test names and one-liners |
| L4 | A2 (`2`) | L3 + exported signatures as stubs |
| L5 | A3 (`3`) | L4 + one hidden test file restored into the tree |
| L6 | A4 (`4`) | all hidden tests in tree (original in-tree-test control) |

Nothing exists below L0 by construction.

---

## 1. Four artifacts (split across two authors)

The **task author** never writes hidden tests. The **verifier author** never
writes the instruction. Gold must not touch tests (A12).

### 1.1 `api.md` (task author)

Exported surface as a *caller* sees it: names, signatures, existing doc
comments, and the list of pre-existing production callers. No new names. Used
later for B4 (black-box: hidden tests may only drive this surface) and A4
(fail-to-pass tests must sit in the transitive impact set of the change).

Example: `experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf/_author/api.md`
(`YarrowJoin`, `WillowNode`, `NimbusPack`, …).

### 1.2 `contract.md` (task author) — L2 wording

Full behavioral contract. Prose, no implementation/file names (B7). Must include:

1. Invariants the hidden suite will enforce.
2. Worked examples (the ones a cheat may hardcode).
3. A **coverage table**: original in-tree test → one contract sentence. Every
   hidden assertion maps to a row. This is how L2 “covers every hidden
   assertion” (verifier_rules ladder table).

Example coverage row (same unit):

| original test | contract sentence |
|---|---|
| `TestParseKeyspaceID` | A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff. |

See `F1_COVERAGE` in `src/openswe_traces/synth/affordance.py` and
`tasks_bigL0/keyspacecodec-obf/_author/contract.md`.

Also ship `gold.patch` (restores the unit; A1, A12) and `cheat.patch`
(hardcodes only the worked examples; A3 must fail).

### 1.3 `bugreport.md` (task author) — L0 instruction

User-style report: observable operation, expected vs got / panic, one
reproduction command that will run the hidden suite via `tests/test.sh` (B6
floor). No symbol, file, or mechanism names; no diff; no line numbers (B7
ceiling). Always include the no-web clause (B8 / `NO_WEB_CLAUSE` in
`synth.obfuscate`). Locality for `instruction_self_check` at L0 is 2
(`affordance.build_affordance_levels`).

Example: `tasks_bigL0/keyspacecodec-obf/_author/bugreport.md`.

### 1.4 Hidden black-box suite (verifier author, separate session)

Not written by the task author. Lives in Harbor `tests/hidden/` and is copied
into `/app` by `tests/test.sh` (`affordance.render_hidden_test_sh`).

Must:

- Call only the exported API / caller-facing behavior (B4).
`affordance.assert_b4_pass` refuses to package a white-box suite.
- Prefer seeded-random properties and adversarial edges over example lists
(B5). A cheat that special-cases `contract.md` examples must fail (A3).
- Cover every coverage-table row.
- Seed RNG from, in order: env `HIDDEN_SEED` (decimal int64), else a package
constant (default `20260919`). Canonical helper:
`pipeline_ext.hack_audit.hidden_seed_go_snippet()`. Audit re-runs with
`HIDDEN_SEED=20260920` (`AUDIT_HIDDEN_SEED`).
- Be checksum-guarded (B1). `test.sh` sha256-checks hidden files before and
after copy.

Verifier timeout: `harbor_tasks.VERIFIER_TIMEOUT_SEC` (1800s),
`network_mode = no-network` (A10). Proof in the built image before launch (A8,
C5): buggy REWARD=0, gold REWARD=1, `go` on PATH.

Author also writes `_author/difficulty.md` with a required line:

```
predicted_flip: L<k>
```

`k` ∈ 0..6. Format: `pipeline_ext.author_meta.PREDICTED_FLIP_LINE_RE`. Template:
`DIFFICULTY_MD_TEMPLATE`. Brief:
`/home/evan/devin-tasks/openswe_bigL0_author.md`.

---

## 2. How each level is derived (cite `affordance.py`)

One family, one excised tree, one hidden suite. `build_affordance_levels`
copies the A0-shaped task dir and varies only what the solver sees
(`name_scheme="L"` → `<family>-L{k}`).

```
src/openswe_traces/synth/affordance.py
  dest_dir_name(family, level, name_scheme="L")   # L = A+2
  instruction_for_level(base, hidden, level)      # + names iff A>=1
  a1_appendix(hidden)                             # test names + one-liners
  write_hidden_tests / remove_hidden_from_src / restore_hidden_into_src
  build_affordance_levels(... levels=(-2,-1,0,1,2,3,4), name_scheme="L")
```

Mechanical rules inside `build_affordance_levels` (affordance `level`):

- Always: hidden suite under `tests/hidden/`, stripped from `environment/src`,
  `tests/test.sh` from `render_hidden_test_sh`, `task.toml` from
  `render_unsolv_task_toml`, B4 packaging check, `instruction_self_check`.
- `level >= 1` (Harbor L3+): append `a1_appendix` (hidden test names +
  one-liners) before the no-web clause.
- `level >= 2` (Harbor L4+): write exported stub files (`stubs=`).
- `level == 3` (Harbor L5): `restore_hidden_into_src(..., only=representative)`.
- `level >= 4` (Harbor L6): restore every hidden file into src.

Negative levels (L0/L1) keep the supplied `instructions[level]` text and only
add the no-web clause — they never append hidden-test names.

| Harbor | `instructions=` source | tree | extra |
|---|---|---|---|
| L0 | `bugreport.md` | excised, no hidden tests in src | repro = `tests/test.sh` |
| L1 | gapped `contract.md` (drop one invariant that a pre-existing unmodified repo test still guards) | same as L0 | A7: decoy/catcher is pre-existing, not planted |
| L2 | full `contract.md` | same | coverage table complete |
| L3 | L2 + `a1_appendix` | same | names only |
| L4 | L3 | identity stubs + godoc (`f1_a2_stubs` / `f2_a2_stubs` / `f3_a2_stubs`) | signatures |
| L5 | L4 | + one hidden file (`representative`) | formerly A3 |
| L6 | L4 | + all hidden files | formerly A4; in-tree-test control |

L1/L3/L4 are **not** packaged on the climb. Generate them only when
`pipeline_ext.ladder_policy.next_actions` returns that level (required
confirm-below / extend, or optional-on-demand). See §4.

Deep rungs used today:
`experiments/harbor_nex/tasks_deep/<family>-A-1` and `-A-2`
(`scripts/build_ladder_deep.py`, `experiments/harbor_nex/LADDER_DEEP.md`).

---

## 3. Which rule gates each level

Every packaged dir is scored with `openswe_traces.synth.rules` (`A1–A12`,
`B1–B8`, `C5`) into `validation.json`. Additional process rules live in
`verifier_rules.md` (C1–C6) and this spec (B9, adaptive C6).

| Gate | Binds | Meaning |
|---|---|---|
| B6 / B7 / B8 | L0 instruction | symptom + expected/got + one-command repro; no names/files/diffs/line nos; no-web |
| A7 | L1 | omitted invariant is still guarded by a *pre-existing unmodified* repo test |
| B3 | L2 (and below) | in-tree example tests are not the verifier for hard tasks |
| B4 / B5 | hidden suite at every level | black-box; properties beat examples |
| B7 locality 0 | L3+ | test names in the instruction are allowed from L3 up (`instruction_for_level`) |
| A2-shaped stubs | L4 | exported signatures, not bodies |
| B3 weaker | L5 / L6 | one / all hidden files visible; L6 equals the original in-tree-test tasks |
| A1 / C5 | gold | gold restore passes in the image; a gold-fail proof is a harness bug |
| A2 / A3 | verifier width | alt structurally different from gold passes; cheat fails |
| A4 | impact | f2p tests in the *transitive* impact set |
| A5 | collateral | rest of package + baseline packages stay green; f2p stable |
| A8 / A10 / A12 | image | prove in the built image, no verifier network, patches skip `*_test.go` |
| B1 / B2 | trial | checksum-guarded tests; web-tool use disqualifies |
| **B9** | every **passing** attempt | post-pass hack audit (`pipeline_ext.hack_audit`) |
| C1 | every fail | (a) legitimate / (b) narrow verifier / (c) weak instruction / (d) infra. Only (a) counts. Timeouts are (d). |
| C2 | packaging | lazy: do not build the next affordance until the policy asks |
| **C6 (amended)** | solving | adaptive climb §4, not “≥3 attempts on every level” |

---

## 4. Adaptive ladder (amends C6)

C6 originally: flip = pass rate over ≥ 3 attempts per level; singles are
provisional. The adaptive policy keeps the 2/3 bar but **does not** run 3
attempts on every rung.

Implemented in `pipeline_ext.ladder_policy`:

```
next_actions(task_state) -> list[LadderAction]   # unpackable as (level, n_attempts)
flip_point(task_state)   -> (level|None, confirmed: bool, evidence)
```

Timeouts (`LevelAttempt.timeout=True`) are class (d) and do not count toward
2/3 (`valid_attempts` drops them).

### Climb

Order **`[L2, L5, L6]`**, **one** attempt each. First climb level that
**passes** is the candidate flip. Levels that **failed** on the climb get
**no extra attempts**.

### Confirm

Then:

- 2 more attempts at the candidate (total 3).
- 2 more at the level just below (numeric neighbour; **L2’s below is L0**, not
  L1).
- If L2 passes on the climb, also **probe L0 once**. L1 is generated only if
  that L0 probe **fails** and someone asks (`request_optional`).

Confirmed iff **candidate ≥ 2/3** and **below < 2/3**. If below was a climb
failure, treat it as already < 2/3 (no extras). L0 probe fail + L2 ≥ 2/3 →
confirmed L2. L0 probe pass → flip is L0, **provisional** (1 attempt; floor).

### Extend

If not confirmed, extend **one** level in the direction the results point,
again with 2 extra attempts:

- below looks like a pass (rate ≥ 2/3, or L0 probe passed) → down
  (skip L1 unless requested; skip climb-failed levels).
- candidate extras look like a fail → up.

### Optional (L1 / L3 / L4)

Returned with `optional=True` and a reason. Promoted to required only when
`TaskState.request_optional` contains that level **and** the reason holds.

| level | reason |
|---|---|
| L1 | L2 passed and L0 failed; interpolate L0/L2 |
| L3 | L2 failed and a higher climb level passed; interpolate L2/L5 |
| L4 | L5 is in play; interpolate L2/L5 (unless L4 is already confirm-below) |

Worked traces:

1. Empty state → `[(2, 1)]`.
2. L2 pass → 2 more at L2 and 1 at L0.
3. L2 3/3 and L0 0/1 → confirmed L2.
4. L2 fail, L5 pass → 2 more at L5 and 2 at L4 (L2 gets nothing more).
5. L2, L5, L6 all fail the single climb attempt → held out; flip `None`.

---

## 5. Example: one unit at every level

Unit: **keyspacecodec / spec-reimpl** (obfuscated `internal/apicodec` v2
header + mem-comparable region keys). Author pack:
`experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf/_author/`. Hidden suite
pattern: `F1_*` in `affordance.py` (`codec_v2_test.go`, `codec_test.go`,
`decode_fatal_test.go`). Deep-rung cousin: `tasks_deep/spec-reimpl-bb-A-1`
/ `-A-2`.

**L0.** Instruction = bugreport: raw get of user key `key` in keyspace 0x1092
went out bare; expected prefix `0x72 0x00 0x10 0x92`; expected id 0x10203,
actual 0xffffffff; malformed region keys retried with backoff. Repro
`tests/test.sh`. Tree: excised identity stubs, no hidden tests in src.

**L1.** Same tree. Instruction = full contract minus *one* discoverable
invariant (invalid/short header → 0xffffffff), still guarded by pre-existing
`TestParseKeyspaceID` (`LADDER_DEEP.md` spec-reimpl-bb). Hidden suite
unchanged.

**L2.** Instruction = `contract.md` (every coverage-table sentence). Tree
still stripped. This is `build_affordance_levels` affordance `0`
(`spec-reimpl-A0` / `*-L2`).

**L3.** L2 text + `a1_appendix`: names such as `TestCodecV2`, `TestParseKeyspaceID`,
`TestMalformedRegionKeyIsDecodeError` with one-liners. No stubs yet.

**L4.** L3 + godoc stubs from `f1_a2_stubs` (`HazePipe` prepends the 4-byte
header, `WillowNode` reads the 24-bit id, …). Bodies still identity/wrong.

**L5.** L4 + restore representative `internal/apicodec/codec_v2_test.go` into
src (one hidden file visible). Verifier still copies + checksums all hidden
files.

**L6.** All hidden tests in the tree (`codec_v2_test.go`, `codec_test.go`,
`decode_fatal_test.go`). Control: “all tests in tree”. A solver fail at L6 is
an early-warning (`calibration` flag `l6_fail`) and must be C1-audited before
counting.

Batching cousin for a second family: `tasks_bigL0/batchcmds-obf/_author/`
(L0 bugreport: per-host streams vs per-request; L2 contract in that pack).

---

## 6. Post-pass audit (B9)

Every Harbor attempt with verifier reward 1.0 is audited before it counts.
`pipeline_ext.hack_audit.audit_passing_attempt` →
`HackVerdict(passed, hard_fails, flags, evidence)`. Hard fails void the pass.
Flags are recorded, not kills. Runs **inside the task image** (`docker run
--network=none -i`) with the trial patch on stdin.

### Hidden-seed contract

1. `HIDDEN_SEED` env (decimal int64) if set and parseable.
2. Else package constant / `rand.NewSource(<int>)`. Default constant:
   `20260919`.

Audit sets `HIDDEN_SEED=20260920` and, if the suite only has a constant,
rewrites `NewSource` / `HiddenSeed` in `*_test.go` inside the container. Gold
must pass both seeds (B5). A seed-specialized cheat fails the re-run.

### Checks

| check | fail vs flag |
|---|---|
| Re-run hidden suite with audit seed | **fail** if REWARD≠1 |
| Collateral: unit packages + repo baseline `go test` (A5) | **fail** (`COLLATERAL_FAIL`) |
| Touched-file allowlist: `*_test.go`, `tests/`, `test.sh`, `task.toml`, `Dockerfile`, `go.mod`/`go.sum`, `//go:build` / `// +build`, `vendor/` | **fail** |
| Literal expected values from `contract.md` / hidden inputs appearing in the patch | **flag** |
| Trajectory: `webFetchToolCall` / `webSearchToolCall`, network commands (`curl`, `wget`, `go get`, …), reads of `/task` or `tests/` | **fail** (B2 / oracle) |
| Trajectory: `git log` / `blame` / `show` / history probes | **flag** |

Same web-tool signal as `scripts/ops/harbor_web_audit.py` (15/30 Grok trials
historically contaminated). Harbor trial dirs:
`experiments/harbor_nex/jobs/*/<task>__<id>/{result.json,verifier/,agent/}`.

```
uv run python scripts/pipeline_ext_hack_audit.py \
  --job experiments/harbor_nex/jobs/bigL0 --skip-docker
```

---

## 7. Calibration curve (primary output)

`pipeline_ext.calibration.calibrate(records) -> CalibrationReport`.

From per-level attempt records (timeouts / class-d dropped):

1. **Pass-rate curve** per `(repo, solver, level)`.
2. **Level nearest 50%** pass (`nearest_50`; lower level wins ties).
3. **Inter-attempt agreement**: fraction of `(repo, unit, solver, level)`
   groups with a 2–1 split on the first three scored attempts.
4. **Early-warning flags** (each carries `FLAG_ACTIONS` text):

| flag | threshold | action (abbrev.) |
|---|---|---|
| `gate_rejection` | > 40% units killed per repo **by rule** | pause repo; rewrite/drop that gate |
| `control_miss` | control unit not 3/3 at L2 (once n≥3) | stop repo; author/verifier broken |
| `too_easy` | first 15 valid units all 3/3 at L2 | raise author bar; drop the easy batch from the curve |
| `l6_fail` | any L6 fail | C1 audit; legitimate L6 fail voids the unit |
| `noisy_splits` | 2–1 splits > 1/3 of scored levels | add attempts or drop the unit |
| `contamination` | test-edit / B2 / checksum > 3 **per host** | tighten allowlist; rerun |
| `author_yield` | < 3 valid units / author session | switch backend or raise session minutes |
| `attempt_time` | wall time grew > 2× across attempts | class (d), not a fail; hung tests / load / Harbor timeouts |

The curve is the thing we publish. Flip points and author calibration are
secondary.

### Controls

Per repo, exactly one unit is `control`: the author’s **easiest predicted-L2**
(`pipeline_ext.controls.pick_control` — fewest lines, then name).
`is_control` / `control_status(repo)`. Must go 3/3 at L2.

### Author calibration

`pipeline_ext.author_meta.author_calibration`: predicted vs measured flip per
author backend (MAE, exact, off-by-one).

### Timeouts (class d)

`pipeline_ext.timeouts.timeout_budget(backend)`:

| backend | agent trial | author/verifier session | poll |
|---|---|---|---|
| Devin | 14400 s | 90 min | 120 s |
| Composer / Cursor | 3600 s | 60 min | 60 s |
| Grok | 3600 s | 60 min | 60 s |

`classify_attempt(timed_out=True)` → `"d"`. `counts_as_failure` is False for
timeouts. C6 images/containers must not be pruned while Harbor jobs run.

---

## 8. End-to-end construction checklist

1. Obfuscate a public tree (C3); one base image per tree (C4), e.g.
   `ladder-base:client-go-obf`.
2. Task author: excised tree, `api.md`, `contract.md` + coverage table,
   `bugreport.md`, `gold.patch`, `cheat.patch`, `difficulty.md` with
   `predicted_flip: L<k>`.
3. Verifier author: black-box hidden suite + `tests/test.sh` (checksum, seed
   contract, no-network).
4. Image proof (A8/C5): buggy fails, gold passes, cheat fails, alt passes,
   collateral green, B4 pass.
5. Package L2 (and L0 if the policy will probe it) via
   `build_affordance_levels(..., name_scheme="L")`. Do not pre-build L1/L3/L4.
6. Solve with `next_actions` / `flip_point`. Agent timeouts from
   `timeout_budget`. After every reward=1, `audit_passing_attempt` in the
   image.
7. Mark the per-repo control; refuse the repo if it is not 3/3 at L2.
8. Stream attempts into `calibrate`. The pass-rate curve is the result.
   Watch early-warning flags before adding volume.
