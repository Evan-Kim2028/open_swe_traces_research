# Verified hidden suites — authored_batch5 kops (2026-09-21)

Hidden test suites for the 20 kops units in
`experiments/pipeline/authored_batch5/kops/<unit>/`, written blind to
`gold.patch` (assertions derive only from `DETAILS.md` + `api.md` + public
signatures/constants). Each unit carries `tests/hidden/<pkg>_bb_test.go`
(`TestDetailNN` ↔ DETAILS row), a generated `tests/test.sh`, and an
`_author/contract.md` mapping every test to a commitment.

Verification (per unit, `verify_hidden.sh`, image `ladder-base:kops`):

| Check | Result |
|---|---|
| excised tree + hidden suite | FAIL (reward=0) — all 20 |
| excised + gold + hidden suite | PASS (reward=1) — all 20 |
| excised + cheat + hidden suite | FAIL (reward=0) — all 20 |
| A12: gold touches no test file | PASS — all 20 |

Full run log: `outputs/VFbatch5kops.log`.

## Per-unit notes

| Unit | Tests | Inferable (yes/doc · partially · no) | What the `no` rows asserted | Refusals / deviations |
|---|---|---|---|---|
| awstags | 7 | 3 · 3 · 1 | D4: bounded deterministic output, prefix carried — literals not pinned | — |
| channelver | 9 | 6 · 2 · 1 | D8: channel-independence + false-for-custom shape | D9 nil-version+ranged case: gold panics; DETAILS "nil skips range check" inaccurate — asserted only non-nil-version behavior, documented |
| clusterpreds | 13 | 3 · 7 · 2¹ | D8: version-gate shape (old→no, new→yes); D9: dotted name ending in cluster name | — |
| featflags | 7 | 4+1doc · 1 · 1 | D7: non-nil error naming the flag | — |
| fichanges | 7 | 4 · 3 · 0 | — | — |
| fideps | 7 | 5 · 2 · 0 | — | D6 "error" is a fatal — asserted as subprocess death, not error return |
| firesources | 7 | 4 · 2 · 1 | D3: valid JSON string shape | — |
| fiutils | 8 | 3 · 3 · 2 | D4: documented form parses, garbage errors; D6: charset/`_`/last-200 per api.md | — |
| gcenames | 8 | 2+1doc · 2 · 3 | D3: truncation bounds+determinism+death (constants unpinned); D4: package-constant key + safe-name value; D5: ClusterSuffixedName equivalence | — |
| gceurl | 7 | 2 · 2 · 2 | D3: quirk shape-only (no Region on under-fed input); D5: per api.md rule + documented example | — |
| hashparse | 7 | 3 · 4 · 0 | — | D4/D6 fatal exits asserted via subprocess |
| igroles | 8 | 3 · 3 · 1¹ | D8: api.md literal key + alloc/value shape | — |
| jsonxform | 8 | 3 · 3 · 2¹ | D2: dotted key chain (leading dot not pinned); D3: element path == slice path | — |
| kapiutil | 5 | 1 · 3 · 0 | — | — |
| manifest | 8 | 3 · 2 · 1 | D2: separator exists, empties dropped, round-trips | D8 root-replacement fatal unreachable through `accept` — asserted reachable shape only |
| nodelabels | 8 | 2 · 4 · 2 | D4: empty-string value (per api.md example); D8: `/`-qualified key shape | D1 multi-role clause inaccurate — comma-role errors like unknown role (asserted error, deviation documented); D2 api.md "(else Spec.Kubelet)" fallback does not exist — asserted CP never consults Spec.Kubelet |
| podmutate | 8 | 4+1doc · 2 · 1 | D8: gate asserted; spc_t/s0 pinned per api.md doc | D2 write-only-on-diff not externally observable — observable half asserted |
| strorset | 6 | 2 · 4 · 0 | — | — |
| subnetmath | 6 | 4 · 2 · 0 | — | — |
| vfsimpl | 9 | 4 · 3 · 2 | D7: brace shape + `%+v` entries; D8: `google_storage_bucket_object` + name shape | — |

¹ Tallies count `Inferable:` annotations in DETAILS.md; some rows carry a second
annotation (e.g. "partially … though the doc comments state it"), so counts can
differ from the row count. The authoritative per-row mapping is each unit's
`contract.md` coverage table.

## Cheat discriminators (what actually caught each cheat)

- awstags D2/D4 area; channelver range check; clusterpreds D13 warm-pool
  default; featflags unknown-flag `Get`; fichanges/fideps dependency
  reflection; firesources; fiutils; gcenames; gceurl; hashparse
  `MustFromString` exit; igroles; jsonxform sort ordering; manifest nested
  visitation through embedded base; nodelabels D7 mandatory-labels count;
  podmutate D8 SELinux gate (cheat applies unconditionally); strorset D5
  `String("a")` forced-array flag; subnetmath D5 allocate-marks-in-use;
  vfsimpl D7 `%+v` rendering.

## Method notes

- Tests live in-package so unexported helpers (`incrementIP`,
  `isGCSNotFound`, `newS3Path`, `o.data`) are reachable without changing
  the public surface.
- Fatal commitments (`klog.Fatalf`, `Exitf`) are asserted in child
  processes re-invoking the test binary behind an env guard.
- `vfsimpl` D4 needed no network: `getDetailsForBucket` short-circuits to
  `S3_REGION` when `S3_ENDPOINT` is set, so the regional/dualstack endpoint
  resolution is exercised offline.
- No `gold.patch` was opened; gold behavior was only observed by running
  the suite. Two DETAILS inaccuracies were found this way (channelver D9
  nil-version, nodelabels D1 multi-role + api.md D2 fallback) and are
  recorded as deviations in the contracts rather than silently bent tests.

No commits, no solver trials.
