# Ladder-2b: black-box property-1pc

Date: 2026-09-18. Rewrite of the ladder-2 `property-1pc` unit. The A0–A4
tasks under `tasks_ladder2/property-1pc-A*` shipped a **white-box** property
suite (`checkAsyncCommit` / `checkOnePC` on the unexported committer), so
rule B4 failed and every Harbor result on that family is void.

This rebuild drives the same decision only through the exported
transaction/commit API and observable probe state: `NewTiKVTxn`,
`SetEnable1PC` / `SetEnableAsyncCommit` / `SetScope` / `SetBinlogExecutor` /
`SetCommitTSUpperBoundCheck`, `Set`, `TxnProbe.NewCommitter`, then
`CommitterProbe.CheckOnePC` / `CheckAsyncCommit` (and mutation size via
`GetMutations`). No unexported production names in the hidden suite.

Base tree: `experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src`.
Dest: `experiments/harbor_nex/tasks_ladder2b/property-1pc-A{0..4}/`.
No Harbor jobs were launched.

Build:

```
uv run python scripts/build_ladder2b.py \
  --src experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-A0/environment/src \
  --dest experiments/harbor_nex/tasks_ladder2b
```

## `property-1pc`

- **Family:** property (black-box rewrite of ladder-2 `property-1pc`)
- **Entry:** `SetEnable1PC`
- **Closure:** 8 functions, 96 lines — `checkAsyncCommit`, `checkOnePC`, `shouldWriteBinlog`, `setOnePC`, `setAsyncCommit`, `isOnePC`, `isAsyncCommit`, `checkOnePCFallBack`
- **Existing tests:** `TestOnePC`, `TestAsyncCommit`
- **Package:** `txnkv/transaction`
- **Hidden:** `txnkv/transaction/onepc_bb_prop_test.go` (seed 20260918, ≥10k cases)

Design. Production bodies of `checkOnePC` / `checkAsyncCommit` are stubbed
in the task tree (enable-flag only). gold/alt/cheat patches restore or
distort only `txnkv/transaction/2pc.go` (rule A12: no `*_test.go` hunks).
`test_probe.go` gains exported `CheckOnePC` next to the existing
`CheckAsyncCommit` so the hidden suite never names the unexported methods.
Instruction is a prose contract at locality L2 plus coverage sentences
(no `Test*` names). A1 names the hidden tests; A2 adds a package godoc
hint; A3 restores one hidden file into the tree; A4 restores all.
`build_affordance_levels` refuses to package if B4 is not a pass.

Coverage table (hidden check → sentence):

| hidden check | sentence |
|---|---|
| `TestOnePCAsyncDecisionProperty` | One-phase and async-commit are refused for a non-global scope, a commit-ts bound check, or a binlog; async-commit is also refused when the mutation count or total key size exceeds the configured limits. |
| `TestOnePCContractExamples` | Global scope, flags on, no binlog, no bound check: both protocols allowed. Local scope, binlog present, or a bound check: both refused. |
| `TestOnePCUnmentionedRandom` | Random key sets that are not the single-letter contract example still follow the same decision; hardcoding that one key fails. |

Validation (built image, every affordance level; harness checked first, C5):

| check | result |
|---|---|
| `proof_harness` | pass |
| `harness_ok` | pass |
| `A0_buggy_fails` | pass |
| `A0_gold_pass` | FAIL |
| `gold_restore` | FAIL |
| `buggy_fails` | pass |
| `A0_cheat_fails` | pass |
| `cheat_rejected` | pass |
| `two_func_alt_accepted` | FAIL |
| `A1_buggy_fails` | pass |
| `A1_gold_pass` | FAIL |
| `A1_cheat_fails` | pass |
| `A2_buggy_fails` | pass |
| `A2_gold_pass` | FAIL |
| `A2_cheat_fails` | pass |
| `A3_buggy_fails` | pass |
| `A3_gold_pass` | FAIL |
| `A3_cheat_fails` | pass |
| `A4_buggy_fails` | pass |
| `A4_gold_pass` | FAIL |
| `A4_cheat_fails` | pass |
| `patches_skip_tests` | pass |
| `blackbox_hygiene` | pass |

## Rules

Each task dir has `validation.json` with `rule_verdicts` from
`src/openswe_traces/synth/rules.py` (A1–A12, B1–B8, C5).
B4 must be pass before `validation.json` is written; packaging raises
`B4PackagingError` otherwise.
Proof harness (C5) was validated before trusting gold REWARD: `go` on PATH,
`false` exits non-zero, `true` exits zero.

