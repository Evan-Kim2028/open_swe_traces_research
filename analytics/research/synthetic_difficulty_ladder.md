# Synthetic difficulty ladder — 5-round adversarial loop synthesis

2026-09-18. Numbers from `experiments/harbor_nex/{RESULT,ITER_3–8,TWO_REPO_TASK}.md`,
`experiments/harbor_nex/jobs/*/result.json` (and each trial `result.json`),
`experiments/codegraph_bugs/RESULT.md`, `analytics/research/irt_summary.md`.
`HARD_TASKS.md` was not on disk; hard-task knobs come from `ITER_*.md` plus the
three `tasks_hard/` instructions and patches. `ITER_1.md` / `ITER_2.md` were
also missing; `jobs/grok-xhigh` and `jobs/grok-xhigh-hard` are those rounds.

Solver wall = `finished_at − started_at` on the trial `result.json` (first
attempt when two were stored). Nex 429 trials are infra, not scored misses.

Failure classes used in the loop notes (`ITER_4.md`): **(a)** verifier too
narrow, **(b)** instruction insufficient, **(c)** legitimate miss, **(d)** infra
(quota / timeout / overwritten trial).

## 1. ELI5

We built Harbor tasks from injected Go bugs (then feature deletions) in client-go and a daily-problems repo.
Frontier solver: Grok 4.6 xhigh (`cursor-cli`). Cheap prober: Nex-N2.5-Pro free (`terminus-2`).
Grok passed all 25 scored tasks, from 1-hop inverts through two-repo, decoys, L2 locality, sequence bugs, and feature excision.
It never stopped passing. Knobs (hops, sites, decoys, cross-module, guards, sparsity, locality, excision) only moved wall time (1.34 → 7.57 min).
Nex passed 3 daily + IsFake + NewRequest, failed IsErrNotFound (class c), and hit 429 on the rest (class d).
`openswe-synth` can set those knobs on a codegraph-indexed repo and emit `validation.json` plus a Harbor dir.
We cannot yet make a Grok-fail task on purpose: the Grok-fail region is empty.
Nex’s only real miss is in the same 1-hop cell as two Nex passes, so knobs do not cut Nex either.
Mid-difficulty data needs a weaker solver on rungs 3–5 (DualExpo / ExtractPhysical), not a higher Grok rung.
Concurrency / config / env classes are still missing from the ladder.

## 2. Ladder

Knobs: hops = shortest reverse-`calls` test←cause (sqlite; “dyn” = interface
dispatch missing from the graph); sites = mutated functions; decoys =
unmodified plausible causes; xmod = different packages / no import edge /
library→consumer; guard = existing test (or checksum) that a decoy-fix fails;
sparse = 0 sqlite Test* callers within 3 hops of the impl. L = instruction
locality (0 names tests, 1 package+cmd, 2 behavior only). Sorted by rung, then
hops, sites, decoys, xmod, guard, sparse.

