# xrepo round 2 — cross-repo closures that resist name-alone reconstruction

Round 1 proved the construction (excise L, hidden tests drive C's public API) but picked
famous utilities: 23 trialled, 4 certified (17% vs 46% bank-wide), 19/23 solved at L0.
The dependency boundary adds nothing when the solver has already memorised the dependency.
Round 2 applies the corrected rule: **excise behaviour the model cannot recall, only infer
from call sites** — obscure or project-specific dependencies, arbitrary-but-consistent
rules (encodings, ordering, precedence, state transitions), and ≥3 distinct consumer call
sites with different arguments.

Selection filter applied per unit: *could a competent engineer who has never seen L write
this from the name and signature alone?* All 15 below answer **no**; the rationale column
says which arbitrary decisions the name does not supply.

Units live under `experiments/pipeline/authored_xrepo2/<unit>/_author/` with
`gold.patch`, `cheat.patch`, `excised/excision.patch`, `api.md`, `DETAILS.md`,
`bugreport.md`, `closure.md`, and `hidden/` tests.

## Units

### chrootbind — go-billy `helper/chroot` → go-git

- **L:** `deps/billy/helper/chroot/chroot.go` (chroot-style filesystem wrapper inside
  go-billy). **C:** go-git (`PlainInit`/`PlainOpen` dotgit binding) plus billy's own
  `memfs`/`osfs` (3 internal construction sites with different bases).
- **Call sites constraining L:** ≥4 distinct (`chroot.New` at memfs memory.go, osfs
  os_rootfs.go, os_js.go; plus go-git worktree→dotgit scoping).
- **Not recallable:** "chroot" suggests prefix-join only. The real contract is a
  component-wise `Lstat` walk with a symlink-rewrite queue, **absolute targets reset the
  walk to the bound root** while relative ones prepend, ELOOP after 40 hops with a
  self-link short-circuit, and a per-operation table of which calls follow the final
  component.
- **Name-alone test:** fails — abs-reset, the 40-hop cap, and the follow-final table are
  arbitrary; a name-driven impl gets prefix-strip only.

### errtrace — pingcap/errors fork → client-go

- **L:** `deps/errors` (pingcap fork of pkg/errors: `errors.go`, `stack.go`,
  `juju_adaptor.go`, `group.go`). **C:** client-go `internal/unionstore` pipelined memdb
  error paths, `internal/apicodec`, txnkv flush — ~429 `errors.*` call sites, plus every
  module whose package init builds sentinels via `errors.New`.
- **Call sites:** broadest in the set; sentinel identity, `Errorf` stack capture, typed
  error extraction, and `ErrorGroup` traversal are all exercised through consumer APIs.
- **Not recallable:** the verb-to-layout table (`%s`/`%v` bare, `%q` quoted, `%+v`
  message + `funcname\n\tpath:line` frames), `withMessage` printing the cause's `%+v`
  **before** the annotation (opposite of upstream pkg/errors), dedup-via-`HasStack`
  marker protocol, `ErrorGroup` depth-first order.
- **Name-alone test:** fails — a pkg/errors recollection produces the wrong `%+v` order
  and misses the marker protocol; the fork's deviations are the graded surface.

### fsutil — go-billy `util` → go-git

- **L:** `deps/billy/util` (`util.go`, `walk.go`, `glob.go`). **C:** go-git
  `Worktree.AddGlob`, status cleanup, packfile/index helpers — ~47 `util.*` references.
- **Call sites:** `util.Glob` (worktree_status.go:450), `util.RemoveAll`
  (x/plumbing/worktree), `TempFile`, `WriteFile`, `ReadFile` across ~10 consumer files.
- **Not recallable:** `**` is **not** recursive — it matches a single level like `*`;
  meta-free patterns return nil-not-error on missing paths; `cleanGlobPath`'s
  `dir == pattern` → `ErrBadPattern` recursion stop; dir-matches-then-file-matches result
  ordering; `RemoveAll` never crosses symlinks.
- **Name-alone test:** fails — every engineer assumes `**` means recursive descent;
  glob-over-Filesystem semantics are a per-implementation choice.

### godifflines — sergi/go-diff `diffmatchpatch` → go-git

- **L:** `deps/godiff/diffmatchpatch/diff.go`. **C:** go-git `utils/diff` (line-mode
  `Do`/`DoWithTimeout`/`Src`/`Dst` wrappers), `plumbing/object/patch.go` (2 sites),
  `blame.go` (hunk attribution via DiffEqual/Insert/Delete).
- **Call sites:** 4 consumer files, 7+ distinct entry points with different arguments
  (line diffs, timeout path, reconstruction, blame walk).
- **Not recallable:** the library is famous at the char level; the excised surface is the
  **line-mode pipeline**: lines keyed by hash including trailing `\n`, a bespoke rune
  packing that skips surrogate/invalid codepoint ranges, the `diffCompute` fallback
  cascade (empty→single op, containment→prefix+eq+suffix, half-match→recurse,
  checklines→lineMode>100 runes), and the post-pass that re-diffs each del+ins block
  char-by-char — the reason replacements return as tight pairs.
- **Name-alone test:** fails — `diffLines`/`diffCompute` internals are not in the public
  memory of the library; the consumer asserts op shapes only this pipeline produces.

### jwtdecode — nats-io/jwt v2 decoders → nats-server

- **L:** `deps/jwt` decoder files + claims decode path (`decoder*.go`, `claims.go`
  parse/verify, `genericlaims.go`). **C:** nats-server `opts.go` (`operator:` file,
  `resolver_preload`, `trusted_keys`), `server.go` (`ReadOperatorJWT`,
  `verifyAccountClaims`), `accounts.go`, `auth.go` — 32 `jwt.*` decode call sites.
- **Call sites:** operator file read, resolver_preload map validation, store Pack/Fetch
  round-trip, creds auth — each with different claim types.
- **Not recallable:** claim dispatch by `nats.type` (v2) vs top-level `type` (v1) with
  GenericClaims fallback; `alg` must be exactly `ed25519-nkey` (v1's `ed25519` rejected);
  **v1 signatures verify over `chunks[1]` only while v2 verify over
  `header.payload`**; `ExpectedPrefixes` gates issuer role after signature passes.
- **Name-alone test:** fails — JWT libs are famous but the version-sniffing rule, the v1
  sig-scope quirk, and the issuer-role gate are nats-specific.

### jwtmigrate — nats-io/jwt v1compat → nats-server

- **L:** `deps/jwt` decoder files including the v1→v2 migration path. **C:** same
  nats-server surface as jwtdecode; hidden tests drive v1-format tokens through the
  consumer's decode entry points.
- **Call sites:** shares the 32-site decode surface; distinct arguments are the v1
  operator/account/user fixtures.
- **Not recallable:** v1 detection = top-level `type` field present (v2 nests it in
  `nats`); the **field re-homing table** (v1 `account_server_url`,
  `operator_service_urls`, `system_account` → v2 `Operator` block; v1 `nats.limits`
  splits into AccountLimits+NatsLimits); migrated claims **keep `Version = 1`** rather
  than rewriting to 2; `issuer_account` → `IssuerAccount` mapping.
- **Name-alone test:** fails — the migration table and version-preservation rule are
  arbitrary; no amount of JWT knowledge supplies them.

### memdborder — pingcap/goleveldb `memdb` → client-go

- **L:** `deps/leveldb/leveldb/memdb` (`memdb.go`, `key.go`). **C:** client-go
  `internal/mockstore/mockkv` (`NewMVCCLevelDB` per column family; every Raw*/MVCC op)
  and `internal/unionstore` tests using `memdb.New` as a golden oracle — 47 call sites.
- **Call sites:** raw put/get/scan/delete, batch gets, MVCC prewrite/commit/scan/
  reverse-scan through timestamped internal keys.
- **Not recallable:** internal key = `ukey || LE u64(seq<<8 | keyType)` with keyType
  0=del/1=val; ordering is ukey ascending then packed num **descending** (newest first);
  `keyTypeSeek = keyTypeVal` (the *highest* type) because the type occupies the low 8
  bits of a descending num; skiplist arena layout (`kvData`/`nodeData` int records,
  tMaxHeight 12); geometric randHeight p=1/4 seeded `0xdeadbeef`.
- **Name-alone test:** fails — "an ordered in-memory map" says nothing about the byte
  packing, descending num, or seek-key convention that MVCC reads depend on.

### memfsfile — go-billy `memfs` file → go-git

- **L:** `deps/billy/memfs/file.go`, `memory.go` (file/content layer). **C:** every
  go-git op through `billy.Filesystem` on mem-backed repos.
- **Call sites:** worktree writes, index/config I/O, pack reads — the whole consumer
  exercises it, plus direct multi-handle opens.
- **Not recallable:** two `Open`s of a path share `content` but keep separate
  `position`; `ReadAt`/`WriteAt` do **not** move the sequential cursor; `WriteAt` gap-
  fills with zeros and never truncates; `Duplicate` shares content but resets position
  to 0; flag matrix gates later calls.
- **Name-alone test:** fails — POSIX intuition covers part; the shared-content/position
  split and gap-fill policy are implementation choices the tests pin.

### memfsstore — go-billy `memfs` store → go-git

- **L:** `deps/billy/memfs/memory.go`, `storage.go` (node/store layer — disjoint symbol
  set from memfsfile's closure on the shared file). **C:** `git.Init`/`PlainInit`
  plumbing, worktree ops, mem-backed repo fixtures.
- **Call sites:** full consumer surface plus `Rename`/`Symlink`/`ReadDir` paths.
- **Not recallable:** dual `files`+`children` index kept consistent on every mutation;
  `Rename` moves **every descendant path** (`HasPrefix(path, parent+"/")`) rewriting both
  indexes; `O_CREATE` implicitly creates missing parents; `Stat` resolves symlinks
  lexically inside the fs without escaping root; symlink target stored verbatim.
- **Name-alone test:** fails — subtree-rename and implicit-parent-create are deliberate
  extras, not defaults anyone would guess.

### nuid — nats-io/nuid → nats-server

- **L:** `deps/nuid/nuid.go`. **C:** nats-server — 53 `nuid.*` call sites (cluster name,
  JetStream IDs, account event IDs, consumer IDs, pin IDs, reload request IDs, hashes).
- **Call sites:** widest single-symbol surface; `nuid.New()` and `nuid.Next()` with
  different roles per site.
- **Not recallable:** 22 chars = 12-char random prefix + 10-char sequential suffix;
  alphabet is `0-9A-Za-z` (digits **first**); sequential increments by a random step in
  [33, 333); on overflow the prefix re-randomizes and the sequence restarts — never grows
  to 11 digits; package `Next` shares one global locked NUID.
- **Name-alone test:** fails — "unique ID" gives nothing; the split, alphabet order,
  increment band, and overflow policy are all arbitrary constants.

### osbound — go-billy `osfs` bound/rootfs → go-git

- **L:** `deps/billy/osfs/os_bound.go`, `os_rootfs.go`. **C:** go-git
  `PlainInit`/`PlainOpen`/`PlainClone` — 15 BoundOS/osfs call sites; every on-disk repo
  path goes through it.
- **Call sites:** init/open/clone plus worktree file ops under bound roots.
- **Not recallable:** three-way path mapping — host-absolute path inside baseDir is
  **rebased** to baseDir-relative, any other absolute path is **clamped** to the root,
  relative joins normally; Windows-volume escape only when baseDir is fs root;
  `os.Root`-scoped symlink confinement (in-root absolute links resolve post-rebase,
  out-of-root links are contained/fail); `Readlink` returns the target verbatim.
- **Name-alone test:** fails — "bound os fs" could mean chroot, path-prefix, or filter;
  the rebase/clamp/join split and `os.Root` semantics are version-specific choices.

### securejoin — cyphar filepath-securejoin → Helm

- **L:** `deps/securejoin/join.go`. **C:** Helm `pkg/chart/v2/util.Expand` and
  `internal/chart/v3/util.Expand` (chart-name join + per-member join) and
  `internal/plugin/installer` extractor — 5 call sites, different arguments.
- **Call sites:** 2 sites × 2 chart versions + 1 plugin extractor.
- **Not recallable:** root containing `..` is rejected outright (wrap-in-slashes search
  trick); component-wise `Lstat` walk; symlink dest + `/` + remaining is **prepended** to
  the queue and **absolute dests clear `currentPath`** (rebased under root);
  `*os.PathError{ELOOP}` after `MaxSymlinkLimit` hops; missing components resolve as
  non-symlinks.
- **Name-alone test:** fails — "secure join" names the goal, not the algorithm; the
  abs-reset and hop-budget rules are the graded contract (and the cheat patch must hit
  ELOOP, not just refuse).

### strkey — nats-io/nkeys strkey/crc16/keypair → nats-server

- **L:** `deps/nkeys/strkey.go`, `crc16.go`, `keypair.go`. **C:** nats-server `opts.go`
  (leafnode `nkey:` seeds, `resolver_pinned_accounts`, auth_callout `xkey`,
  `trusted_keys`), `server.go` (identity keypair), `auth.go` — 32 call sites.
- **Call sites:** seed parsing, public-key validation per role, keypair creation —
  distinct arguments per site.
- **Not recallable:** base32 RFC 4648 **without padding**; payload =
  `prefix_byte || raw_key || crc16-LE(prefix||key)` — the little-endian CRC16-CCITT
  trailer is appended *before* base32; public keys pack the role in ONE byte
  (`role << 3`) while seeds split it across a TWO-byte boundary so text starts
  `S<role>`; `Prefix` returns `PrefixByteUnknown` on undecodable input instead of
  erroring; per-prefix strict validators.
- **Name-alone test:** fails — the crc16 trailer, the split-pack seed format, and
  failure-as-Unknown are wire-format decisions invisible from `strkey`.

### uitable — gosuri/uitable → Helm

- **L:** `deps/uitable/table.go`, `strutil.go`, `wordwrap.go`. **C:** `helm list`,
  `helm history`, `helm repo list`, `helm search repo/hub`, `helm plugin list`,
  `helm dependency list`, `pkg/cli/output.EncodeTable` — 13 uses across 9 files.
- **Call sites:** ≥7 distinct commands rendering different table shapes (headers,
  wrapping, right-align, empty).
- **Not recallable:** column width = max `Cell.LineWidth` over all rows incl. header,
  `MaxColWidth` clamp; truncation keeps runes while `w+rw < n-3` then appends `...`
  (ellipsis eats the last three columns); ANSI escapes stripped before width math,
  wide runes count double; multi-line cells fan out into a line matrix with blank-
  padded short cells.
- **Name-alone test:** fails — "ASCII table" supplies rows+columns but not the n-3
  truncation budget, ANSI-aware width, or the multi-line fan-out.

### xkeys — nats-io/nkeys `xkeys.go` (X25519) → nats-server

- **L:** `deps/nkeys/xkeys.go`. **C:** nats-server `server.go` (`CreateCurveKeys` server
  identity, `s.xkp`, `info.XKey`), `opts.go` (`IsValidPublicCurveKey` on auth_callout
  config), `auth_callout.go` (payload sealing) — ~4 distinct sites.
- **Call sites:** keypair creation, config validation, seal/open round-trip through the
  callout path.
- **Not recallable:** the seed **is** the private scalar (`PublicKey` =
  `curve25519.ScalarBaseMult(seed)`, not ed25519); wire format `xkv1 || nonce(24) ||
  box.Seal(...)` with the literal `xkv1` tag and **no embedded sender key**; `Open`
  maps failures to distinct sentinels (`ErrInvalidEncrypted`, `ErrInvalidEncVersion`,
  `ErrInvalidSender`, `ErrCouldNotDecrypt`); `Seal` rejects non-`X` recipients as
  `ErrInvalidRecipient`, `Open` as `ErrInvalidSender`; `Sign`/`Verify` error out.
- **Name-alone test:** fails — "x keys" gives X25519 at best; the version tag, absent
  sender key, and per-failure sentinel split are unguessable.

## Verification

- `uv run python experiments/xrepo2/tools/verify_hidden.py` — **15/15 PASS**
  (excised-state tests fail, gold-state tests pass), `outputs/XREPO2.log` 03:28 run.
- Overlap (`scripts/check_unit_overlap.py` machinery, ad hoc driver): 15 xrepo2 units vs
  127 existing (124 bank + 3 round-1 xrepo) — **0 symbol overlaps**. One informational
  pair: `memfsfile`/`memfsstore` share `deps/billy/memfs/memory.go` with disjoint symbol
  closures (file layer vs store layer).

## Fixture note

`jwtdecode`'s hidden test originally embedded a user JWT minted with a 1-hour `exp`
claim — a time bomb: the unit passed at 00:54 and failed at 03:18 purely because the
fixture expired (StorePack filtered the expired user). Regenerated all 8 constants
(operator/account/user/server pubs + 4 JWTs) with no `exp` on a verified gold tree.
Caution discovered mid-fix: a first regeneration ran while the scratch tree still had
`cheat.patch` applied, producing subtly malformed claim JSON (limits flattened into
`nats`); fixtures must be minted only on the pristine gold tree. Worth a gate note: any
hidden fixture embedding `exp`/`iat` must be checked against wall-clock decay.
