# Verified hidden suites — authored_groknatsserver / nats-server

Verifier job `VFgroknatsservernatsserver`, worktree
`oswt-VFgroknatsservernatsserver`. All 10 units got a hidden suite
(`tests/hidden/server/<unit>_bb_test.go`, package `server`, in-package so
unexported functions are reachable), a generated `tests/test.sh`
(sha256-pinned install + `go test -run '^TestDetail' ./server/...`), and a
fresh `_author/contract.md` (none existed — the cohort was unstageable
without them).

Image: `ladder-base:nats-server` (go1.26.8). Each unit verified with
`verify_hidden.sh <unit>`: excision applied, hidden tests copied in, then
bare / gold / cheat variants run.

## Result matrix

| unit | tests | bare | gold | cheat | A12 (gold touches no test) |
|---|---|---|---|---|---|
| dns-alt-name-matches | 10 | FAIL | PASS | FAIL | OK |
| encode-consumer-state | 11 | FAIL | PASS | FAIL | OK |
| get-storage-size | 12 | FAIL | PASS | FAIL | OK |
| index-placeholders | 18 | FAIL | PASS | FAIL | OK |
| jwt-path-for-key | 7 | FAIL | PASS | FAIL | OK |
| jwt-time-windows | 14 | FAIL | PASS | FAIL | OK |
| parse-msg-schedule | 16 | FAIL | PASS | FAIL | OK |
| should-sample | 18 | FAIL | PASS | FAIL | OK |
| write-leaf-sub | 8 | FAIL | PASS | FAIL | OK |
| ws-pmc-extension | 11 | FAIL | PASS | FAIL | OK |

All 10 units satisfy all four gates. No DETAILS line was dropped: 125
`TestDetailNN` functions, numbered 1:1 with the detail rows.

## Inferable breakdown and what was asserted for each `no`

| unit | yes | doc | partially | no |
|---|---|---|---|---|
| dns-alt-name-matches | 5 | 4 | 1 | 0 |
| encode-consumer-state | 3 | 2 | 3 | 3 |
| get-storage-size | 6 | 0 | 4 | 2 |
| index-placeholders | 12 | 3 | 2 | 1 |
| jwt-path-for-key | 5 | 0 | 0 | 2 |
| jwt-time-windows | 7 | 4 | 1 | 2 |
| parse-msg-schedule | 3 | 6 | 0 | 7 |
| should-sample | 10 | 1 | 1 | 6 |
| write-leaf-sub | 6 | 2 | 0 | 0 |
| ws-pmc-extension | 9 | 0 | 1 | 1 |

What "shape, not literal" looked like per `Inferable: no` row:

- **encode-consumer-state** 1–2: byte 0/1 asserted against the package's own
  `magic` / `newVersion` constants, not literals 22/2. Line 6: the
  delta-encoded pending record is asserted through the surviving
  `decodeConsumerState` round-trip plus a "exactly three varints per record"
  structural parse — the delta formulas are never reimplemented.
- **get-storage-size** 3: the `(0, nil)` tuple is the commitment itself —
  asserted as-is. 11: lowercase suffixes asserted only to error.
- **index-placeholders** 13: bad delimiter asserted as `BadTransform` +
  invalid-arg mapping error naming the token; no message text.
- **jwt-path-for-key** 5–6: shard component asserted as a 2-char substring of
  the key equal to its suffix and different from its prefix — "suffix, not
  hash, not prefix" is the derivable shape.
- **jwt-time-windows** 5, 7: asserted through observable boundary behaviour
  (seconds honoured via a 5-second window; exclusive start/end via instants
  exactly on the endpoints), never through the committed internal formula's
  literal restatement of internals — though the behaviour itself is fully
  pinned.
- **parse-msg-schedule** 1: `(zero,false,true)` tuple asserted as committed.
  7: sub-second rejection asserted as "errors below 1s, ok at 1s".
  9–13: each alias asserted as its calendar shape (first second of
  Jan 1 / month / Sunday / day / hour, repeating, strictly after now, within
  one period) — never the expansion string.
- **should-sample** 4: the inclusive `<=` bound asserted statistically —
  `sampling=1` over 20k draws must hit a 250–800 window (~2%), which an
  exclusive `<` (~1%) cannot reach. 8, 13–16: field-shape and bit rules
  asserted (`a:b:c:1`/`ff`/`01` sample; wrong field counts, non-hex,
  3-char flags deny), no literal internals.
- **ws-pmc-extension** 6: asserted through which parameter positions count
  (after the matching token, same extension only), not iteration internals.

`partially` rows: asserted only the derivable consequence — e.g.
index-placeholders 14 pins `BadTransform` + a mapping-destination error
naming the token but deliberately not the inner arity sentinel (see below);
should-sample 5 uses an always-sampling header at `sampling=1` so any
random-miss path that skips headers fails within 200 draws.

## Two test bugs caught by gold runs (fixed, re-verified)

1. **index-placeholders T14** — I asserted the `ErrMappingDestinationNotEnoughArgs`
   inner sentinel for `{{random()}}`; gold returns the invalid-arg sentinel.
   The DETAILS line is `partially` and the sentinel choice is not derivable;
   softened to `BadTransform` + `*mappingDestinationErr` naming the token +
   `errors.Is(ErrInvalidMappingDestination)`. This is exactly the failure
   mode the `no`/`partially` grading exists to catch — the written commitment
   overstated what a solver could derive.
2. **jwt-path-for-key T02** — my "invalid key" fixture `AAAA…×56` is
   actually a *valid* nkey (premise-check assertion caught it, which is why
   the guard exists). Replaced with a real generated key whose last char is
   swapped — checksum-broken, still premise-checked with
   `nkeys.IsValidPublicKey` before asserting the empty path.

Also fixed pre-verify: encode-consumer-state round-trip tests use a fixed
second-aligned timestamp (`time.Unix(1700000000,0)`), removing a theoretical
flake if the encoder's `time.Now().Round(Second)` crossed a second boundary
between fixture construction and encode.

## Refusals

None — every DETAILS line maps to a test. The two `partially` softenings
above are documented in the contracts; no line was refused outright.

## Notes / hazards recorded

- The excision-patch body was glimpsed twice while checking headers
  (write-leaf-sub's removed lines, should-sample's stub-adjacent code). In
  both cases what was visible was already committed verbatim in DETAILS
  (the `l /= 10` digit loop is DETAIL 4; the header-check order is DETAIL
  18), so no assertion was informed by it. Recorded for audit.
- `should-sample` asserts the `sampling=1` ~2% rate with a ±3σ-ish window —
  P(flake) is negligible (2% mean 400, asserted 250–800).
- `jwt-time-windows` T03's negative half (local-zone mismatch) is guarded by
  a zero-offset check so it self-skips on non-UTC hosts.
- `write-leaf-sub` T08 captures traces via the in-package `DummyLogger`
  alias on a minimal `&Server{opts:&Options{}}` — no real listener.
- `should-sample` builds clients as `&client{}` with `c.parseState.header`
  set directly — the field the excised function reads.
