# verified_vfgrokgogitgogit — hidden suites for the 10 authored_grokgogit go-git units

Date: 2026-09-21. Branch `vfgrokgogitgogit`. Docker image `ladder-base:go-git`
(digest `sha256:8e91c0a4…b515`, Go 1.26.x, module `example.internal/gitkit/v6`).
Raw gate output: `outputs/VFgrokgogitgogit.log`.

## Method

For each unit under `experiments/pipeline/authored_grokgogit/go-git/<unit>/`:

- Hidden suite at `tests/hidden/<pkg>/*_bb_test.go`, in-package (`package <pkg>`,
  not `*_test`), one `TestDetailNN` per DETAILS.md numbered line — the numbering
  is the failure→commitment trace.
- Written from `DETAILS.md`, `api.md`, and the *post-excision* tree only. The
  Grok cohort ships `excised/excision.patch` without a stub tree, so per-unit
  post-excision trees were materialized under `/tmp/gogit-exc/<unit>/` (patch
  applied to a pristine export) and only those were read. `gold.patch`,
  `cheat.patch`, `_author/cheat/`, and `bugreport.md` were never opened.
  Patch *headers* were used solely to enumerate affected paths.
- `Inferable:` obeyed per line: `yes`/`doc` → exact assertion; `partially` →
  only the derivable half; `no` → SHAPE (error-vs-not, sentinel identity,
  ordering, presence/absence, boundedness, byte-layout on committed
  structure), never a literal a solver could not derive.
- `tests/test.sh` generated per unit by `gen_test_sh.sh` (sha256-verified
  install into `/app`, `go test -run '^TestDetail'`).
- `_author/contract.md` written after the suite: one prose commitment per
  assertion plus a coverage table pairing every `TestDetailNN` with its
  DETAILS row. Every test has a row; every row has a test.
- Harness: `verify_hidden.sh` + `gen_test_sh.sh` copied from the verified
  authored_batch3 cohort (same image, same gates).

Four-gate check per unit (docker, `git apply`): excised+bare → **FAIL** ·
excised+gold → **PASS** · excised+cheat → **FAIL** · gold.patch touches no
`*_test.go` → **OK**.

## Result: all 10 units satisfy all four gates

| unit | lines | inferable breakdown (yes/doc/partially/no) | bare | gold | cheat | A12 |
|---|---|---|---|---|---|---|
| advrefs-encode | 9 | 2/3/1/3 | FAIL | PASS | FAIL | OK |
| config-quote | 8 | 3/0/1/4 | FAIL | PASS | FAIL | OK |
| fold-key | 8 | 0/3/0/5 | FAIL | PASS | FAIL | OK |
| gitattributes-match | 12 | 4/0/3/5 | FAIL | PASS | FAIL | OK |
| gitproto-nul | 12 | 3/0/2/7 | FAIL | PASS | FAIL | OK |
| hfs-dot | 10 | 5/0/2/3 | FAIL | PASS | FAIL | OK |
| index-v4-name | 8 | 1/1/2/4 | FAIL | PASS | FAIL | OK |
| instead-of | 8 | 3/1/1/3 | FAIL | PASS | FAIL | OK |
| refspec-map | 10 | 5/1/2/2 | FAIL | PASS | FAIL | OK |
| ulreq-encode | 13 | 7/2/3/1 | FAIL | PASS | FAIL | OK |

Totals: 98 detail lines → 98 tests → 98 contract rows.
Inferable totals: yes 33, doc 11, partially 17, no 37.

## What was asserted for each `Inferable: no` (shape, not literal)

- **advrefs-encode 2,3,4**: first-line selection asserted on observable
  ordering only (HEAD when present, else first non-peeled ref in input order);
  empty-ref advertisement asserted as name `capabilities^{}` + zero hash;
  first-line framing asserted as `hash SP name NUL caps` with the NUL present
  on empty caps.
- **config-quote 3,5,6,7**: subsection escaping asserted only for the two
  committed characters; quoting asserted as presence/absence of `"`…`"` on the
  committed trigger set plus two committed non-triggers; the five committed
  escapes asserted verbatim; off-set unusual bytes asserted emitted verbatim
  and unquoted.
- **fold-key 3,4,5(no-adjacent lines 6,7,8)**: orbit-minimum rule asserted via
  its committed consequences on known orbits (long s → `s`, Kelvin → `k`,
  ASCII capitals → lowercase); invalid-UTF-8 asserted as collision shape —
  differing bad bytes share a key, differing bad-byte counts do not.
- **gitattributes-match 3,5,7,11,12**: single-segment patterns asserted to hit
  the last component only; interior empty segments asserted skipped without
  consuming a path component; embedded `**` asserted to fail the whole match
  even on literal-identical text; malformed char class asserted false (no
  error, no panic); middle-component non-match asserted.
