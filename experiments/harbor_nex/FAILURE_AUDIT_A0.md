# Failure audit — first Composer 2.5 failures (2026-09-18)

Classes: (a) legitimate miss, (b) verifier too narrow, (c) instruction insufficient, (d) infra.

| trial | job | outcome | class | evidence |
|---|---|---|---|---|
| client-go-onepc-scope-obf | composer25-obf | FAIL 4.1 min | (a) rule violation | Agent identified the inverted scope check in the library (its own summary) but ALSO edited consumer `*_test.go`; checksum guard tripped: "consumer test files were modified; the library must be fixed". Instruction states tests must not be edited. Counts as a real miss: the model took the disallowed shortcut. |
| spec-reimpl-A0 | composer25-unsolv-A0 | FAIL 10.1 min | (a) legitimate | 63 edits, full codec reimplemented from the prose contract; hidden `TestCodecV2/TestDecodeEpochNotMatch` and `TestDecodeBucketKeys` fail. The instruction (lines 36–39) describes all three epoch-clip cases (whole-keyspace → empty/empty, outside → dropped, overlap → truncated). Behavior was specified; implementation wrong. |
| dynamic-pipeline-A0 | composer25-unsolv-A0 | FAIL 4.5 min | (a) legitimate, boundary ambiguity noted | `Get` implemented; hidden `TestPipelinedFlushTrigger` / `TestPipelinedFlushGet` fail on `Flush(false)` returning false at the trigger point and on `OnFlushing()` state. Instruction gives thresholds (10000 keys and 16 MiB min; 128 MiB force) and the swap-to-in-flight semantics, but not the exact boundary (`>=` vs `>`) nor that `Flush(false)` must report `true` and enter the flushing state immediately. Partial (c): A1/A2 should resolve this; if A1 (test names + one-line descriptions) flips it, the missing information was the state-observation contract, not the algorithm. |

Obfuscation control (same six tasks, renamed): Composer 5/6 pass, solve times equal or faster than the public tree
(batchcmds 12.9 vs 23.4 min; keyspacecodec 8.8 vs 8.0). Recall of upstream code is NOT what made the earlier tasks easy;
test-suite completeness was. This is why removing the tests (spec-only) and replacing them with properties/dynamic gates is
where the first real failures appear.

Ladder status: A0 fails for spec-reimpl and dynamic-pipeline; property-backoff passes at A0 (properties in words + 3
examples were enough). A1 launched for the two failing families (`composer25-unsolv-A1`). Continue lazily: build/run A(k+1)
only where A(k) fails.

## A1 update (2026-09-18 19:55Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| dynamic-pipeline-A1 | FAIL | (a) legitimate | same hidden flush tests fail (`TestPipelinedFlushSize/Skip/Trigger`); names+descriptions did not help |
| spec-reimpl-A1 | FAIL | **(b) verifier too narrow** | hidden `codec_v2_test.go:105` calls `suite.codec.ThornSlot(...)`; solver implemented `thornSlot` (unexported). Build failed. The hidden tests are white-box: they call internal methods by name that no prose contract can convey. For spec-only tasks the verifier must be black-box (exported API / caller-facing behavior only), or A2 (signatures) is the minimum fair level. Action: rewrite the hidden suite for spec-reimpl to black-box before counting A0/A1 as legitimate. |

## A2 / Devin / black-box update (2026-09-18 20:25Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| spec-reimpl-A2 (Composer) | FAIL | (a) legitimate | signatures given, compiles, hidden `TestCodecV2` still fails on behavior; 84 edits, no web |
| dynamic-pipeline-A2 (Composer) | FAIL | (a) legitimate | implementation deadlocks in `asyncFlush`; verifier hit the 900 s timeout; 27 edits, no web |
| dynamic-pipeline-A0 (Devin swe-2-high) | FAIL | (a) legitimate, clean | only `TestPipelinedFlushGet` fails (Composer failed 3 tests at A0); tools exec/read/grep/edit/write only; no web tools; one harmless `git log` on a history-less tree |
| spec-reimpl-A0 (Composer), re-scored on the BLACK-BOX property suite | FAIL | (a) legitimate (reinstated) | compiles under the black-box suite; passes contract examples and unseen-random symmetry; fails key-range round-trip on empty ranges and epoch clip out-of-keyspace. The original A0 fail was behavioral, not naming. |

