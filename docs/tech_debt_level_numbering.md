# Tech debt: ladder level numbering

The code numbers the ladder L0 to L6 and keeps an L1 "partial description" rung. The write-up
numbers it L1 to L6 with no partial rung. Fix the code once the Stage 1 experiments finish, not
while runs are in flight, because rung keys are baked into job directories and ledger rows.

## Why the write-up differs

The partial description is poorly defined. Which requirement to withhold is a free choice, so two
L1 prompts for the same task need not be comparable, and the ladder's ordering argument (each level
contains the one below it, Blackwell) holds without it. The climb already goes straight from the
bug report to the full description. Numbering the bug report L1 makes the published ladder
contiguous.

| Meaning | Code | Write-up |
|---|---|---|
| bug report | `0`, `*-L0` | L1 |
| partial description | `1`, `*-L1`, `sweep_L1` (16 Composer runs, $4.00) | not shown |
| full description, test names, signatures, one test, all tests | `2` to `6` | L2 to L6 |

## Where the old numbering lives

- Task directories: `experiments/dose_response/*/<task>-L0` and `-L1`.
- Ledger rung keys in `scripts/ops/trial_ledger.py`, plus the many ops scripts that compare rungs
  against `"0"` (`rg -l 'L0|"0"' scripts`).
- `docs/ladder_construction_spec.md` (28 mentions of L0).
- The writings site maps code rung `0` to L1 in `scripts/information-gap-figures.py`
  (`post_level`) and drops rung `1`. Delete that mapping once the code is renumbered.

## Fix

1. Retire rung `1`: stop authoring `-L1` prompts, and check that the 16 existing runs never set a
   certificate level.
2. Renumber rung `0` to `1` in the ledger schema and task directory names, with a one-shot migration
   over existing job results.
3. Update the spec and ops scripts, then remove `post_level` from the site script.
