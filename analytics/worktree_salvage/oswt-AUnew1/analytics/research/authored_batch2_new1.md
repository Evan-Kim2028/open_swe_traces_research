# authored_batch2 — NEW repo #1: go-git (25 units)

Date: 2026-09-19. Job: `AUnew1` (task-author role only — no hidden tests
written, no solver trials, no commits).

Repo #1 from `oswt-NEWREPOS/analytics/research/new_repos_2026-09-19.md`:
**go-git** @ `0f3a0a2c25513f2666ac9b88a6745f7b2382f572`, rebranded
`example.internal/gitkit/v6`, tree at
`experiments/pipeline/repos2/go-git/src`, base image `ladder-base:go-git`.
The `rebrand:` entry was already present in this worktree's
`experiments/pipeline/repos.yaml`.

## How the survey was used

The NEWREPOS survey listed 15 promising go-git closures with the
specific behavioural commitments each offers. The 25 units map onto all
15; where one survey closure held two independent commitment clusters it
was split so each unit keeps a disjoint file set:

- config machinery → `cfgdecode`, `cfgencode`, `cfgsection`, `cfgurl`
- revision parsing → `revparse`
- ref names / refspecs → `refnames`, `refspec`
- pkt-line framing → `pktline`
- ignore + attributes matching → `ignorepattern`, `ignorescope`, `gitattrs`
- index + pack decoding → `indexdec`, `indexops`, `packdelta`, `packscan`,
  `idxdecode`
- commit/tree objects → `commitobj`, `treeobj`
- upload-request / advertised-refs / sideband protocol → `ulreq`,
  `advrefs`, `sideband`
- endpoints, ref-hash ids, filesystem refs+reflogs, merkle diffing →
  `endpoints`, `objectid`, `fsrefs`, `merklediff`

Authoring was FOR difficulty per
`authoring_hard_l0_units.md`: every unit's contract commits to edge
behaviour that is *not* inferable from signatures alone — empty input vs
empty-message sentinels, reserved values, ordering rules (sorted refs
but peeled-under-base, directory-order merge, last-wins sections),
duplicate handling, error-vs-zero conventions, and context/flag stickiness.
The in-tree tests that pinned each commitment were removed by the
excision patch (the discoverability knob); `DETAILS.md` records which
removed test covered each numbered commitment.

Two closures were widened during authoring: `fsrefs` was extended from
the thin `storage/filesystem` delegation layer into `dotgit_setref.go` +
`dotgit_rewrite_packed_refs.go` (where the check-and-set, locking and
packed-refs rewrite commitments actually live), and its test removal
grew to both `storage/filesystem` and `storage/filesystem/dotgit` test
dirs to keep the tree compiling.

## Units