Ladder state: dynamic-pipeline fails A0, A1, A2 for Composer; A3 (one hidden test file restored) launched.
spec-reimpl: white-box ladder stopped at A2 (fair from A2 up, fails there); black-box family `spec-reimpl-bb` A0 running.
Devin: 1 clean legitimate fail so far, marginally closer than Composer on the same task.

## A3 / black-box A0 update (2026-09-18 20:50Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| dynamic-pipeline-A3 (Composer) | **PASS** 5.4 min | clean | one hidden test file restored; race clean, perf gate 37.65 ≤ 113 ns/op; no web, no network commands, 18 edits. **Flip point for Composer on this unit = A3.** |
| spec-reimpl-bb-A0 (Composer) | FAIL | (a) provisional | black-box property suite; 93 edits, no web; fails `TestCodecContractExamples` (the instruction's own worked examples). Provisional pending a check that the instruction examples and the test's expectations agree (if they disagree it is class (c)). bb-A1 launched. |

Addendum 20:58Z: `client-go-onepc-scope-obf` is INVALID as built (consumer module does not compile offline
after obfuscation); its Composer fail is excluded. `spec-reimpl-bb-A0` examples in the instruction match
`TestCodecContractExamples` byte-for-byte (0x72 00 10 92 + "key"), so that fail is class (a), confirmed.

## Devin final (2026-09-18 21:05Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| spec-reimpl-A0 (Devin swe-2-high, white-box hidden tests) | FAIL | (a) legitimate | compiled against the white-box suite (it inferred the internal names from callers), then failed `TestCodecV2` on behavior; tools exec/read/edit only; no web |
| dynamic-pipeline-A0 (Devin) | FAIL | (a) legitimate | see above; one test short |

Devin swe-2-high vs Composer 2.5 at A0: both fail both units. Devin is closer on the dynamic unit (1 failing
test vs 3) and cleared the naming barrier on the codec unit. No evidence of web use in any Devin trial.

## Black-box A1 (2026-09-18 21:20Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| spec-reimpl-bb-A1 (Composer) | **PASS** 7.0 min | clean | black-box property suite; hidden test names + one-line descriptions added; no web, no network commands, 64 edits. **Flip point for Composer on the codec unit (black-box) = A1.** |

## Second batch on lake-vps (2026-09-18 23:10Z) — Composer 2.5, egress sidecar, all trials web=0

| unit (client-go, obfuscated) | family | A0 | A1 | A2 | A3 |
|---|---|---|---|---|---|
| spec-bb-chain (RPC interceptor chain) | spec-only, black-box props | PASS | | | |
| spec-bb-bucket (bucket lookup) | spec-only, black-box props | PASS | | | |
| property-policy (backoff policy table) | property | PASS | | | |
| property-1pc (1PC/async-commit decision) | property | FAIL | FAIL | FAIL | running |
| dynamic-snapshot (snapshot getter/iter; perf gate re-measured on ARM: gold 137 ns/op, limit 411) | dynamic | PASS | | | |
| dynamic-latch (latch scheduling; race gate) | dynamic | PASS | | | |

Ablation round-2 units (mgechev/revive, the "five hardest" from each condition, 3 judged valid): Composer passed all
three at A0 (file-exclude-filter, file-filter, revivelib-runner) in ~5 min each, web=0. Both builders' "hardest"
picks on revive sit at A0 for a mid-tier model.

Reading so far: 5 of 6 new client-go units and 3 of 3 revive units are A0 for Composer. Flip points above A0 remain
the codec (A1) and the two state-machine units (dynamic pipeline A3; 1PC decision > A2). Hardness for this model is
concentrated in units whose behavior is a sequence/state contract rather than a pure function of inputs.

## 1PC unit voided; Devin codec pass (2026-09-18 23:40Z)

| trial | outcome | class | evidence |
|---|---|---|---|
| property-1pc A0..A4 (Composer, lake-vps) | FAIL x5 | **(b) verifier too narrow** | validation.json B4 = FAIL: the property test calls unexported `checkAsyncCommit` / `checkOnePC` by name. At A3/A4 the solver edited the in-tree property test (checksum guard: "test file modified"), consistent with renaming to its own identifiers. All five results voided; unit needs a black-box property verifier (exported commit path only). Builder queued. |
| spec-reimpl-bb-A0 (Devin swe-2-high) | **PASS** 37.1 min | clean | black-box codec at A0; tools exec/read/edit only, no web. Composer failed this level. **First measured capability gap: Devin flips at A0 on the codec, Composer at A1.** |

Rule reinforcement: B4 must be checked mechanically at build time for EVERY hidden/property test (the rules
registry already flags it; the builder shipped the unit anyway). Make B4=FAIL block packaging.

## Replication (2026-09-19 00:05Z) — single-attempt flip points are NOT stable

lake-vps, Composer 2.5, 3 attempts per level around each flip (partial, 7/12 done): codec black-box L2 (old A0)
PASS (earlier FAIL); codec L3 (old A1) FAIL (earlier PASS); pipeline L5 (old A3) FAIL (earlier PASS); pipeline L4 FAIL
(consistent). Three trials errored with `CancelledError` at 23:58Z, coinciding with the first Docker cleanup run
on the host — the image-prune step can race a trial between build and start. Cleanup now skips images/containers
while any Harbor job is active; the errored trials will be re-run.

**Rule C6 (new):** a flip point is defined on pass RATE over >= 3 attempts per level (flip = lowest level with
>= 2/3 passes). All earlier single-attempt flip points are provisional until replicated. Every ladder run from
now on uses `--n-attempts 3`.

## Score test and 1PC rebuild (2026-09-19 01:15Z) — Composer 2.5, laptop, 3 attempts each, egress sidecar

| unit | statefulness score (mean_entries / mean_calls) | predicted | L2 result |
|---|---|---|---|
| delete-range | 9.0 / 22.0 (top) | fail | **3/3 pass** |
| batch-delete | 9.0 / 19.0 (top) | fail | **3/3 pass** |
| decode | 2.0 / 2.0 (bottom) | pass | 3/3 pass |
| next | 2.0 / 2.0 (bottom) | pass | 3/3 pass |

**Negative result:** the statefulness score did not predict the L2 outcome; both high-score units were as easy
as the low-score ones. The n=11 correlation (ρ≈0.5–0.6) was fit to single-attempt labels and does not transfer.
Selection rule status: "stateful contract" remains a description of the two units that were hard, not a
manufacturing lever. Dropped as a lever.

| unit | verifier | L2 result |
|---|---|---|
| property-1pc (rebuilt black-box) | seeded properties through exported commit API | **3/3 pass** (VPS) |

The earlier five white-box failures were entirely the verifier. Both findings sharpen the same point: with a
fair verifier and a full contract, nearly every unit in client-go is L2 for a mid-tier model; the two exceptions
(pipelined buffer at L5, codec at L3, both pending replication) are the only difficulty we have found.

## Replication final (2026-09-19 01:45Z) — Composer 2.5, 3 scored attempts per level (pass 1 on lake-vps + pass 2 on laptop)

| unit | level | passes / attempts | single-attempt reading was |
|---|---|---|---|
| codec (black-box) | L2 | 1 / 3 | fail |
| codec (black-box) | L3 | **0 / 3** | pass |
| pipelined buffer | L4 | 0 / 3 | fail |
| pipelined buffer | L5 | **2 / 3** | pass |

Under C6: pipeline flip = **L5** (confirmed). Codec flip is **> L3** (the earlier "L3" was a one-off; L4/L5 not yet
replicated). Single attempts mislabeled one of two flip points. Devin's readings (codec L2 pass, pipeline L5
pass) remain single-attempt and provisional.

## Big-unit test (2026-09-19 02:50Z) — scale × withholding, verifier written BLIND by a separate agent

| unit | lines | L0 | L2 | audit |
|---|---|---|---|---|
| batchcmds (interface removed) | 221 | 3/3 pass | 2/3 pass | clean |
| keyspacecodec (906 lines / 5 files) | 906 | **0/3** | **0/3** | see per-trial line above; gold passed the blind suite, cheat failed |

First replicated unit above L2 for Composer 2.5 with a fair, independently authored verifier. Lever confirmed:
large closure + withheld tests. The separated author/verifier roles produced a valid task on the first try.