| rung | task | class | hops | sites | decoys | xmod | guard | sparse | L | Grok | min | Nex | fail |
|---:|---|---|---:|---:|---:|---|---|---|---:|---|---:|---|---|
| 0 | dailycodingproblem-go-twosumbest | intra-pkg control | 1 | 1 | 0 | N | N | N | 0 | pass | 1.34 | pass 1.19 | |
| 0 | dailycodingproblem-go-twosumbrute | intra-pkg control | 1 | 1 | 0 | N | N | N | 0 | pass | 1.39 | pass 2.29 | |
| 0 | dailycodingproblem-go-match | intra-pkg control | 1 | 1 | 0 | N | N | N | 0 | pass | 3.06 | pass 5.29 | |
| 1 | client-go-iserrnotfound | 1-site invert | 1 | 1 | 0 | N | N | N | 0 | pass | 2.51 | **fail** 4.52 | c |
| 1 | client-go-isfakeregionerror | 1-site invert | 1 | 1 | 0 | N | N | N | 0 | pass | 4.36 | pass 23.10 | |
| 1 | client-go-newregionrequestsender | 1-site ctor | 1 | 1 | 0 | N | N | N | 0 | pass | 3.24 | 429 | d |
| 1 | client-go-newrequest | 1-site factory | 1 | 1 | 0 | N | N | N | 0 | pass | 6.47 | pass 42.35 | |
| 1 | client-go-newbackofferwithvars | 1-site ctor | 1 | 1 | 0 | N | N | N | 0 | pass | 4.03 | 429 | d |
| 1 | client-go-getglobalconfig | 1-site, 2 hops | 2 | 1 | 0 | N | N | N | 0 | pass | 4.66 | 429 | d |
| 2 | client-go-decodekeyv1 | 1-site codec (`tasks_hard`) | 1 | 1 | 0 | N | N | N | 0 | pass×2 | 1.65 / 1.52 | — | |
| 2 | client-go-getstoretypebymeta | 1-site meta (`tasks_hard`) | 1 | 1 | 0 | N | N | N | 0 | pass×2 | 4.23 / 4.76 | — | |
| 2 | client-go-keyspaceidcodec | 2-site xor (`tasks_hard`) | 2 | 2 | 0 | N | N | N | 0 | pass×2 | 2.89 / 1.69 | — | |
| 3 | client-go-keyspaceprefix | 3-site same pkg | 1–2 | 3 | 0 | N | N | N | 0 | pass | 2.65 | — | |
| 3 | client-go-dualexpo | 2-site, no import | 5 | 2 | 0 | Y | N | Y | 0 | pass | 2.83 | — | |
| 4 | client-go-onepc-scope | library→consumer | 3 | 1 | 0 | Y | Y | N | 0 | pass | 4.11 | — | |
| 5 | client-go-intervalcontains | fair amb. (Contains vs ByEnd) | 1 | 1 | 1 | N | Y | N | 0 | pass | 3.33 | — | |
| 5 | client-go-extractphysical | fair amb. (vs GetTimeFromTS) | 2 | 1 | 1 | N | Y | N | 0 | pass | 2.00 | — | |
| 5 | client-go-decodebucketkeys | fair amb. (vs decodeKey) | 2 | 1 | 1 | N | Y | Y | 0 | pass | 2.46 | — | |
| 5 | client-go-getphysical | fair amb. L1 (vs ExtractPhysical) | 2 | 1 | 1 | N | Y | Y | 1 | pass | 2.53 | — | |
| 5 | client-go-gettimefromts (iter5) | fair amb. 4-hop (vs tsoSub) | 4 | 1 | 1 | N | Y | N | 0 | pass | 3.26 | — | |
| 6 | client-go-gettimefromts (iter6) | L2 + inverse decoy | 2–3 | 1 | 1 | N | Y | N | 2 | pass | 2.52 | — | |
| 7 | client-go-gettimestamp | sequence vs single-call | 1 dyn | 1 | 0 | N | N | Y | 1 | pass | 2.18 | — | |
| 7 | client-go-memsetvalue | sequence overwrite | 3 | 1 | 0 | N | N | N | 2 | pass | 2.14 | — | |
| 8 | client-go-interceptor | excision, interface kept | 3 | 4 | 0 | Y | N | N | 2 | pass | 3.85 | — | |
| 8 | client-go-batchcmds | excision, interface deleted | 3 | 7 | 0 | Y | N | N | 2 | pass | 7.57 | — | |

25 Harbor tasks. Grok: 25/25 pass (hard job stored two attempts, both 1.0).
Nex scored: 5 pass + 1 fail on 6 trials; 12 RateLimitError (class d) of 18.
Nex job mean reward 0.278 (5/18), pass@2 0.556 (`RESULT.md`).

