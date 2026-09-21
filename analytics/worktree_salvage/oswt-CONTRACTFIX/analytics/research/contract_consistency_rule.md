# A13 — contract-vs-gold consistency as a validity rule

2026-09-20. Companion to `analytics/research/sweep_L0_2026-09-19.md` (oswt-LIE),
sections "BANK-WIDE CONTRACT AUDIT" and "ADVERSARIAL CONTRACT".

## Why this rule exists

Measured, not assumed: inverting ONE coverage row in a passing unit's contract
flipped 7 of 8 units from pass to fail, and every one failed on exactly the
inverted property. The contract outranks the code the solver can read — a wrong
contract manufactures failures the solver cannot dodge by reading the tree.

Independently, a solver-free audit (`oswt-LIE/outputs/contract_gold_dump/`,
per-unit JSON) found **20 of 50 bank units** whose contract misdescribes gold:

- 11 units with at least one FALSE coverage row (the claim is untrue of
  `gold.patch`);
- 9 units missing a row (and usually prose) for an assertion the hidden suite
  makes.

Four of those defective units had already consumed 24 solver trials and $28 on
the ladder before the audit caught the contradiction — the trials were failing
against a contract that could never be satisfied. Those 20 units' results are
provisional until re-screened; the usable bank under the old contracts was 30,
not 50.

## The rule

> **A13 (validity):** The contract must be consistent with gold: every coverage
> row's claim is true of `gold.patch`, and every assertion the hidden suite
> makes is covered by a row. Vague rows are surfaced, not failed.

Documented in `analytics/research/verifier_rules.md` (A-series), tier `static`,
enforced at packaging/gate time by `openswe_traces.gate.contract_gold`.

## Check design

Two layers; the gate never performs network access at gate time.

**Deterministic** (`contract_gold.deterministic_findings`):

- parses coverage rows, including rows keyed by source-file labels
  (e.g. `` `2pc_test.go committer suite` `` — an earlier `Test*`-only filter
  silently skipped whole contracts);
- maps each hidden test to covering rows/contract sentences (`missing`
  suspects);
- flags quoted identifiers/literals in a row absent from `gold.patch`
  (`false`/`suspect` suspects);
- `vague` rows (topic-level, no checkable claim) are reported, never fail.
- unresolved deterministic suspects fail closed — a suspect with no cached
  judgment is a gate failure until the judge resolves it.

**LLM judge** (`openswe_traces.contract_judge`, driver
`scripts/contract_consistency_audit.py`):

- judges (row prose, gold hunk, hidden assertion) over the OpenRouter free
  tier (key in `.env`; models `nvidia/nemotron-3-super-120b-a12b:free`,
  `nvidia/nemotron-3-ultra-550b-a55b:free`, `nex-agi/nex-n2.5-pro:free`);
- budget ≤300 requests; results cached under `outputs/contract_judge/` keyed
  by evidence hash — editing a contract changes the key and re-judges only
  that unit; resume-safe;
- returns per-row `true|false` and `missing` hidden assertions.

Packaging-time enforcement: `scripts/gate_tasks.py` sweeps the gate over task
dirs (`--no-executed` for static+documentary only); `pipeline/preflight.py`
delegates executed rules to `gate.exec_rules`. The documentary rule path
(`synth/rules.py`) was fixed so documentary validation can no longer overwrite
executed/static verdicts — verdicts merge under tier precedence.

## Calibration vs the 20 hand-found defects (pre-repair bank)

50/50 contracts judged, 68 requests total.

| metric | result |
|---|---|
| defective units flagged (of 20) | 19/20 (missed `kops/templater`, missing-only) |
| clean units flagged (of 30) | 13/30 — judge over-reports `missing` |
| per-defect recall: false rows | 11/17 |
| per-defect recall: missing assertions | 12/19 |

Judge failure modes observed:

- **Missing-assertion overreach** (the dominant error): the judge invents
  hidden assertions the prose doesn't literally enumerate — e.g. flagging
  clean units whose coverage is table-level rather than sentence-level.
  These land as `missing` findings on clean units.