| unit | closure | files | lines | details | predicted_flip | commitments | compiles |
|---|---|---|---|---|---|---|---|
| advrefs | plumbing/protocol/packp | 3 | 514 | 15 | L2 | caps only on first line behind NUL; zero-id `capabilities^{}` marker; `ErrEmptyAdvRefs` vs `ErrEmptyInput`; ref-after-shallow rejection; peeled emitted under base not sorted; v1-only version line; HEAD heuristic (master then first non-peeled same-hash) | yes |
| cfgdecode | plumbing/format/config | 2 | 235 | 10 | L2 | section last-case-insensitive-match vs exact subsection; header-only sections; options append; non-empty targets; EqualFold-orbit fold key ("ſ"≡"s") | yes |
| cfgencode | plumbing/format/config | 1 | 83 | 7 | L2 | header elided without options; quoting on six chars or edge spaces; five escapes (not \r); `key = ` even empty; literal tab; options-before-subsections | yes |
| cfgsection | plumbing/format/config | 2 | 315 | 11 | L2 | asymmetric comparators (folded section/option vs exact subsection); backward-scan last-wins; create-on-miss; remove-ALL; SetOption keep-if-listed/append-rest ordering | yes |
| cfgurl | config | 1 | 90 | 8 | L2 | longest-prefix insteadOf wins; ties → config order; empty insteadOf never matches (length>0); plain prefix no boundary check; duplicate url merge; multi-valued collect in order | yes |
| commitobj | plumbing/object | 2 | 941 | 16 | L2 | `tree` strictly first; out-of-position author/committer + late parents silently dropped; first-wins duplicates; repeated gpgsig concatenated; one-space continuation strip; EncodeWithoutSignature prefers raw bytes when fields match | yes |
| endpoints | internal/url | 1 | 156 | 8 | L2 | scp port 1–5 digits between colons; digit path segments stay paths; `/` before first `:` forces local; Windows-only drive-letter rule; `\`-start rejected; SCP→file→URL order; empty `ParseFile`; host:port kept joined | yes |
| fsrefs | storage/filesystem (+dotgit) | 5 | 354 | 13 | L2 | old-ref forwarding on CAS; truncate-only-when-no-old; stat-read-compare fallback with different error string; packed rewrite rename-then-copy; loose-wins-over-packed; packed fallback only on missing/empty/dir | yes |
| gitattrs | plumbing/format/gitattributes | 4 | 544 | 14 | L2 | line grammar (quotes, macros, state prefixes, `=` split, name charset); matching differs from gitignore (last-component simple match, embedded `**` never matches); matcher priority + macro expansion; loader error swallowing | yes |
| idxdecode | plumbing/format/idxfile | 2 | 510 | 10 | L3 | `Stat` before any read; size-formula gate pre-allocation (min+max, (nr−1)·8 allowance); fanout monotonicity names entry; 64-bit table keyed on high bit of first offset byte read as trailing block; idx checksum read after pack checksum; empty buckets keep noMapping | yes |
| ignorepattern | plumbing/format/gitignore | 2 | 593 | 12 | L2 | parse state (`!`, `\ ` escape, dirOnly, glob-vs-name); name patterns hit any component; globs anchor at domain end; `**` traverses but `foo**` doesn't; trailing `**` needs component-or-dir; last-to-first scan; ASCII-only classes | yes |
| ignorescope | plumbing/format/gitignore | 2 | 323 | 12 | L2 | excluded-ancestor stickiness flips match short-circuit/readOwn laziness/frozen descendants; nil/empty readOwn returns same pointer; exclude-file errors swallowed but root errors propagate; info/exclude gated on `.git` in listing | yes |
| indexdec | plumbing/format/index | 1 | 593 | 14 | L2 | zero timestamps stay `IsZero` not epoch; 0xFFF name-length flag → NUL scan with terminator counting toward padding; V4 strip-length typed errors incl. first-entry-must-be-zero; all-zero trailer skips checksum; invalidated TREE entries consume no OID | yes |
| indexops | plumbing/format/index | 2 | 434 | 10 | L2 | `*` stops at `/` (filepath.Match semantics); SkipUnless is plain prefix not glob; NTFS/Unicode `.git` variant rejection; Remove returns removed entry; zero-time `IsZero` on adjacent entries | yes |
| merklediff | utils/merkletrie | 4 | 1011 | 15 | L2 | file-only emission (empty dir insert/delete silent); file→empty-dir emits only delete; both-dirs-diff-hash descends not blasts (empty side flips wholesale, both-empty-diff-hash errors); Skip() excluded everywhere; directory-order merge; shallow-vs-deep advance; ErrCanceled | yes |
| objectid | plumbing | 1 | 154 | 11 | L2 | format inferred from hex LENGTH (64→SHA256 else SHA1, any even length accepted); FromBytes strict 20/32; failed ReadFrom zeroes id; compare/render honour Size() but Equal/IsZero scan full array; ResetBySize mapping | yes |
| packdelta | plumbing/format/packfile | 1 | 692 | 15 | L2 | zero size field means 65536; `0x00` → `ErrDeltaCmd` not `ErrInvalidDelta`; PatchDelta rejects empty source while streaming accepts zero base; target-size zero is legal no-op; literal-op payload-length check | yes |
| packscan | plumbing/format/packfile | 2 | 686 | 17 | L2 | empty vs bad vs short signature = three sentinels; version failure unwrapped but mid-scan EOF wrapped; zero-count pack emits no header section; OFS base predicate `>=` at self-offset; inflate bound in two paths; limit+1 sentinel reader | yes |
| pktline | plumbing/format/pktline | 3 | 446 | 14 | L2 | length prefix includes itself; 0003 invalid but 0004 real empty packet; sentinels return sentinel-as-length nil payload; `ERR ` typed errors trimmed; undersized-buffer drain-to-resync; peek non-nil empty slice for 0004; uppercase-hex read/lowercase write; nil distinctions | yes |
| refnames | plumbing | 1 | 518 | 12 | L2 | `@{` banned only as pair; bare `@` banned; IsSafe `[A-Z_]` vs IsRoot `[A-Z_-]` + six-name allowlist + `_HEAD` suffix; branch-HEAD check on spliced name vs `-` on shorthand; Short's last-successful-extraction chain | yes |
| refspec | config | 1 | 158 | 10 | L2 | exactly-one-`:` and not-last-char (empty src ok, empty dst not); wildcard parity per side <2 cap; `+` meaningful only at byte 0; glob = prefix+suffix empty-middle ok; Dst index-arithmetic substitution; Reverse keeps `+`; first-byte flag readers panic on empty | yes |
| revparse | internal/revision | 3 | 772 | 14 | L3 | `@`-word vs `@{` lookahead; `^{}`→tag default; `^N` capped at 2; `@{-n}` alone-only; `:` stage digits 0-3 fold into path otherwise; `!`/`!-` escapes; `.lock` only before `/` or end; chunk-ordering validity matrix | yes |
| sideband | plumbing/protocol/packp/sideband | 2 | 218 | 9 | L2 | chunk budget `cap−5` (prefix and channel byte both charged); Write counts payload not wire bytes; partial-chunk buffering; flush = EOF with trailing packets ignored; zero-payload pkt-line is error not EOF; channel-3 `unexpected error`; unknown channel echoes payload; nil Progress discards | yes |
| treeobj | plumbing/object | 2 | 946 | 16 | L2 | unsorted-tree lookup early-breaks past `name/` sort key; Decode invalidates subtree path cache; Validate joins all errors + four symlinked metadata names + 4096-byte name bound + deprecated mode accepted; permissive-decode/gated-encode split | yes |
| ulreq | plumbing/protocol/packp | 3 | 325 | 15 | L2 | caps only on first want line; sort+dedup wants and shallows; deepen-since forced UTC; mutual exclusion both directions; only flush may follow `deepen <n>`; `deepen 0` = no-limit, negative fails; filter rejected on decode; empty stream → `pkt-line 1: EOF` | yes |

Totals: 25 units, 53 files, ~11.2k source lines, 316 commitments.
predicted_flip L2 ×23, L3 ×2 (`idxdecode`, `revparse` — larger wire/parser
closures where the survey suggests an L2 contract may not fully
disambiguate).

## Verification

- Excision: each `excised/excision.patch` stubs bodies, keeps
  declarations/imports (unused imports blanked `_`), deletes the
  in-tree tests that exercised the commitments.
- Compile: all 25 excised trees pass `go build` + `go test -run NONE`
  for their packages (driver `author_batch2_gogit.py` → `problems: 0`);
  every `gold.patch` applies (restores, never touches `*_test.go`) and
  every `cheat.patch` applies and compiles.
- Cheat: literal worked-example implementations only — each omits the
  ordering/empty/error-taxonomy commitments, so property tests fail it.
- Overlap: `check_unit_overlap.py --extra` from the main checkout →
  `CLEAN`, 0 overlaps vs existing authored repos (go-git is new) and no
  file sharing inside the batch.
- Artifacts: all 25 units carry non-empty `api.md`, `contract.md`,
  `bugreport.md`, `closure.md`, `difficulty.md`, `DETAILS.md`,
  `gold.patch`, `cheat.patch`, `excised/excision.patch`.

## Known limitations

- `predicted_flip` is authorial judgement, not measured — the verifier
  session supplies the hidden tests that confirm it.
- `fsrefs` removes 17 test files across two dirs; that is heavier
  discoverability removal than most units, but required for
  compilation (dotgit test suites share helpers).
- `ulreq` stubs only 3 functions; the rest of its difficulty rides on
  constants/types kept in `common.go` — commitments are still pinned by
  removed tests and the contract.