Job dirs: `grok-xhigh` mean 1.0 (9); `grok-xhigh-hard` 1.0 pass@2=1.0 (6);
`grok-xhigh-iter3`…`iter8` each mean 1.0 (2); `grok-xhigh-two-repo` 1.0 (1).

## 3. Threshold estimate

**Highest knob setting Grok passed:** rung 8, interface deleted, 7 functions / 4
files / ~221 lines, hops 3, L2 (`client-go-batchcmds`, 7.57 min). Also passed
rung 8 interface-kept (4 functions / 2 files, 3.85 min) and rung 4 two-repo
(hops 3, consumer checksum, 4.11 min).

**Lowest legitimate fail:** Grok — none. Nex — `client-go-iserrnotfound`
(rung 1, hops 1, 1 site, no decoy; 4.52 min, class c). Two other scored 1-hop
client-go bugs **passed** Nex (IsFake 23.10 min, NewRequest 42.35 min), so that
fail is not a hops/sites threshold.

**Bracket:** Grok-fail is open above rung 8 / D_max. Nex-fail vs Nex-pass sits
inside the same 1-hop / 1-site cell, so the Grok–Nex ability gap is **not**
identified with these knobs. Daily (rung 0) is below both solvers.

**One-parameter score** (construction intensity, not a Grok-fail predictor):

```
D = rung + 0.25·hops + 0.75·max(sites−1,0) + 1.25·decoys
    + 1.5·xmod + 0.5·guard + 0.5·sparse + 0.5·L + 1.0·iface_rm
```

| side of a Grok cut | n tasks | D range (this formula) |
|---|---:|---|
| Grok pass (all observed) | 25 | 0.25 (TwoSumBest) … 16.75 (BatchCmds) |
| Grok fail | 0 | — |
| Nex pass (scored) | 5 | 0.25–1.25 |
| Nex fail (c) | 1 | 1.25 (tied with two Nex passes) |
| Nex infra (d) | 3 tasks × 2 att. | 1.25–1.50 |

Cut that separates Grok pass/fail: **none** (0 fails; any cut D > 16.75 is
vacuous). Cut that separates Nex pass/fail on knobs: **none** (IsErrNotFound
D = IsFake D = NewRequest D = 1.25). Fitted to binary Grok outcomes the
weights are unidentified (all y=1). Fitted to Nex scored outcomes the
error-minimizing threshold anywhere in (0.25, 1.25) still misclassifies the
1-hop split 2 pass / 1 fail.

**What we can set on purpose:** rung, hops, sites, decoys, xmod, guard,
sparsity, locality, keep_interface. **What we cannot yet hit:** a task Grok
xhigh fails.

## 4. Verifier calibration audit

No Grok fail was reclassified as (a) too-narrow or (b) instruction-insufficient:
Grok never failed (`ITER_3`–`ITER_8`, every `grok-xhigh*` `result.json`).
Nex’s only scored fail (`IsErrNotFound`) was kept as **(c)** — `RESULT.md`:
“the 1 fail is a real miss, reward 0”.

Construction / instruction fixes that were **not** solver-fail reclasses:

| case | what looked wrong | class | change | re-run |
|---|---|---|---|---|
| `dailycodingproblem-go-twosumbest` smoke-rerun | pass 4.11 min **without** a bug (`RESULT.md`: git-archive mtimes, patch not applied) | false-easy construction | rebuild snapshot so the mutation is present | valid smoke **pass** 1.73 min (104 s; 6 API calls) |
| `client-go-getglobalconfig` instruction | nested `integration_tests/go.mod` made the issue a module-prefix error instead of f2p | would have been (b) | `discover_packages_for_tests` skips nested `go.mod` | docker buggy fail / fixed pass; Grok pass 4.66 min; Nex 429 (d) |
| `client-go-isfakeregionerror` docker restore | 1st restore flake `TestTiKVRecoveredFromDown` nil panic | infra flake | retry restore | pass; Nex later pass 23.10 min |
| alt/cheat on all shipped bugs | breadth check, not a solver outcome | — | alt must pass f2p; cheat must fail | all six original client-go bugs + later ITER bugs: alt accept / cheat reject (`RESULT.md`, each `ITER_*.md`, `TWO_REPO_TASK.md`) |

