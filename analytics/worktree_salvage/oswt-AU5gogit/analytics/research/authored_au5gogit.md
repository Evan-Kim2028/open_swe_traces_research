# authored_au5gogit — 20 new go-git units

Batch of 20 behavioral units authored against the obfuscated go-git tree
(`example.internal/gitkit/v6`), branch `au5gogit`. Bank baseline: 45
existing units (25 batch2 in `oswt-AUnew1`, 20 batch3 here). Overlap
check (`scripts/check_unit_overlap.py`) against both cohorts: CLEAN, 0
shared file+symbol. All three trees (excised / gold / cheat) build for
every unit via `scripts/ops/verify_unit.sh`. Cheat ratios via
`scripts/ops/cheat_validity.py`: all `< 0.6` ("plausible-cheat").
No solver trials were run. Nothing committed.

## Units in authoring order

| # | unit | closure | surface | rej. before | cheat ratio |
|---|------|---------|---------|-------------|-------------|
| 1 | fetchmsg | packp/fetch.go upload-request encode/decode | wire codec | 0 | 0.41 |
| 2 | packenc | packfile encoder.go + object_pack.go | serialization | 0 | 0.46 |
| 3 | packparse | parser.go + parser_cache.go state machine | parser | 1 | 0.33 |
| 4 | cgfile | commitgraph file.go reader (chunks, guards) | binary parser | 0 | 0.51 |
| 5 | deltadiff | diff_delta.go + delta_index.go | algorithmic codec | 0 | 0.52 |
| 6 | deltasel | delta_selector.go windowing/depth policy | selection policy | 0 | 0.04 |
| 7 | renamedet | rename.go similarity scoring | algorithmic | 1 | 0.00 |
| 8 | mergebase | merge_base.go ancestor paint walk | graph algorithm | 0 | 0.18 |
| 9 | objpatch | patch.go diff stats + patch building | serialization | 0 | 0.01 |
| 10 | commitwalk | commit_walker*.go iterator family | traversal | 0 | 0.13 |
| 11 | idxindex | idxfile.go + lazy_index.go | index structure | 0 | 0.37 |
| 12 | packhandle | internal/packhandle FD lifecycle + meta | resource lifecycle | 0 | 0.44 |
| 13 | sigcodec | object.go Signature codec + blob.go | serialization | 1 | 0.24 |
| 14 | revwalk | revlist/object_walk.go paint walk | graph algorithm | 0 | 0.31 |
| 15 | fsnode | merkletrie filesystem noder | hashing/policy | 0 | 0.33 |
| 16 | negotiate | transport negotiate.go haves batching | protocol state machine | 1 | 0.33 |
| 17 | archive | internal/archive tar/zip writer | serialization | 0 | 0.49 |
| 18 | packlookup | packfile.go/fsobject/iter/handle read path | read dispatch | 0 | 0.50 |
| 19 | memstor | storage/memory full storer | storage semantics | 1 | 0.56 |
| 20 | ioutil | utils/ioutil common+context+sync | io helpers | 0 | 0.35 |

258 behavioral commitments total: doc=71, partially=135, no=52, yes=0.

## Rejected candidates

Overlap collisions (never re-excised):
- `commit.go`/`commit_scanner.go` → commitobj (batch2)
- `tag.go`/`tag_scanner.go` → tagparse
- `signature.go` → sigblock
- `tree.go`/`treenoder.go` → treeobj
- `file.go`/`change.go` → filechange
- `internal/revision/parser.go`,`scanner.go` → revparse
- `plumbing/reference.go` → refnames
- `format/index/{decoder,encoder,index,match}.go` → indexdec/indexenc/indexops
- `packfile/{scanner,scanner_reader,patch_delta}.go` → packscan/packdelta
- merkletrie `{change,difftree,doubleiter,iter}.go` → merklediff
- gitignore `{pattern,matcher,dir}.go` → ignorepattern/ignorescope
- config decoders/sections/urls → cfgdecode/cfgencode/cfgsection/cfgurl/modconfig
- packp `{ulreq,updreq,advrefs,inforefs,lsrefs,...}` → existing packp units
- dotgit/path files → pathutil/fsrefs

Rejected on surface type, not overlap:
- `upload_pack.go`/`receive_pack.go`/serve paths — session orchestration
- `worktree_status.go`/`blame.go`/`worktree_commit.go` — porcelain
  orchestration over already-covered internals
- `internal/fsnoder` — test-DSL helper, not product surface
- `utils/diff/diff.go` alone — too thin (61 lines)
- `remote.go`/`repository.go` — orchestration

Per-unit rejection counts are candidates evaluated-then-dropped during
the survey immediately preceding each acceptance.

## Tooling fixes made this batch

- `scripts/excise_funcs.py`: body-open brace now found by scanning `{`
  candidates and skipping type-literal keywords (`struct{}`, composite
  literals) — previously mangled signatures like
  `func f() (map[H]struct{}, error)`. Body depth initialized to 1 and
  `stub_file` uses the recorded body-open column.
- Textual import pass skips `v\d+` path elements (`go-billy/v6` no
  longer mistaken for a package name); compiler-driven blanking in
  `author_excise.py` remains the authority on unused imports.

## Search cost

Flat through unit ~17 — every survey round found a viable closure with
at most one rejected candidate. Slight rise at 18–20: the pure
parsing/serialization inventory was largely drained, pushing into
resource-lifecycle (packhandle), protocol state (negotiate) and storage
semantics (memstor) — still certifiable surfaces per the funnel, but
thinner. The next batch would face real scarcity in this repo:
remaining untouched surface is mostly porcelain orchestration
(worktree_*, remote.go) and transport session glue, both disfavored
by the funnel data. Stopping at 20 was correct.
