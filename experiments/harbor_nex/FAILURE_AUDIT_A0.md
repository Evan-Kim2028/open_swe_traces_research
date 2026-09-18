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