Checksum guards (`*_test.go`, consumer module) and hidden guard tests
(`TestPDOracle_UntilExpired`, `TestContainsByEnd`, …) were added **before**
Grok ran, to block (a) in advance. No post-hoc widening after a Grok miss.

## 5. IRT view

Bank (`irt_summary.md`): 36,015 tasks, 7 combos, `P = sigmoid(a (θ − b))`.
θ_2pl: minisweagent/qwen38_27b **+0.180** … openhands/qwen35_122b **−1.050**.
b cuts (quantile-matched to pass-rate buckets): all_pass ≤ **−1.213**, easy ≤
**−0.517**, mid ≤ **+0.191**, hard ≤ **+0.894**, all_fail above that.
Median b_2pl **+0.112**. Spearman(b_2pl, −solve_rate) +0.9507.

Pass-rate → b (same cuts): solve_rate 1.0 → all_pass (b ≤ −1.213); ~0.5 → near
median b +0.112 (mid); 0 → all_fail (b > +0.894).

**Abilities (synthetic, not bank-refit):**

| solver | scored n | rate | link |
|---|---:|---:|---|
| Grok 4.6 xhigh | 25/25 | 1.000 | stronger than every task attempted |
| Nex-N2.5-Pro free | 5/6 (plus 12× d) | 0.833 scored; 0.278 of 18 trials | cheaper prober; not bank-calibrated |
| 7 teachers | 280,116 rollouts | 0.318–0.538 | θ in [−1.050, +0.180] |

**Shared tasks** (Grok and Nex both scored):

| task | Grok | Nex | rate | mapped b bucket |
|---|---|---|---:|---|
| twosumbest / twosumbrute / match | pass | pass | 1.00 | all_pass, b ≤ −1.213 |
| isfakeregionerror | pass | pass | 1.00 | all_pass, b ≤ −1.213 |
| newrequest | pass | pass | 1.00 | all_pass, b ≤ −1.213 |
| iserrnotfound | pass | fail | 0.50 | mid, b ∈ (−0.517, +0.191] (median +0.112) |

Grok-only tasks (rungs 2–8 + two-repo): rate 1.0 **for Grok** is not a bank
all_pass. `irt_summary.md` §3: when strong combos dominate, the pass-rate
bucket reads easier than b (2,456 instances, 72.0% strong-combo share). Mapping
those 19 tasks to b ≤ −1.213 would repeat that bias.

**Where Grok’s θ sits vs the 7 teachers:** lower bound θ_Grok > b_IsErrNotFound
≈ +0.112, hence **above median bank b**, and Grok’s 25/25 on constructions the
teachers never saw is **above the strongest teacher θ +0.180** as a rank
statement, not a fitted magnitude (no overlapping bank items; 1PL location is
prior-anchored). Nex: θ_Nex > −1.213 (passed all_pass-mapped daily + two 1-hop
bugs) and θ_Nex < b_IsErrNotFound ≈ +0.112, which covers almost the whole
teacher band; 429s block a tighter place. Nex is **not** the calibrated cheap
prober in the workflow (`minisweagent/qwen36_27b`, θ_2pl = −0.605).

## 6. Recipe for arbitrary repos

CLI (registered in `pyproject.toml` `[project.scripts]` as `openswe-synth` →
`openswe_traces.synth.difficulty:main`; thin wrapper `scripts/openswe_synth.py`):

