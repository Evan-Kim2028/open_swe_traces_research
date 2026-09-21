# The pipeline, as it actually runs

*Rewritten 2026-09-20 to match reality. Diagram: `analytics/research/pipeline_flow.html`.
Dashboard: `analytics/research/dashboard.html`. Rules: `analytics/research/verifier_rules.md`.*

## Stages

Three agents and two mechanical gates. The **information barriers** are the design, not the stage
count — an agent that can see the answer writes it into the question.

| # | stage | kind | reads | must NOT read | produces |
|---|---|---|---|---|---|
| 1 | **Author** | agent | the repo | — | `gold.patch`, `cheat.patch`, `api.md`, `DETAILS.md`, `bugreport.md` (L0) |
| 1b | **DETAILS gate** | mechanical + 1 call | `DETAILS.md` | — | per-line `grade` / `grade-shape-only` / `drop` |
| 2 | **Verifier** | agent | `api.md`, `DETAILS.md`, excised tree | **gold, contract, bugreport** | `tests/hidden/TestDetail01…NN` |
| 3 | **Reconciler** | agent | gold + the hidden suite | `bugreport.md` | `contract.md` (L2), one row per assertion |
| 4 | **Preflight** | mechanical | the built image | — | bare FAIL · gold PASS · cheat FAIL |
| 5 | **Audit** | mechanical + 1 call | contract vs suite | — | missing/conflicting commitments, each labelled |

Stage 1b is new and is the highest-leverage one: the verifier is blind to gold, so it grades only
what `DETAILS.md` lists. An arbitrary line there becomes an assertion no solver can derive, and
**no downstream stage can catch it because nothing downstream did anything wrong.**

## Gate order — cheapest first, and enforced in code

| order | gate | cost | enforced by |
|---|---|---|---|
| 1 | deterministic lint (B10 digest oracle, A12, scrub placeholder, TOO-EASY) | free | `scripts/ops/task_lint.py` |
| 2 | reachability / orphan symbols | free | `scripts/cg_coverage.py` — **advisory**, 50% precision |
| 3 | **contract gap read** | **~6k tokens** | `scripts/ops/contract_gap_read.py` |
| 4 | shadow implementation | ~1 weak-model call | `scripts/shadow_gate.py` |
| 5 | in-image preflight | one docker run | preflight |
| 6 | **trial** | **2.08M tokens, $0.47** | `scripts/ops/sweep_seq.sh` |

**`sweep_seq.sh` now refuses to trial an L2 unit that has no audit row.** It runs the gap read
first and drops any unit the audit says cannot flip as written. L0 and L1 are exempt — there is no
contract at those rungs, so TOO-EASY is the only applicable gate.

`post_sweep.sh` closes every sweep: B9 hack audit, durable trial archive, and a fail-closed
`verdicts.json` where a unit with no valid trial is `UNKNOWN`, never `0`.

## Repair, and how a fix is verified

An audit is a diagnosis, not a fix, and a single repair round is one iteration — the hand ladder on
`helm-repindex` needed **five**: 0/3 → 0/3 → 1/3 → **0/3 (two wrong rows introduced)** → 2/3.

Route by where the answer lives. **Never repair uniformly.**

| kind | broken | action |
|---|---|---|
| ARBITRARY (17%) | the **test** over-specifies | weaken the assertion to shape; never write the literal into the contract |
| DERIVABLE (54%) | the contract under-specifies | add the row |
| COUNTER (28%) | the contract under-specifies | add the row, **flag high-value** |

Three verification layers, each cheaper than the next:

1. **re-audit** — the gap list must shrink (~6k tokens)
2. **literal budget** — `A3-SURFACE`: literal count must not rise, grounded fraction must not fall
3. **preflight with a regenerated cheat** — `cheat.patch` is otherwise byte-identical after repair,
   so A3 would be re-checked against an adversary that predates the fix

Only then a trial.

## Solver split

| rung | solver | why |
|---|---|---|
| L0 screen | Devin (target) | "is this too easy" — a second model makes the dataset better, and Devin is ~3× cheaper per trial |
| L2 confirm | Composer | the flip certificate's meaning depends on the screener; changing it breaks comparability with the existing dataset |

Devin solves 2 of 7 units Composer fails at L0 — they are **differently-abled, not ordered**, so run
both on an overlap set rather than assuming equivalence. Harbor's devin agent needs
`api.devin.ai`, `*.devin.ai`, `*.cognition.ai` in the task's `allowed_hosts`; without them the
container cannot reach the API and the CLI reports `Unknown model: 'swe-2-max'` with an empty list.

## Roadmap

1. **Prove repair recovers units.** 36 of 38 double-failures have a named contract defect. Repair
   and re-trial is the open test of the whole thesis.
2. **Go to 300–400 certified**, with the gate stack enforced end to end.
3. **Then Rust in parallel.** Same properties that make Go work: one canonical test runner
   (`cargo test`), unambiguous compile failure, and `syn` gives the parse-level reachability
   `go/parser` gives today. The excision machinery ports; `test.sh`, the hack-audit patterns and
   the `TestDetailNN` convention need per-language variants.
4. **1000 total tasks** across both languages.

Dart/Flutter is not a near-term candidate: widget-tree behaviour does not decompose into
excise-a-function-and-assert-its-behaviour, which is what the whole construction depends on.
