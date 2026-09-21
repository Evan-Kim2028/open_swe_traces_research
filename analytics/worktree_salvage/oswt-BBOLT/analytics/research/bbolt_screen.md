# bbolt cohort screen

Date: 2026-09-20. Worktree `oswt-BBOLT` (branch `bboltscreen`). Packaged units at
`oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt` (25 families, 50 dirs).
Gates ran cheapest-first per `analytics/research/PIPELINE.md`. No solver trials.

Staged for the parent:

- L0: 25/25 at `experiments/dose_response/sweep_bbolt/`
- L2: 20/25 at `experiments/dose_response/sweep_bbolt_L2/`
- Held (repair, not trial): `bucket-L2`, `cmdsurgerymeta-L2`, `db-L2`, `flshared-L2`, `meta-L2`

Clean L2 audit rows were appended to `experiments/dose_response/audit/gap_read.jsonl`
so `sweep_seq.sh` will not refuse them.

## Trials the gates saved

A trial is 2.08M tokens. `sweep_seq.sh` is k=1 sequential, three rounds, so a unit
that cannot flip costs **3 trials**.

First gap read found contract defects on **21 of 25** L2 units (4 were already
`NONE`). Sending those 21 unrepaired through three rounds would have been
**63 wasted trials (~131M tokens)**. That is the comparison against screening
all 25 L2 units blind.

What we did instead:

| bucket | n | fate |
|---|---:|---|
| L2 already `NONE` | 4 | staged |
| L2 gapped, repaired to `NONE` | 16 | staged |
| L2 still gapped after two repair rounds | 5 | **held** (saves 5×3 = 15 trials if they cannot flip) |
| L0 TOO-EASY | 0 | all 25 staged |
| L2 dropped (B10 / A12 / SCRUB) | 0 | none |

The 16 repaired units would have been 48 of those 63 wasted trials. They are now
fair enough to trial. The 5 held ones still would not flip as written.

**Low-discrimination even if preflight-clean.** `db-L2` (5 ARBITRARY coverage-table
mismatches on the latest read) and `meta-L2` (Print output is a byte-exact template
the suite grades and the contract should not copy). `cmdsurgerymeta-L2` still has
one ARBITRARY warning-substring gap. None of those three should be trialled until
the *test* is weakened (ARBITRARY) or the remaining DERIVABLE/COUNTER rows are
added without a coverage-table lie.

## Gate 1. Lint (`scripts/ops/task_lint.py --batch`)

**0 of 50 blocked.** No B10 digest oracle, no A12 gold-touches-tests, no SCRUB
`"the call"` placeholder, no TOO-EASY (`assertions<=8 AND testfns<=5`). Smallest
suite is `inbucket` (13 assertions / 5 tests) and `inode` (16 / 7).

Surgeon-L2 did **not** carry an unresolved `"the call"` placeholder. Lint's
`\bthe call\b` is clean. The nearby English is "the caller" in the freelist
paragraph. No edit.

INFO/WARN only (advisory, do not block):

- COVERAGE row-count heuristic on 19 L2 units. Refuted as a flip predictor.
  Semantic gap read is the real coverage gate.
- B7 file names in L2 prose (`cmddump`, `cmdpage`, `cmdpages`, `cmdutils`, `compact`).
- UNGROUNDED example literals on several L2 units.

Final lint after repairs is still 0 blocked.

## Gate 2. Reachability

`scripts/ops/orphan_scan_batch.py` (cgscan + `gap_deterministic` = reached ∩ gold − implied).
Advisory.

| metric on 25 L2 units | mean |
|---|---:|
| `gap_deterministic` (cg_coverage orphans) | 4.56 |
| gold exported funcs the hidden suite never *names* | 2.52 |

The early "mean 0.80, only page and flshared risky" figure used the unnamed-gold-fn
count after dropping `sort.Interface` methods. Under that filter:

- `page-L2`: 11 unnamed gold funcs of 57 (Len/Less/Swap ×2, `NewPage`,
  `NewLeafPageElement`, three `Unsafe*` helpers).
- `flshared-L2`: 3 unnamed after dropping `txIDx` Len/Less/Swap
  (`IsFreelistPage`, `Typ`, `Mergepgids`).

**page-L2 repair.** Hidden tests constructed pages via `LoadPage`, never `NewPage`.
Added `TestDetail13_NewPageHeader` (round-trip id/flags/count/overflow). Checksums
and `-run` lists updated on L0 and L2. `NewLeafPageElement` returns an unexported
type; `Unsafe*` and `sort.Interface` are same-file plumbing. Left untested.

