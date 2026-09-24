# Verified — VFbatch5gogithub

Hidden test suites for the 20 go-github units in
`experiments/pipeline/authored_batch5/go-github`. Image
`ladder-base:go-github`, module `github.com/google/go-github/v92`,
go1.26.8. Every suite is `TestDetail01..NN` numbered 1:1 to the unit's
`DETAILS.md`; `Inferable: no` lines are asserted as committed shape
(error type, presence, non-leakage, route/field shape), never as
invented literals. Gold was used only to avoid asserting behavior gold
cannot satisfy — never to derive assertions.

All 20 units pass all four checks: excised+hidden → FAIL, gold → PASS
(reward=1), cheat → FAIL, A12 (gold touches no test file) → PASS.
Every `tests/test.sh` sha256 matches its hidden file; test counts equal
DETAILS line counts; every unit has a `_author/contract.md` with one
prose commitment per assertion and a coverage table pairing each hidden
test with its DETAILS row. Durable log: `outputs/VFbatch5gogithub.log`.

## Per-unit results

| unit | tests | Inferable mix | excised | gold | cheat | A12 |
|---|---|---|---|---|---|---|
| authtransports | 4 | doc/doc/partially/partially | FAIL | PASS | FAIL | PASS |
| baredo | 7 | doc/partially/partially/partially/no/doc/yes | FAIL | PASS | FAIL | PASS |
| clientclone | 6 | doc/yes/no/partially/partially/partially | FAIL | PASS | FAIL | PASS |
| createcommit | 4 | yes/partially/partially/yes | FAIL | PASS | FAIL | PASS |
| createfork | 3 | partially/partially/yes | FAIL | PASS | FAIL | PASS |
| dounit | 5 | partially/yes/yes/doc/doc | FAIL | PASS | FAIL | PASS |
| downloadasset | 5 | yes/doc/yes/partially/partially | FAIL | PASS | FAIL | PASS |
| downloadcontents | 5 | doc/partially/yes/partially/no | FAIL | PASS | FAIL | PASS |
| fetchsbom | 5 | doc/yes/partially/yes/partially | FAIL | PASS | FAIL | PASS |
| getcontents | 4 | partially/yes/no/yes | FAIL | PASS | FAIL | PASS |
| gitrefs | 5 | partially/partially/partially/partially/yes | FAIL | PASS | FAIL | PASS |
| markdown | 4 | doc/doc/partially/yes | FAIL | PASS | FAIL | PASS |
| newformreq | 5 | doc/yes/doc/partially/yes | FAIL | PASS | FAIL | PASS |
| newreq | 3 | doc/partially/yes | FAIL | PASS | FAIL | PASS |
| pullmerge | 3 | partially/no/yes | FAIL | PASS | FAIL | PASS |
| ratelimitget | 4 | doc/partially/partially/yes | FAIL | PASS | FAIL | PASS |
| searchq | 4 | doc/no/partially/partially | FAIL | PASS | FAIL | PASS |
| statsreshape | 4 | partially/partially/partially/yes | FAIL | PASS | FAIL | PASS |
| uploadasset | 5 | partially/partially/doc/partially/no | FAIL | PASS | FAIL | PASS |
| uploadreq | 7 | partially/doc/no/partially/partially/yes/doc | FAIL | PASS | FAIL | PASS |

## What was asserted for `Inferable: no` (and notable weakenings)

- **baredo D3** — DETAILS says the secondary limit "sleeps then retries
  once"; gold does not sleep or retry — it records
  `now + min(Retry-After, maxSecondaryRateLimitRetryAfterDuration)` and
  returns the `*AbuseRateLimitError`, short-circuiting subsequent calls
  inside the window. Asserted the derivable commitment: error type +
  recorded-window short-circuit (zero additional network hits), not the
  retry mechanics.
- **baredo D5** — asserted `*url.Error` type and that the error's `URL`
  field carries no sensitive query material. The whole `Error()` chain is
  not asserted: real transports produce a single `*url.Error` whose
  message derives from the sanitized field; a synthetic nested
  `*url.Error` would leak via its inner message without contradicting
  the commitment.
- **clientclone D3** — asserted as wire behavior: a token-bearing clone
  sends `Bearer` credentials to the clone's own configured origins and
  leaks nothing to a foreign origin. The base-vs-wrapped transport
  mechanism is not pinned.
- **downloadcontents D5** — asserted a `*Response` is returned, no body
  is returned, and the error is the transport failure (not
  `ErrContentsNoDownloadURL`); which response object propagates is not
  pinned.
- **getcontents D3** — asserted an error surfaces when neither response
  shape decodes; the combined-error wording is not pinned.
- **pullmerge D2** — asserted `commit_message` is present-and-empty in
  the sent JSON body; the flag-times-empty pairing is the commitment.
- **searchq D2** — asserted each previewed search type's `Accept` carries
  a `vnd.github` media type beyond the bare default, with every
  comma-joined token non-empty; the preview-name table is not pinned.
  (Gold sends only the per-type preview — not base+preview — so the
  suite does not assert a multi-token join on the no-TextMatch path.)
- **uploadasset D5** — asserted a non-empty `Content-Type` is produced;
  the media-type precedence chain is not pinned.
- **uploadreq D3** — asserted the request body does not expose the
  caller's concrete reader type (a caller-supplied seekable reader
  arrives wrapped in a non-`io.Seeker` facade); the wrapper type is not
  pinned.

## DETAILS lines adjusted or refused

- **newformreq D5** — DETAILS says "`Content-Type` is the multipart
  boundary produced by the form buffer"; the method's documented
  contract (and upstream) is `application/x-www-form-urlencoded`. The
  multipart wording is an authoring slip — asserted the documented
  urlencoded literal.
- **authtransports D4** — DETAILS says `client_id`/`client_secret` are
  added "as URL query parameters"; the solver-visible surviving test
  (`TestUnauthenticatedRateLimitedTransport`) and gold both use basic
  auth headers. Asserted the derivable core: credentials are conveyed
  while existing query values are preserved; the query-param mechanism
  claim was not asserted.
- **baredo D3** — the "sleeps the indicated retry-after … retries the
  request once" sub-claim is contradicted by gold (no sleep/retry
  observed); asserted the recorded-window short-circuit half only.

## Notes

- `markdown` is the only unit whose excision deletes a test file; all
  shared helpers (`setup`, `testMethod`, `mustNewClient`,
  `mustParseURL`, `openTestFile`) survive everywhere and are reused.
- `uploadreq` D1/D3/D4 and `getcontents` D1 discrimination relies on
  caller-supplied reader types and raw `?`/`..` path segments rather
  than literals.
- `verify_hidden.sh` runs excised → gold → cheat in one container with
  `GOPROXY=off`; the cheat suites in this batch genuinely exercise the
  bug (e.g. clientclone's cheat fails D3's origin-scoping assertion,
  baredo's cheat fails D3's recorded-window assertion).
