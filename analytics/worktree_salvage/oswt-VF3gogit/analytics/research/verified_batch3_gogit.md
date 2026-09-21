# verified_batch3_gogit — hidden suites for the 20 batch-3 go-git units

Date: 2026-09-21. Branch `vf3gogit`. Docker image `ladder-base:go-git`
(digest `sha256:8e91c0a4…b515`, go1.26.8, module `example.internal/gitkit/v6`).
Raw gate output: `outputs/VF3gogit.log`.

## Method

For each unit under `experiments/pipeline/authored_batch3/go-git/<unit>/`:

- Hidden suite at `tests/hidden/<pkg>/*_bb_test.go`, in-package (`package <pkg>`,
  not `*_test`), one `TestDetailNN` per DETAILS.md numbered line — the numbering is
  the failure→commitment trace.
- Written from `DETAILS.md`, `api.md`, and the *kept* (non-excised) source only.
  `gold.patch` was never opened. Pristine `/app` was used only to read untouched
  sibling files — never the excised ones (one exception, see Process note below).
- `Inferable:` obeyed per line: `yes`/`doc` → exact assertion; `partially` → only
  the derivable half; `no` → SHAPE (error-vs-not, termination, presence/absence,
  ordering, bounded-ness) never a literal outcome a solver could not derive.
- Contract at `_author/contract.md`, one row per test, written after the suite.

Four-gate check per unit (`verify_hidden.sh`, docker, `git apply`):
excised+bare → **FAIL** · excised+gold → **PASS** · excised+cheat → **FAIL** ·
gold.patch touches no `*_test.go` → **OK**.

## Result: all 20 units satisfy all four gates

| unit | lines | inferable breakdown (doc/partially/no/yes) | bare | gold | cheat | A12 |
|---|---|---|---|---|---|---|
| binio | 12 | 4/4/4/0 | FAIL | PASS | FAIL | OK |
| capability | 12 | 1/3/8/0 | FAIL | PASS | FAIL | OK |
| cgenc | 12 | 4/4/4/0 | FAIL | PASS | FAIL | OK |
| filechange | 10 | 1/4/4/1 | FAIL | PASS | FAIL | OK |
| indexenc | 12 | 3/7/2/0 | FAIL | PASS | FAIL | OK |
| inforefs | 11 | 8/2/0/1 | FAIL | PASS | FAIL | OK |
| lsrefs | 13 | 1/7/5/0 | FAIL | PASS | FAIL | OK |
| modconfig | 12 | 5/1/6/0 | FAIL | PASS | FAIL | OK |
| negside | 10 | 1/4/5/0 | FAIL | PASS | FAIL | OK |
| objfile | 12 | 3/4/5/0 | FAIL | PASS | FAIL | OK |
| pathutil | 12 | 5/4/3/0 | FAIL | PASS | FAIL | OK |
| reflog | 12 | 3/3/6/0 | FAIL | PASS | FAIL | OK |
| reportstatus | 10 | 1/6/3/0 | FAIL | PASS | FAIL | OK |
| reqframe | 13 | 4/6/3/0 | FAIL | PASS | FAIL | OK |
| revfile | 12 | 2/4/6/0 | FAIL | PASS | FAIL | OK |
| sigblock | 10 | 3/5/2/0 | FAIL | PASS | FAIL | OK |
| srvresp | 10 | 1/4/5/0 | FAIL | PASS | FAIL | OK |
| tagparse | 12 | 8/2/2/0 | FAIL | PASS | FAIL | OK |
| unidiff | 12 | 1/8/3/0 | FAIL | PASS | FAIL | OK |
| updreq | 12 | 1/8/3/0 | FAIL | PASS | FAIL | OK |

Totals: 231 detail lines → 231 tests → 231 contract rows.
Inferable totals: doc 60, partially 90, no 79, yes 2.

## What was asserted for each `Inferable: no` (shape, not literal)