- **gitproto-nul 3,4,5,7,9,10,12**: control-byte rejection asserted per field
  on representative bytes; payload layout asserted byte-exact on one concrete
  request (`command SP pathname NUL [host=.. NUL] [NUL param NUL…]`); missing
  trailing NUL asserted decode failure; `host=` strip-or-keep shape asserted;
  empty extra-param drop asserted via double-NUL; sentinel asserted by
  `errors.Is` on `ErrInvalidGitProtoRequest`.
- **hfs-dot 2,3,6**: ignored code points asserted skipped in all three
  positions; every committed range member probed plus boundary outsiders
  (U+200B, U+00AD, U+2070) asserted not skipped; non-ASCII needle-position
  runes asserted to fail including foldable ones.
- **index-v4-name 2,4,5,7**: padding asserted through total-length and
  zero-byte content on the discriminating aligned case; strip varint and
  suffix+NUL asserted as exact byte regions on committed layout; `lastEntry`
  asserted to name the failing entry after a mid-write error.
- **instead-of 4,7,8**: equal-length tie asserted first-in-slice-order across
  two `URL` values; cross-slice longest-prefix asserted; zero-length
  `insteadOf` asserted as no-match on helper and public method.
- **refspec-map 2,8**: wildcard-count rule asserted via sentinel on clean
  single-violation inputs; `Dst` substitution asserted on the committed `*bc`
  example plus one parallel case.
- **ulreq-encode 9**: `deepen-since` asserted as `deepen-since <unix-seconds>`
  with the committed `UTC().Unix()` value.

## Deliberately not asserted (refusals/softening)

- **gitattributes-match**: a *trailing* empty pattern segment on an exhausted
  path is not asserted either way — rule 5 ("empty segments are skipped")
  and rule 10 ("path exhausted with segments remaining → fail") conflict
  there; gold resolves to fail. Conversely the trailing-`**`-on-exhausted-path
  case IS asserted (false) under rule 10, whose `yes` annotation commits the
  exhaustion check; the rule-6 early-true is asserted only where the `**` is
  reached with path remaining.
- **advrefs-encode 6**: peeled-ref ordering is asserted only when the base is
  a non-first line. Gold drops the peeled line when the base is the first
  advertisement line — an undocumented interaction that a solver cannot
  derive, so it is excluded from the suite.
- **index-v4-name 3**: input is fed unsorted only because the kept
  `encodeEntries` visibly sorts (`sort.Sort(byNameAndStage(...))`), making
  sorted-predecessor compression solver-derivable. If the sort were not kept,
  only sorted input would be asserted.
- **gitproto-nul 1**: asserted against `packp.ErrNilWriter`, the in-package
  sentinel declared in kept `common.go` — *not* `pktline.ErrNilWriter`, which
  shares the message but is a different var. (First suite version asserted
  the pktline sentinel; corrected without reading gold — the kept sentinel is
  what the spec text's "refuses … with ErrNilWriter" resolves to in-package.)
- No DETAILS line was left without a test; no test lacks a contract row.

## Test-side fixes made during gold convergence (no gold reads)

1. `advrefs-encode` TestDetail06: first version put the peeled ref's base on
   the first line; gold omits the peeled line there. Rebased the case onto a
   non-first base — the committed "immediately after `name`" ordering is
   still exercised.
2. `gitproto-nul` TestDetail01: `pktline.ErrNilWriter` → `packp.ErrNilWriter`
   (two same-message sentinels; the in-package one is the committed target).
3. `index-v4-name` TestDetail06: expected-bytes arithmetic corrected — byte
   prefix of `éz`/`ëz` is 1, so strip is `3-1=2` (`[2 0xab 'z' 0]`), not 1.
   Rune-wise would be `[3 0xc3 …]`; gold is byte-wise as committed.
4. `index-v4-name` TestDetail05: dropped a "no second NUL" probe that was
   actually reading the zeroed skip-hash footer, not name padding.
5. `ulreq-encode` TestDetail13: split the newline sweep into separate
   requests per depth flavor — `Deepen`+`DeepenNot` together trip the
   committed `ErrDeepenMutuallyExclusive`.
6. `gitattributes-match` TestDetail05/06/10: removed/refiled the two
   exhausted-path edge assertions described above.

## Cohort caveats

- The Grok excision stubs do not all compile standalone: several leave unused
  imports, and `config-quote`'s excised `encoder.go` carries a malformed stub
  (stray `)`, duplicated `valueReplacer` decl) — a syntax error, not just an
  unimplemented panic. Bare correctly FAILs (build error counts), and
  gold/cheat apply cleanly on top, so the gates still certify; flagged so a
  future cohort fixes stub generation.
- `gitproto-nul` shares `packp` with two sentinel errors of identical message
  (`ErrNilWriter`); suites must qualify assertions to the in-package var.
