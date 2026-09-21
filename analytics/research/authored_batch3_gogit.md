# Authored batch 3 — go-git (20 units)

Date: 2025-12-15. Branch: `au2gogit` (worktree `oswt-AU2gogit`). No commits.

Dataset math that motivated this batch: 86 certified / 224 authored (38% hit rate);
300–400 certified needs ~900 authored. go-git cohort went 14/25 flipped, 25/25 hard at L0,
so this batch doubles down on the surface class that produced that: **parsing, predicate and
serialisation closures over the same gold tree** (`ladder-base:go-git`, module obfuscated to
`example.internal/gitkit/v6`).

## Method

- Closures chosen only on files untouched by the 25 existing go-git units (batch2 closure map).
  Preference order: wire-protocol codecs > object/format codecs > config/path predicates.
- Excision is mechanical: `scripts/ops/author_excise.py` wraps `scripts/excise_funcs.py`,
  then runs a compiler-driven pass that blank-imports whatever `go build` reports unused
  (the exciser's textual usage check counts doc comments — the compiler is authoritative).
  `excision.patch`, `gold.patch` produced by `git diff --no-index`; `gold.patch` is
  test-file-stripped by construction.
- Every `DETAILS.md` line carries `Inferable: yes|doc|partially|no`. `doc` means the kept
  doc comments in the excised tree state the commitment (this tree is unusually doc-rich —
  several comments cite upstream git source lines). `no` marks deliberate arbitrary choices —
  error-vs-silent, ordering, bound placement — where only shape should be asserted.
- `bugreport.md` = symptom + reproduce command, no symbol/file/line names (checked by grep:
  zero leak lines across all 20).
- Overlap: `scripts/check_unit_overlap_batch3.py` (adapted from
  `check_unit_overlap.py`, which hardcodes batch2-vs-authored) — new units vs the 25-unit
  batch2 cohort AND pairwise within the batch. **Result: CLEAN, 0 overlaps, 0 shared-file
  warnings** (one intentional adjacency: `indexenc`/`cgenc`/`tagparse` keep their sibling
  decoders readable — different files, so no symbol-level overlap).
- Cheat validity: each cheat special-cases the worked path only — naive wire literals, no
  guard edges, no verification. `cheat_validity.py` expects `tests/` layout so ratios were
  computed on added-lines of `_author/cheat.patch` vs `_author/gold.patch` (same rule,
  < 0.6). All 20 pass; max 0.58 (`objfile`, where gold is short), most ≤ 0.40.
- Builds: `scripts/ops/verify_unit.sh` runs `go build` on excised, excised+gold, and
  excised+cheat variants for every unit — **20/20 all three OK**.
- Removed-test hygiene: units delete only the tests that exercise the excised surface
  (whole-package suites where the package is single-purpose); sibling tests for intact
  decoders/types are left as the "unrelated tests" that must keep passing.

## Per-unit table

| unit | closure (files) | surface | details | yes | doc | partially | no | overlap | cheat+ | gold+ | ratio |
|---|---|---|---|---|---|---|---|---|---|---|---|
| updreq | plumbing/protocol/packp `updreq{,_decode,_encode}.go` | parse+ser | 12 | 0 | 1 | 8 | 3 | clean | 48 | 222 | 0.22 |
| reportstatus | packp `report_status.go` | parse+ser | 10 | 0 | 1 | 6 | 3 | clean | 37 | 111 | 0.33 |
| srvresp | packp `srvresp.go` | parse | 10 | 0 | 1 | 4 | 5 | clean | 30 | 92 | 0.33 |
| inforefs | packp `inforefs.go` | parse+ser | 11 | 1 | 9 | 2 | 0 | clean | 24 | 60 | 0.40 |
| lsrefs | packp `lsrefs.go` | parse+ser | 13 | 0 | 1 | 7 | 5 | clean | 43 | 208 | 0.21 |
| reqframe | packp `command.go`,`gitproto.go` | parse+ser | 13 | 0 | 4 | 6 | 3 | clean | 62 | 186 | 0.33 |
| negside | packp `shallowupd.go`,`pushopts.go` | parse+ser | 10 | 0 | 1 | 4 | 5 | clean | 37 | 107 | 0.35 |
| capability | protocol/capability `list.go`,`capability.go` | predicate+ser | 12 | 0 | 1 | 3 | 8 | clean | 72 | 187 | 0.39 |
| objfile | format/objfile `reader.go`,`writer.go` | ser (zlib) | 12 | 0 | 3 | 4 | 5 | clean | 67 | 116 | 0.58 |
| reflog | format/reflog `reflog.go` | parse+ser | 12 | 0 | 3 | 3 | 6 | clean | 65 | 167 | 0.39 |
| revfile | format/revfile `decoder.go`,`encoder.go` | ser+verify | 12 | 0 | 2 | 4 | 6 | clean | 88 | 300 | 0.29 |
| indexenc | format/index `encoder.go` | ser | 12 | 0 | 3 | 7 | 2 | clean | 66 | 199 | 0.33 |
| unidiff | format/diff `unified_encoder.go` | ser | 12 | 0 | 1 | 8 | 3 | clean | 106 | 257 | 0.41 |
| cgenc | format/commitgraph `encoder.go` | ser+bitpack | 12 | 0 | 4 | 4 | 4 | clean | 75 | 233 | 0.32 |
| tagparse | object `tag.go`,`tag_scanner.go` | parse+ser | 12 | 0 | 8 | 2 | 2 | clean | 75 | 271 | 0.28 |
| modconfig | config `modules.go`,`branch.go`,`optbool.go`,`config.go`(1fn) | predicate+ser | 12 | 0 | 5 | 1 | 6 | clean | 95 | 185 | 0.51 |
| sigblock | object `signature.go` | predicate+ser | 10 | 0 | 3 | 5 | 2 | clean | 52 | 104 | 0.50 |
| filechange | object `file.go`,`change.go` | predicate | 10 | 1 | 1 | 4 | 4 | clean | 58 | 103 | 0.56 |
| pathutil | internal/pathutil `dotgit,path_util,tree,hfs,ntfs.go` | predicate | 12 | 0 | 5 | 4 | 3 | clean | 70 | 215 | 0.33 |
| binio | utils/binary `read.go`,`write.go` | ser+predicate | 12 | 0 | 4 | 4 | 4 | clean | 71 | 137 | 0.52 |

Totals: 20 units, 235 details — yes 2, doc 72, partially 95, no 66. Cheat ratios 0.21–0.58
(mean ≈ 0.38). Overlap: 0.

## Why each surface is parsing/predicate/serialisation

- `updreq`, `reportstatus`, `srvresp`, `inforefs`, `lsrefs`, `reqframe`, `negside`:
  pkt-line protocol codecs — byte-in/struct-out and struct-in/byte-out, the measured-best
  surface class. `reqframe`/`negside` pair two asymmetric codecs so missed edges differ
  between directions (flush-required vs flush-tolerated, EOF semantics).
- `capability`, `modconfig`, `sigblock`, `pathutil`, `filechange`: predicate-heavy
  (validation tables, name canonicalisation, signature-block detection, change
  classification) plus a serialisation half (capability list, .gitmodules, header
  stripping). Predicate units are where arbitrary-choice details concentrate.
- `objfile`, `reflog`, `revfile`, `indexenc`, `unidiff`, `cgenc`, `tagparse`, `binio`:
  format codecs — loose-object zlib, reflog lines, RIDX state machine, index v2/3/4,
  unified diff, commit-graph chunks, tag objects, offset-VLQ. Several keep the sibling
  decoder readable: layout is `Inferable: doc`, semantics (ordering, guards, teardown)
  stay hidden.

## Distribution notes

- `inforefs` is deliberately doc-dominated (9/11 doc): a control for "documented edge,
  still missed" — the doc comment is unusually complete yet past units show solvers still
  misread it.
- `no`-heavy units (`capability` 8, `modconfig` 6, `reflog`/`revfile`/`negside`/`lsrefs` 5):
  the hidden surface is the choice itself (skip-vs-error, order, bound), not the format —
  expected to be the flip drivers.
- `filechange` is the thinnest closure (predicted L3) — included as a floor probe: if it
  also comes back hard, the predicate-surface rule holds even at small scope.

## Notes for the verifier stage

- DETAILS annotations verified: every numbered line has exactly one `Inferable:` label
  (the only counter hit outside items is the inforefs preamble NOTE, not a detail line).
- gold patches touch zero `*_test.go` lines (checked); cheat ditto.
- Excision specs kept under `outputs/specs/<unit>.json` for re-runs;
  `author_excise.py` is resumable (re-runs rebuild from pristine).
- Known cosmetic quirk: `indexenc`/`revfile`/`tagparse` etc. ship `excised/tree/` pruned
  to touched files (batch2 convention); the full repo + `excision.patch` is the task input.
