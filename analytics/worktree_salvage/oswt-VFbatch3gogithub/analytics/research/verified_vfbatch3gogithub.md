# Verified hidden suites — authored_batch3/go-github (2 of 21 units authored)

Date: 2026-09-20. Branch: `vfbatch3gogithub`. Worktree: `oswt-VFbatch3gogithub`.
Log: `outputs/VFbatch3gogithub.log`.

Method: each authored unit got a black-box suite at
`tests/hidden/github/<unit>_bb_test.go` (external `github_test` package,
exported API only — `client.Do` against httptest for pagination, `json.Unmarshal`
for Timestamp) plus `tests/test.sh` (hidden-file checksum guard + `go test`).
Suites were written against `DETAILS.md` + `api.md` + the excised (solver-visible)
tree only. `gold.patch` was never opened; `excision.patch` minus-lines were never
read (they are gold-equivalent) — solver-visible state was taken from
`_tools/dump_excised.sh` output. `bugreport.md` was read once to confirm what the
Inferable annotations reference; no assertion traces to it alone.

## Scope — 19 of 21 units are authoring stubs, reported blocked

`authored_batch3/go-github` holds 21 unit dirs. Only **pagevalues** and
**timestamp** carry the full author pass (`DETAILS.md`, `api.md`, `cheat.patch`,
`closure.md`, `bugreport.md`). The other 19 hold only `gold.patch` +
`excised/excision.patch`. A gold-blind verifier has nothing to grade: no
DETAILS lines to number `TestDetailNN` against, no documented API surface, no
cheat patch for the A3 leg. Deriving commitments from gold would break the
information wall that makes a unit fair. Verified identical state in the
matching author worktree `oswt-AU4gogithub2` (excisions byte-identical) and the
main worktree; batch commit `456d45e` ("74 authored units, and the rate limit
that stopped them") records the cause. Blocked, not dropped:

`apiversion, auditstream, boolresp, brnotprot, comfortfade, credhdrs, enturls,
errfmt, erris, gethdr, hostrunvalid, patopts, propvalues, ratecat, ratehdrs,
respclassify, runidre, signmsg, urlpolicy`

When their `_author/` material lands, the same `_tools/verify_hidden.sh`
(one `docker run`, four checks) verifies each in minutes.

## Per-unit verification (four checks via `_tools/verify_hidden.sh`)

| unit | tests | excised | gold | cheat | A12 |
|---|---|---|---|---|---|
| pagevalues | 6 | FAIL (D1, D2) | PASS | FAIL (D1, D2) | PASS |
| timestamp | 7 | FAIL (D4, D5) | PASS | FAIL (D4, D5) | PASS |

Both units satisfy all four checks; `docker_rc=0` on every run.

## Per-unit detail

Convention: `yes/doc` = asserted exactly; `partially` = derivable part
asserted; `no` = shape only. "Refused" = a clause not asserted, with reason.

### pagevalues (`Response.populatePageValues`, github/github.go) — 6 tests

Inferable: no, no, partially, partially, partially, partially.
Excised stub parses `page`/`before`/`after` and silently drops `cursor=` and
`since=` links — the discrimination surface.

- D1 no: `cursor=abc123; rel="next"` asserted `Response.Cursor == "abc123"`
  (the cursor link is honored — the bug report's own claim) plus
  `NextPage == 0` for a cursor-only link (cursor is not a page field —
  derivable). **Refused:** the "only rel=next picks up cursor" exclusion —
  `Cursor` is left unasserted for a `rel="prev"` cursor link, since the
  rel-scoping rule is arbitrary and a cursor-from-any-rel implementation is a
  legitimate fix for the reported symptom.
- D2 no: `since=2; rel="prev"` asserted `PrevPage == 2`; `since=<ts>&page=4;
  rel="next"` asserted `NextPage == 4`. The since→page alias is arbitrary per
  the gate, but the bug report states since-style links "report no next page",
  so a numeric since reaching the rel's page field is the derivable symptom
  fix. `Response` has no `Since` field; the integer page fields are the only
  target.
- D3 partially: a param-free link and an `unrelated=1` link asserted to leave
  all page fields zero — skipping empty links is derivable; *which* params
  qualify is not pinned.
- D4 partially: `page=abc; rel="next"` asserted `NextPage == 0 &&
  NextPageToken == "abc"` — the field is named in `api.md`, so the token
  landing place is derivable.
- D5 partially: a Link header mixing `garbage-segment`, `<missing-close`, and
  two well-formed links asserted `NextPage == 2`, `Before == "B1"` — malformed
  segments must not disturb valid links (robustness shape); exact validity
  rules not pinned.
- D6 partially: `before=` on `prev` asserted `Before == "B1"`; `after=` on
  `next` asserted `After == "A2"`. The named pairings are asserted; the
  exclusivity of the pairing is not (a superset implementation reading the
  params from additional rels still satisfies the assertion).

Docker: excised fails D1 (`Cursor = ""`) and D2 (`PrevPage = 0`); gold passes
all six; cheat fails D1 and D2; gold patch touches no test file.

### timestamp (`Timestamp.UnmarshalJSON`, github/timestamp.go) — 7 tests

Inferable: doc, doc, no, no, no, partially, partially.
Excised stub is a magnitude-threshold decoder (`i > 9999999999` → ms). The only
region where stub and gold disagree is 11-digit values whose seconds decode
lands in years 2286–3000, plus the boundary strictness.

- D1 doc: quoted RFC3339 asserted exactly — `"2006-01-02T15:04:05Z"`,
  fractional `.000Z` and `.999Z` forms, and a `+01:00` offset form all decode
  to the correct instant.
- D2 doc: `1136214245` asserted `Unix() == 1136214245` (2006-01-02T15:04:05Z).
- D3 no: `1136214245000`, `1615077308538`, `1136214245001` asserted to decode
  without error to the millisecond instant (`Unix()==1136214245`,
  `Unix()==1615077308`+538ms, `Unix()==1136214245`+1ms). Shape only: every
  reasonable implementation (gold, stub, naive magnitude) agrees that a
  13-digit value is ms — asserting the ms instant does not pin the arbitrary
  *trigger* (decoded-year rule), only the agreed granularity.
- D4 no: `32503680000` asserted to decode without error to a far-future
  instant (`Year() > 2286`). **Refused:** the exact boundary and its
  strictness — gold's `year > 3000` cutoff is arbitrary; an implementation
  flipping anywhere past the window passes my assertion but would fail a
  pinned `Unix()==32503680000` or `Year()==3000`.
- D5 no: `30000000000`, `20000000000`, `10000000000` asserted `Unix() == n` —
  "decodes to the instant it encodes" for interior points, which is the
  derivable claim of the bug report. The window's edges are not pinned.
- D6 partially: `0` asserted `Equal(time.Unix(0,0))` — epoch, not zero-time,
  not error.
- D7 partially: `"asdf"`, `"1234"` (quoted digits), `null` asserted to return
  a decode error; message content not pinned.

Docker: excised fails D4 (boundary → 1971) and D5 (interior → 1970s); gold
passes all seven; cheat fails D4 and D5 (it keeps a magnitude-style decode);
gold patch touches no test file.

## Notes for the reconciler

- pagevalues D1: the suite deliberately leaves `Cursor` unasserted for
  non-next rels. If the contract row for D1 claims next-only scoping, that row
  is DERIVABLE only at gold — it should be reconciled as gold-truth, not as a
  hidden assertion.
- timestamp D4/D5 are the load-bearing discriminators and are asserted as
  era/identity shapes, not the committed literals — consistent with the `no`
  grading. If a future contract row needs the exact year-3000 strictness, it
  cannot come from this suite.