**flshared-L2 cannot be repaired at the symbol level.** The three unnamed names
are page helpers gold.patch references, not freelist API. Adding page-type tests
inside a freelist unit would expand the excision. The remaining defect is a
COUNTER gap on release-range bounds (see Gate 3), not those symbols.

## Gate 3. Contract gap read (`ask_composer`, model composer-2.5)

25/25 L2 units. First pass:

| first-read | n |
|---|---:|
| `NONE` (cmdutils, compact, surgeon, verifyenv) | 4 |
| gapped, mostly DERIVABLE/COUNTER | 21 |
| majority-ARBITRARY (low-discrimination) | 0 |

ARBITRARY items weakened in the **test**, never written into the contract:

| unit | what | repair |
|---|---|---|
| `bucket-L2` | `tx.Bucket(name) ==` pointer identity | require non-nil lookup |
| `node-L2` | file size `<= 2×` after delete+refill | dropped the numeric bound; data-intact remains |
| `cmdsurgerymeta-L2` | exact `already exists` / `invalid key-value pair` / WARNING template / `invalid meta page id: N` | error-is-non-nil and shape (`warning` + `page size`) |

DERIVABLE/COUNTER: added prose. Composer full-rewrites that raised the literal
count were refused (`A3-SURFACE`). Those units got unquoted "Derived commitments"
paragraphs instead. Coverage-table lies on `bucket` and `page` were rewritten to
match the actual `TestDetailNN` names.

Second and third gap reads (fresh cache keys `gapread-r2` / `gapread-r3`):

| after repair | n | units |
|---|---:|---|
| `NONE` | 20 | staged |
| still gapped | 5 | held |

`db-L2` got *worse* on re-read (coverage-table mapping vs hidden tests). That is
model variance plus a real table lie. Do not trial it.

## Gate 4. In-image preflight

`scripts/preflight_task.py` from the closure-O gate (bare FAIL, gold PASS, cheat FAIL).

**50/50 PASS.** 38 were cache hits from the verifier's executed A1/A3/A8. The 12
cmd units lacked a `preflight` block and still cache-hit on the same image+digest.
Re-ran docker for the four families whose hidden tests changed (bucket, node,
cmdsurgerymeta, page). All eight dirs PASS (50–76s each).

## Per-unit recommendations

Lint findings are INFO/WARN only. Orphans = `gap_deterministic` / gold exported
funcs. `g1` is the first gap read (`missing/conflicts`, kinds a/d/c). `gF` is the
latest read. Rec is `trial` / `repair` / `drop`.

| unit | lint | orphans | unnamed | g1 | gF | preflight | rec |
|---|---|---:|---:|---|---|---|---|
| bucket-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| bucket-L2 | COVERAGE | 0/36 | 11 | 6/1 a1d5c1 | 7/1 a0d7c1 | PASS | **repair** |
| cmddump-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmddump-L2 | COVERAGE, UNGROUNDED, B7 | 6/1 | 1 | 2/1 a0d3c0 | 0/0 | PASS | **trial** |
| cmdget-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmdget-L2 | COVERAGE | 0/4 | 1 | 3/0 a0d3c0 | 0/0 | PASS | **trial** |
| cmdpage-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmdpage-L2 | COVERAGE, B7 | 19/5 | 5 | 7/1 a0d7c1 | 0/0 | PASS | **trial** |
| cmdpages-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmdpages-L2 | COVERAGE, UNGROUNDED, B7 | 5/5 | 0 | 8/0 a0d8c0 | 0/0 | PASS | **trial** |
| cmdsurgerymeta-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmdsurgerymeta-L2 | COVERAGE | 0/23 | 11 | 8/1 a4d4c1 | 5/0 a1d3c1 | PASS | **repair** |
| cmdutils-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cmdutils-L2 | COVERAGE, UNGROUNDED, B7 | 12/4 | 0 | 0/0 | 0/0 | PASS | **trial** |
| compact-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| compact-L2 | COVERAGE, B7 | 0/6 | 2 | 0/0 | 0/0 | PASS | **trial** |
| cursor-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| cursor-L2 | ok | 0/6 | 0 | 2/1 a0d2c1 | 0/0 | PASS | **trial** |
| db-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| db-L2 | COVERAGE, UNGROUNDED | 0/20 | 5 | 2/1 a0d2c1 | 11/6 a5d9c3 | PASS | **repair** (low-disc) |
| flarray-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| flarray-L2 | COVERAGE | 0/8 | 1 | 16/4 a0d16c4 | 0/0 | PASS | **trial** |
| flhashmap-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| flhashmap-L2 | ok | 0/7 | 0 | 7/0 a0d6c1 | 0/0 | PASS | **trial** |
| flshared-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| flshared-L2 | COVERAGE, UNGROUNDED | 0/28 | 6 | 10/0 a0d9c1 | 1/1 a0d0c2 | PASS | **repair** (COUNTER, high-value) |
| gutscli-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| gutscli-L2 | COVERAGE, UNGROUNDED | 9/9 | 1 | 2/0 a0d2c0 | 0/0 | PASS | **trial** |
| inbucket-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| inbucket-L2 | ok | 8/9 | 0 | 2/1 a0d2c1 | 0/0 | PASS | **trial** |
| inode-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| inode-L2 | ok | 1/19 | 3 | 1/0 a0d1c0 | 0/0 | PASS | **trial** |
| loadutil-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| loadutil-L2 | UNGROUNDED | 4/4 | 0 | 3/0 a0d3c0 | 0/0 | PASS | **trial** |
| meta-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| meta-L2 | ok | 4/30 | 0 | 3/0 a0d2c1 | 1/1 a0d2c0 | PASS | **repair** (Print template) |
| node-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| node-L2 | ok | 0/0 | 0 | 6/0 a1d5c0 | 0/0 | PASS | **trial** |
| page-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| page-L2 | ok | 33/57 | 11 | 8/0 a0d7c1 | 0/0 | PASS | **trial** |
| surgeon-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| surgeon-L2 | COVERAGE | 2/14 | 5 | 0/0 | 0/0 | PASS | **trial** |
| tx-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| tx-L2 | ok | 0/8 | 0 | 7/1 a0d6c1 | 0/0 | PASS | **trial** |
| txcheck-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| txcheck-L2 | ok | 3/6 | 0 | 2/1 a0d2c1 | 0/0 | PASS | **trial** |
| verifyenv-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| verifyenv-L2 | COVERAGE | 8/6 | 0 | 0/0 | 0/0 | PASS | **trial** |
| xray-L0 | ok | n/a | n/a | n/a | n/a | PASS | **trial** |
| xray-L2 | COVERAGE | 0/8 | 0 | 2/2 a0d2c2 | 0/0 | PASS | **trial** |