- **Patch-invisible claims** → false `false`: `client-go/doactionbatches`'s
  "region errors regroup" row describes retry machinery that lives in the
  un-excised upstream tree, invisible in `gold.patch`. The hand audit counted
  the unit clean; the judge cannot see the evidence and votes false.
  Mitigation: none automatic — a documented class. The correct response is
  NOT to reword a true row.
- **Semantic misread**: `client-go/onregionerror` — gold returns
  `(false, nil)` from the region-error handler meaning "stop retrying, return
  the response (which carries the region error)". The judge read it as
  "treated as successful". The contract was correct; wording was sharpened
  ("the handler stops retrying and the response carrying the region error is
  returned to the caller") to make the terminal path legible.

Verdict: the check reproduces 19/20 defective units but is not a clean
pass/fail oracle — 13/30 clean over-flagging means a `missing` finding needs
human arbitration, while `false` findings are high-precision (11/17 recall,
near-zero false positives aside from the patch-invisibility class). The gate
fails closed on unresolved deterministic suspects and surfaces judge findings
for review.

## Repairs (contract prose only — gold.patch and hidden tests untouched)

Canonical contract: `experiments/pipeline/authored/<repo>/<unit>/_author/contract.md`.
Propagated to `tasks_composerver/<repo>/<unit>-L{2,5,6}/instruction.md` by
splicing the contract prefix before the `Reproduce with:` marker
(`package.refresh_contract_instructions`); L0 instructions are bug reports
and are unchanged.

| unit | defect | repair |
|---|---|---|
| helm/chartloader | 2 false rows | dev/null & irregular files: aborts with an error (was "skipped"); archive backslashes: load after path normalization (was "rejected") |
| helm/depresolver | 1 false + 1 missing | hash row now describes `HashReq`/`HashV2Req` gold behavior; added `TestDRHashV2Stable` coverage |
| helm/ignorerules | 1 false | `**` is rejected, not supported |
| helm/kindsorter | 2 false + 1 missing | hook weight ordering corrected; name-sort tie-break corrected; added unknown hook-event drop + `_`-prefixed partial skip rows |
| helm/memorydriver | 2 false + 1 missing | List/Query rows corrected to gold semantics; adversarial-property coverage added |
| helm/repindex | 3 false + 1 missing | duplicate add appends; empty/malformed index handled; `MustAdd` validation row added |
| helm/storage | 2 false + 1 missing | `Create` returns pruning errors (except not-found); `ListReleases` returns everything; adversarial-property row added |
| helm/strvalsparser | 1 missing | contract-table property coverage added |
| kops/clustervalid | 1 missing | constructor requires ≥1 InstanceGroup |
| kops/templater | 1 missing | missing-include error behavior added |
| gin/bodydecoders | 1 false | BSON row corrected; custom codec/protobuf/plain-binder details added |
| gin/htmlrender | 1 missing | content-type, empty-name, debug-source, panic coverage added |
| gin/streamrenders | 1 false | redirect status range + content-length/content-type corrected |
| goa/httpxray | 1 false + 1 missing | URL query stripped in recorded request data; record-request property row added |
| goa/xrayseg | 1 false + 1 missing | subsegment row corrected; `AddAnnotation` coverage added |
| client-go/lockresolver | 2 missing | `ExtractLockFromKeyErr` row added; `GetSecondariesFromTxnStatus` returns all secondary keys (row + prose, round 2) |
| client-go/memdbstaging | 1 missing | nil/empty `Set`/`SetWithFlags` errors added |
| client-go/onregionerror | 2 missing | terminal RegionNotFound/KeyNotInRegion; StoreNotMatch closes the target connection (rows added; terminal wording sharpened round 2) |
| client-go/pdoracle | 1 missing | invalid `prevSecond` errors added |
| client-go/replicaselector | 2 missing | WithMatchLabels stale first-hop row added; round 2: mixed reads leader-first, stale first-hop sets stale not replica-read, stale failover advances across stores |

Post-repair A13 re-runs (judge passes 2–4 over repaired contracts) surfaced
real residual gaps beyond the audit's 20 — each confirmed against the hidden
test and gold before editing:

- **round 2:** replicaselector's mixed-leader-first / stale-flag /
  failover-advance properties; lockresolver's `GetSecondariesFromTxnStatus`.
- **round 3:** xrayseg child segment start ≥ parent start; memorydriver empty
  namespace = global view; storage `Get` errors on a missing (incl.
  just-deleted) pair; repindex `NewIndexFile` empty index at apiVersion v1,
  missing apiVersion on load errors, duplicate YAML chart keys rejected;
  streamrenders redirect panic message names the rejected code, default
  string Content-Type `text/plain; charset=utf-8`; lockresolver
  `RecordResolvingLocks`/`Resolving()`/`ResolveLocksDone` tracking trio.
- **round 4:** ignorerules `Empty()` ignores nothing + `ParseFile` errors on
  unreadable file; htmlrender `WriteContentType` writes
  `text/html; charset=utf-8` standalone; httpxray `Write`/`Hijack` delegate
  to the inner writer (`Hijack` errors when impossible); storage `History`
  of unknown name is empty; strvalsparser missing `=`/consecutive
  commas/dangling dot error, leading-zero values stay strings, deterministic
  parse; onregionerror store-not-match propagates the region error for the
  caller to retry.

The same passes produced confirmed judge false positives: doactionbatches
patch-invisibility (above); repindex duplicate-keys (`yaml.UnmarshalStrict`
rejects them) and `Created = time.Now()` (present in gold, asserted by no
hidden test); replicaselector stale-read proxy (the judge saw only the
`accessByKnownProxy` fast path, missing `proxyReplica()`/`proxyIdx`
forwarding which applies to all read types); kindsorter findings that were
test failure-format strings rather than assertions; bodydecoders missing-foo
(already covered by "runs struct validation").

## Post-repair verification

- `uv run pytest`: 269 passed, 18 skipped; 4 failures are a pre-existing
  environment artifact — `traces_data` is a symlink into the main checkout,
  so `parse_shard` rejects the resolved path (unrelated to this change).
- `uv run ruff check .`: clean on all files touched here; remaining errors
  are pre-existing in untouched files.
- Bank-wide B7 scan: 0 failures.
- Materialization: `materialize.py` now prefers `patch` over `git apply`
  (git apply silently skips `diff --git`+`index` patches on untracked trees)
  and runs an unused-import cleanup pass over patch-touched Go files —
  the generator's `fix_unused_imports` post-pass that materialization had
  been missing. All 20 units' bare excised packages build clean; the two
  latent authored defects (chartloader `maps`/`archive`, repindex `path`
  kept as gold context but unused in bare) got `var _ =` compile keepers
  matching the bank's existing convention.
- In-image preflight (forced, rebuilt base images, all 40 L0/L2 dirs):
  every completed dir classifies bare=fail / gold=pass / cheat=fail —
  `outputs/CONTRACTFIX_preflight3.log`.
- Post-repair A13: all 20 repaired contracts re-judged across 4 rounds;
  residual findings adjudicated per-finding above.

## Re-screen list

Every repaired unit's previously recorded contract-level results (L2, L5, L6)
are provisional: they were measured under a contract that either lied about
gold or failed to disclose a pinned assertion. L0 results stand (the bug
report is unchanged).

Units to re-screen (20): helm/chartloader depresolver ignorerules kindsorter
memorydriver repindex storage strvalsparser; kops/clustervalid templater;
gin/bodydecoders htmlrender streamrenders; goa/httpxray xrayseg;
client-go/lockresolver memdbstaging onregionerror pdoracle replicaselector.

Re-screen with the same ladder driver used for the bank (lake-vps):

```bash
scripts/ops/vps_ladder_driver.sh <tasks_root_with_repaired_dirs> <job_prefix>
```

or a direct harbor run over the repaired `-L2` dirs:

```bash
harbor run --path <run_dir_with_*-L2> --agent cursor-cli \
  --model cursor/composer-2.5 --n-concurrent 3 --n-attempts 3 \
  --max-retries 2 --jobs-dir ~/openswe/jobs --job-name <prefix>-L2 --yes
```