```bash
# 1. Index
codegraph init -y <repo> && codegraph index <repo>

# 2. Design + validate (writes <out>/validation.json; exit 0 if checks pass)
uv run openswe-synth \
  --repo <repo> --rung R --hops H --sites S --decoys D \
  --cross-module --guard --out <out>

# 3. Manual: write a 1–N line (or excision) patch at the designed site(s),
#    plus .alt.patch and .cheat.patch.

# 4. Harbor task dir + validation.json (instruction self-check included)
uv run openswe-synth \
  --repo <repo> --rung R --hops H --sites S --decoys D \
  --cross-module --guard --locality 0|1|2 \
  --patch <Symbol>.patch --f2p TestFoo --out <out> --base-commit HEAD
```

`--out` always writes `validation.json` (`ok`, `checks`, design, resolved
guards). `--patch` also writes `instruction.md`, `task.toml`,
`environment/`, `tests/test.sh` (checksums on `*_test.go`). `--guard` pulls
existing tests from the graph; `--guard-tests` overrides names.
`--cross-module` requires a two-site pair with no import edge (or a multi-file
excision). Rung 8: `--keep-interface` / `--no-keep-interface`.
Rung 4 library→consumer: `uv run python scripts/two_repo.py build …`
(`src/openswe_traces/synth/two_repo.py`) — parent index, not a second repo
root.

**Codegraph provides:** callers / reverse-`calls` hops, impact files, two-site
homonyms, decoy ranking (path intermediates + same-pkg siblings + inverse
pairs), guard-test BFS, sequence-named tests, callee closure for excision,
import-edge absence. **Needs dynamic analysis / runtime:** interface dispatch
(`Oracle.GetTimestamp` → `localOracle.GetTimestamp`; hops `None` in sqlite),
testify suite methods that are not `Test*` callers (DecodeBucketKeys
sparsity), coverage (`go test -coverprofile` → `find_sparse_branches`),
fail-to-pass discovery (`go test` on the patched tree), alt/cheat, flakes,
Docker buggy-fail / gold-pass. **Still manual:** choosing the mutation (the
CLI does not inject), writing alt/cheat, confirming decoy-fix-fails on an
**existing** test, L2 issue prose that does not name the symbol, two-repo
consumer pairing, skipping hang-prone symbols (`EvalFailpoint`).

Sanity before any rollout (`synthetic_task_difficulty_workflow.md`): gold
patch passes, empty patch fails, instruction `ok` (no leaked symbol/file).

## 7. Next steps

**Weaker solver, mid data:** run the bank-calibrated cheap prober
`minisweagent/qwen36_27b` (θ_2pl = −0.605) or paid Nex on **rung 3**
(`client-go-dualexpo`: hops 5, two-site, sparse) and **rung 5**
(`client-go-extractphysical`: decoy + guard). Daily is too easy (Nex 3/3).
Rung 8 is the wrong next probe: Grok already saturates it, so a weaker model
going 0/n only yields all_fail (b > +0.894) with no mid mass. Fisher
information is highest when θ ≈ b; DualExpo / ExtractPhysical sit in the
bracket between Nex’s mixed 1-hop cell and Grok’s empty fail region.

**Add for missing classes:**

- **Concurrency:** `go test -race` f2p; mutations on shared maps / memdb
  overwrite already exist as sequence (MemSetValue) but not as races. Skip
  failpoint/retry-infra symbols (`EvalFailpoint` hung ~6 min,
  `codegraph_bugs/RESULT.md`).
- **Config:** default-vs-override across packages. `GetGlobalConfig` was rung 1
  hops 2 and Grok-pass in 4.66 min — need a config that library tests never
  flip (the 1PC `enable1PC` pattern) plus a consumer or hidden guard.
- **Env:** flag/env parsing (`WITH_TIKV` / `*withTiKV` skips). Do not use
  skip-gated tests as f2p; the two-repo task already avoided live PD/TiKV.

Reproduce synth checks: `uv run pytest tests/test_synth_difficulty.py` ;
`uv run ruff check src/openswe_traces/synth/difficulty.py tests/test_synth_difficulty.py`.