No unit is `drop`. Nothing is B10.

## Held L2, what is left

**bucket-L2.** ForEach/MoveBucket tx wrappers, MaxKeySize inclusive bound, sentinel
`errors.Is` for same-bucket / different-db, parent Stats KeyN counting child names.
Composer rewrite of this contract was reverted after it introduced 10 ARBITRARY
coverage-table conflicts.

**cmdsurgerymeta-L2.** Empty `--fields` still copies and can print "Nothing changed!";
Validate vs parseFields split semantics (COUNTER); remaining ARBITRARY `match`
substring from the weakened warning test.

**db-L2.** Dual-meta failover, version/checksum vs generic invalid (COUNTER), timeout
against read-only shared locks, OpenTxN on rollback, MaxSize, OpenFile hook. Plus
five ARBITRARY coverage-table rows that name the wrong TestDetail. **Low-discrimination
until the table is rewritten the way bucket/page were.** Would pass preflight today.

**flshared-L2.** Two COUNTER rows on the above-last-reader alloc-txid interval
(lower bound is the *greatest* registered readonly txid, and range release also
requires the pending map key ≤ range end). High-value. Symbol-level orphans are
page helpers and cannot be fixed in this unit.

**meta-L2.** `Print` is a byte-exact eight-line template. That is a test defect.
Weaken the assertion to field identity, do not paste the template into the contract.

## Reproduce

```bash
# lint
uv run python scripts/ops/task_lint.py --batch \
  /home/evan/Documents/oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt

# orphans (L2)
uv run python scripts/ops/orphan_scan_batch.py \
  --batch /home/evan/Documents/oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt \
  --out outputs/bbolt_orphans.jsonl --l2-only

# gap read (Composer, cached)
uv run python scripts/ops/contract_gap_read.py outputs/bbolt_gap_read.jsonl \
  /home/evan/Documents/oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt/*-L2

# preflight (from oswt-closureO)
uv run python scripts/preflight_task.py \
  /home/evan/Documents/oswt-VFnew2/experiments/pipeline/tasks_batch2/bbolt/*-L{0,2}
```

Parent trials (not run here): `scripts/ops/sweep_seq.sh sweep_bbolt` then
`scripts/ops/sweep_seq.sh sweep_bbolt_L2`.

Artifacts: `outputs/BBOLT.log`, `outputs/bbolt_orphans.jsonl`,
`outputs/bbolt_gap_read.jsonl`, `outputs/gaps/`, `outputs/preflight/`,
`.audit/bbolt-screen.tsv`.