- **binio 5,7,8,10,11**: non-termination/termination and bounded-write shape on
  negative VLQ input; `ReadUntil` delimiter exclusion + EOF/underlying-error
  shape; `IsBinary` on a `(0,nil)` reader terminates false/nil; consecutive-call
  independence (not the pool symbol).
- **capability 4,5,7,8,9,10,11,12**: no-value `Add` doesn't clear; `name=` ⇒ one
  empty-string value vs bare `name` ⇒ none; empty list ⇒ len-0 `All`, zero bytes;
  delete+re-add lands at end; empty-arg error precedes unknown-name; three
  DISTINCT validation failure modes + symref multi-value + required-arg members;
  non-printable session-id rejected + oversize rejected (bound unpinned);
  agent = kept base + optional non-blank env suffix.
- **cgenc 8,10,11,12**: oversized gen-v2 ⇒ flagged u32 slot + u64 overflow area
  recoverable in file order (no GDO2 TOC sig pinned); deterministic encode +
  optional chunks absent when unneeded; overflow queue contents (slice-reuse
  mechanic unobservable); empty index still emits OIDF(1024)/OIDL/CDAT.
- **filechange 2,3,4,6,9**(partial): name-only entry counts as PRESENT; non-file
  entry ⇒ (nil,nil,nil) no error under a real storer; display name prefers FROM
  (substring, not format); `\r` never stripped, interior `\r` keeps its line;
  String/malformed asserted by substring.
- **indexenc 10,11**: ext-flags word counted in padded footprint (total-length
  invariant); trailer = sha1 of stream / 20 zeros under skip-hash.
- **lsrefs 2,5,8,9,11**: bad prefix ⇒ error + zero arg frames; ≥bound prefixes ⇒
  NOT a capped list (drop-fallback); symref line carries `symref-target:` and
  either resolved hash or `unborn`; bare `unborn` produces no reference;
  `symref-target:` decodes to SymbolicReference type.
- **modconfig 2,4,5,10,11,12**: bad name/path never reach map while empty
  path/URL DO land in map; empty-name marshal still emits a submodule section
  via path label; empty branch fields absent (`HasOption` false); unset renders
  a third non-empty spelling; unknown options survive marshal; validation order
  via which sentinel surfaces.
- **negside 2,3,4,6,9**: two decoders DISAGREE on missing flush (one accepts
  EOF) — which-is-which unpinned; wrong-length/unrecognised lines never silently
  accumulate; one frame per option; flush ⇒ non-nil empty Options.
- **objfile 4,6,9,10,11**: reader tolerates-or-refuses negative size without
  panic while writer refuses `ErrNegativeSize`; `ErrHeaderNotRead` + zero
  format-sized hash before header; overflow reported AND fitting bytes committed
  (inflated stream verified); write-before-header never returns nil.
- **pathutil 4,6,10**: `~N` digit range is bounded non-empty (bound unpinned);
  plain `.git`/`git~1` VALID to `WindowsValidPath` vs disguises fail; tilde
  expansion returns original on failure, unchanged on bare `~`/`~x`.
- **reflog 2,3,5,6,7,8**: blank lines ⇒ no phantom entries, unterminated tail
  ±accepted; malformed middle ⇒ error + only well-formed entries back;
  `>`-in-angles never truncates email (or line rejected); oversized seconds
  field never decodes; `+2460` never yields zero offset; encoded tz reflects
  the timestamp's own zone.
- **reportstatus 2,3,5,8**: non-ok unpack status reaches `Error()` carrying the
  status text (Decode-error polarity free); well-formed ok/ng lines populate in
  order; non-`ok` status encodes a non-`ok ` line carrying name+status.
- **reqframe 7,10,11**: EOF-first ⇒ prompt return + empty Command; degenerate
  first lines ⇒ prompt return + no half-parse into Command; second field's
  value survives into Host-or-param (which field unpinned).
- **revfile 4,6,8,9,10,12**: objCount=0 ⇒ `ErrEmptyReverseIndex`; post-checksum
  bytes ⇒ malformed; nil and typed-nil writers rejected without panic; hasher
  SIZE selects hash-fn id (sha1→1, sha256→2); dirty hasher still produces a
  decodable file; short pack checksum ⇒ malformed.
