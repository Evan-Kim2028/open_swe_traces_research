# authored_batch2 — nats-server (14 units)

Authoring run `AUnatsserver` (worktree `oswt-AUnatsserver`, branch `au-natsserver`).
Units live at `experiments/pipeline/authored_batch2/nats-server/<unit>/_author/`.
Every unit verified locally with `_tools/verifyunit.sh`: excision applies +
compiles, gold applies + compiles + passes the original in-tree tests, cheat
applies + compiles + fails them. Overlap check
`uv run python scripts/check_unit_overlap.py --extra <worktree>` → **CLEAN**
(14 new vs 0 existing nats-server units, 0 overlaps, also disjoint among
themselves — verified by the checker's file-set diff).

| unit | closure | files | lines | details | predicted_flip | commitments excised | compiles |
|---|---|---|---|---|---|---|---|
| ldapdn | LDAP distinguished-name parse/serialize | internal/ldap/dn.go | 305 | 10 | L2 | DN grammar, escapes, case, RDN ordering, malformed inputs | yes |
| archiveio | archive read/write + index records | server/archive/archive.go | 356 | 11 | L2 | stream framing, index records, short/corrupt reads, checksums | yes |
| hashwheel | time-hash wheel (encode/decode, schedule) | server/thw/thw.go | 244 | 8 | L2 | wire version errors, empty hash, ordering, duplicate adds | yes |
| seqset | AVL sequence set + v1/v2 codecs | server/avl/seqset.go | 715 | 17 | L2 | dup inserts, min/max, delete-vs-empty, set ops, codec versions, truncated decode | yes |
| gslsublist | generic subject sublist | server/gsl/gsl.go | 566 | 15 | L2 | `*`/`>` token rules, invalid subjects, literal wildcards, dup values, removal, interest queries | yes |
| stree | subject tree (radix trie) + wildcard parts | server/stree/stree.go, parts.go | 700 | 15 | L2 | byte-127 refusal, token-aligned wildcards only, `>` terminal+≥1, `foo.bar.>` excludes `foo.bar`, nil receivers, dedup | yes |
| confparse | config file parser (lexer stays) | conf/parse.go | 514 | 16 | L2 | 1000-vs-1024 suffixes, 6 bool spellings, Zulu datetime, scoped vars+env re-parse, bcrypt `2a$`, cycles, dangling `}`, pedantic tokens, digest | yes |
| subjecttransform | subject mapping/transform engine | server/subject_transform.go | 688 | 17 | L2 | `$N`/mustache placeholders, arg arity/range per fn, int32 bound, strict-mode rules, `>`-tail emit, missing-token slot, FNV partition, n=0→"0" | yes |
| ipqueue | bounded generic intra-process queue | server/ipqueue.go | 334 | 12 | L2 | signal only empty→non-empty, pushMany all-or-nothing revert, unchanged len on limits, at-limit admitted, in-progress accounting, recycle caps | yes |
| utilparse | shared parse/format helpers | server/util.go | 408 | 18 | L2 | 9-digit vs silent-wrap parsers, -1 vs (0,false) conventions, port 0/-1→default, userinfo URL equality, original-slice redact, MinInt64 comma, GOMAXPROCS floor, saturating math | yes |
| proxyproto | PROXY protocol v1+v2 reader | server/client_proxyproto.go | 415 | 16 | L2 | 6-byte detect + byte replay, 107B line bound, CRLF over-read, UNKNOWN/LOCAL no-ops, colon-based family check, UNSPEC discard, TLV extras | yes |
| jwtvalidate | operator JWT load + claim validators | server/jwt.go (+2 tests snipped in jwt_test.go) | 275 | 14 | L2 | `eyJ` inline fallback, buffer wipe, operator option matrix, strict-signing expansion, strict time boundaries, overnight windows, max-of-windows | yes |
| jsversioning | JetStream metadata API-level bookkeeping | server/jetstream_versioning.go | 246 | 12 | L2 | feature→level table (1/2/4), max-of-features, always-stored "0", dynamic-key stripping, nil-collapse on copy, bad-int header → reject | yes |
| cronparse | six-field cron evaluator | server/cron.go (+cron/cron_tz subtests snipped in jetstream_test.go) | 327 | 14 | L2 | 6 fields (not 5), `?` alias, `n/k`→`n-max/k`, `*/k` loses star mark → dom/dow AND-vs-OR flip, strict-after second alignment, 5-yr cap | yes |

## Existing closures avoided

- The authored dataset (`experiments/pipeline/authored/`) contains **zero**
  nats-server units, so no prior closures constrained selection.
- `ldapdn`, `archiveio`, `hashwheel` were authored earlier in this batch
  (pre-existing when work resumed); all 11 newly authored units excise
  disjoint file sets and the overlap checker reports CLEAN.
- Two units (`jwtvalidate`, `cronparse`) live in files whose test coverage
  is embedded in large shared test files (`jwt_test.go`,
  `jetstream_test.go`); only the test functions/subtests exercising each
  closure were excised — the surrounding tests remain, keeping the
  excision minimal and the trees compiling.
- `confparse` deliberately excises only `parse.go` and leaves `lex.go`
  (and its tests) in place: the item stream is visible scaffolding, which
  keeps the unit information-bound (semantics hidden) rather than
  capability-bound (rewrite-a-lexer).
- `cronparse` similarly leaves `scheduler.go`'s dispatcher visible; only
  the pattern-language evaluator is excised.

## Contract-vs-gold audit (2026-09-20, second pass)

Every contract claim was re-checked against the pinned gold implementation;
the following false claims were corrected (all were wording defects in
`contract.md`/`DETAILS.md`, never patch defects):

- `archiveio`: magic is written lazily on first entry — an empty archive is
  0 bytes, not 8; `Write(nil)` on a closed writer errors (closed checked
  first); incomplete-entry check precedes the sum-overflow check.
- `confparse`: unknown integer suffixes are unreachable (lexer emits them as
  strings) — hedge resolved.
- `cronparse`: empty comma terms (trailing/leading/doubled commas, bare `,`)
  are silently skipped, not errors.
- `gslsublist`: `NumInterest` returns at the first confirmed match —
  overlapping wildcard+literal branches may go uncounted.
- `hashwheel`: decode resets slots and lowest but NOT the member count —
  decoding into a non-empty wheel accumulates it.
- `jwtvalidate`: the pinned-account block (validation + system-account
  auto-pin) only runs when the pinned set is already non-empty.
- `ldapdn`: a trailing backslash is silently dropped; a pending type with
  empty value at EOF is silently dropped; buffered text with no `=` still
  errors. Details count 9 → 10.
- `proxyproto`: bad ports produce plain port errors, not the invalid-header
  sentinel.
- `seqset`: worked example corrected — `0..8191` spans 4 windows (was
  `0..4095`, which spans 2).
- `stree`: `'*.*` matches `'*.123` (first `*` not token-aligned → literal
  `'*.` part), not `bar`.
- `subjecttransform`: only `{{…}}`-shaped tokens with unknown names are
  errors; tokens not ending `}}` are literals.
- `utilparse`: only valid semver `-prerelease`/`+build` suffixes parse —
  not "arbitrary" suffixes.

Re-verified after edits: overlap CLEAN; all gold patches touch zero
`*_test.go` files (A12); all 14 excised trees compile (package + remaining
tests); details counts match `difficulty.md` on all 14 units.

## Independent re-verification (2026-09-20, third pass — `reverify.sh`)

All 14 units re-checked end-to-end in fresh scratch copies: excision patch
applies, excised package builds, **remaining in-tree tests still compile**
(`go test -c`), gold applies + builds + passes every closure test, cheat
applies + builds + fails them. 14/14 `RESULT rc=0`.

Two findings folded back into the artifacts:

- `jsversioning`: the coverage table was missing rows for
  `TestGetAndSupportsRequiredApiLevel` and
  `TestJetStreamApiErrorOnRequiredApiLevel` — added (both true of gold).
- `jsversioning`: `TestJetStreamApiErrorOnRequiredApiLevelDirectGet` and
  `TestJetStreamApiErrorOnRequiredApiLevelPullConsumerNextMsg` **fail on the
  pristine pinned tree** (verified at `repos/nats-server/src` HEAD): those
  two paths do not invoke `errorOnRequiredApiLevel` at this revision, so the
  asserted 412 is never emitted. Not a unit defect — upstream-mid-feature
  tests. Documented in `contract.md` (listed but flagged out-of-contract)
  and `DETAILS.md` line 11 caveat: **the verifier must not write hidden
  assertions over the direct-get or pull-next-message paths** — a hidden
  test mirroring them would fail gold and violate A1.
- `seqset`/`stree`: `TestNoRace*RelativeSpeed` and `*Perf` tests excluded
  from the re-run oracle (timing gates, not contract assertions); they still
  compile in the excised tree.

## Post-review fixes (2026-09-20, independent subagent review)

- **Reproduction commands**: all 14 bugreports now use `tests/test.sh`
  (spec §1.3 — the repro runs the hidden suite). The previous
  `go test ./<pkg>/` forms ran zero tests on the excised tree for 12 units
  (all closure tests are deleted) — vacuous repros. `tests/test.sh` both
  satisfies B6 and removes the package-locality hint.
- `stree`: removed a phantom `Benchmark*` coverage row (no benchmarks exist
  in the package); added real benchmark rows for the 5 `BenchmarkSeqSet*`
  and 6 `BenchmarkHashWheel_*` functions the excisions delete.
- Contract-prose symbol leaks scrubbed (B7): `MatchUntil`/`Match` in stree
  prose → "the match walk"/"the completion-reporting variant"; `MinInt64` in
  utilparse → "the minimum 64-bit integer"; test names in the jsversioning
  note → de-named. Kept as caller-facing vocabulary (not leaks): `eyJ`
  prefix, `Options` field names (`TrustedKeys`/`Nkeys`/…), transform DSL
  function names (`splitfromleft`, `slicefromright`, …).

## Notes

- Tooling used: `_tools/mkunit.sh` (extended for `file::Spec` scoping),
  `genspecs.sh`, `applycheat.py`, `verifyunit.sh`.
- `verifyunit.sh` was extended to take a test-name regex so server-package
  units can run only their closure's tests.
- All predicted_flip values are L2: every unit hides ≥8 independent
  behavioural commitments not inferable from the remaining tree.
- No hidden tests written (author role only), no solver trials, no Harbor,
  no commit.
- Log: `outputs/AUnatsserver.log`.