- **sigblock 6,9**: sig header + continuations + chained second sig header all
  dropped while other content survives; unterminated tail copied through.
- **srvresp 2,3,4,7,8**: NAK terminates + never recorded as ack (success/error
  polarity free); bare ACK appends (last-line semantics free); unrecognised
  status ⇒ error or non-named status; empty response ⇒ exactly one data packet;
  encode ⇒ first frame `ACK `+first hash (emit-all-vs-first free).
- **tagparse 9,12**: name-only tagger ⇒ still emits tagger line; a
  `gpgsig-sha256` line at tagger position still lands in SignatureSHA256
  (pushback re-serves).
- **unidiff 3,6,8**: same-hash metadata change ⇒ no index/`---`/`+++`/hunk;
  post-`@@` tail is empty-or-` heading`; ctx=0 ⇒ no ` ` context lines.
- **updreq 4,9,11**: shallows+flush ⇒ valid empty request vs bare-flush fails;
  padded name preserved verbatim; caps appear after first NUL (space-leader
  spelling free).

## Refusals / lines not fully asserted (recorded in each contract)

- **cgenc 1**: >255-chunk refusal not exercised (input size impractical vs value).
- **objfile 8**: over-bound *writer* clause unreachable via public API
  (max legal header = 31 bytes < 32). Noted in contract.
- **pathutil 7**: volume-name-prefix clause unobservable off Windows
  (`filepath.VolumeName` no-op on Linux).
- **updreq 8**: trailing-payload-after-flush half — gold tolerates it; only the
  derivable "missing flush fails" is asserted.
- **filechange 9 / inforefs-style formats**: exact `Print` punctuation asserted
  by substring rather than literal equality.

## Fixes made during gold=PASS convergence (all test-side, spec stays)

- binio: writer-side instrumentation replaced by non-termination shape (gold
  buffers internally — observable commitment is "never returns").
- cgenc: GDA2 entries are u32 (not u64); overflow u64s sit between last TOC
  offset and checksum — test repointed at the documented chunk structure.
- filechange: `Files()` returns (from,to) — insert ⇒ nil from (test had swapped);
  `Lines` splits on `\n` only so interior `\r` stays mid-line; dir entries need a
  real storer (nil storer panics inside resolution, unreachable in practice).
- indexenc: padding counted relative to entry start (test double-counted header).
- modconfig: `x:y` IS a drive-letter prefix (moved to reject list); empty
  path/URL entries land in the map (they do not skip — corrected my reading).
- objfile: dropped unreachable >32-byte public-API header case.
- pathutil: `HasUnsafeComponent` covers `..`-canonicalisations + control bytes,
  not `.git` (probed gold behavior, tests rewritten to committed text).
- srvresp/reqframe/reportstatus/lsrefs/negside/reflog/revfile/updreq: `no`-marked
  outcomes rewritten to shape assertions where first drafts pinned literals.
- unidiff: `---` file header excluded from delete-line count.

## Process note — information barrier

One barrier slip occurred: while checking `cgenc` constants, a grep surfaced
~15 lines of the *pristine* `plumbing/format/commitgraph/encoder.go` (an excised
file — equivalent to reading gold). The cgenc suite was subsequently written
strictly from `DETAILS.md` + the kept `doc.go`/`index.go` + format docs, and its
final assertions were re-derived from the documented wire layout rather than the
seen code. No other excised/pristine-file reads occurred; `gold.patch` files
were never opened for any unit.

## Files

- `tests/hidden/**` — 20 suites, 231 `TestDetailNN` funcs.
- `_author/contract.md` — 20 contracts, 231 rows.
- `gen_test_sh.sh` generated `tests/test.sh` per unit (checksum-gated install
  into `/app` + `go test -run TestDetail`).
- `verify_hidden.sh` — the four-gate harness (reusable for later batches).
