# authored_au5natsserver — 20 new units, nats-server

Worktree `oswt-AU5natsserver`, branch `au5natsserver`. Obfuscated source
at commit `8ad52657` (module `example.internal/msgkit/v2`). All units
verified: bare excision builds, gold passes restored tests, cheat builds
and fails the same tests. `scripts/check_unit_overlap.py`: CLEAN — 0
symbol overlaps within the batch and against `authored_batch2` (shared
files are disjoint-symbol only: filestore.go hosts four units,
mqtt.go three, websocket.go two).

Surface mix: 9 wire/record codecs, 5 predicate clusters, 4 config/arg
parsers, 2 header/subject string transformers.

## Rejected candidates (overlap or coverage)

| Candidate | Reason rejected |
|---|---|
| `server/sublist.go` trie (Sublist.Match etc.) | banked `gslsublist` covers the same domain (server/gsl) |
| `tokenizeSubjectIntoSlice` (sublist.go) | byte-identical copy inside banked `gslsublist` — dropped from `subjvalid` mid-authoring, regenerated |
| `server/ipqueue.go` | banked `ipqueue` |
| `server/avl/seqset.go` | banked `seqset` |
| `server/thw` | banked `hashwheel` |
| `server/cron.go` | banked `cronparse` — kept as scaffolding for `schedcodec` |
| `server/util.go` parsers | banked `utilparse` — `parseSize` kept as scaffolding for `leafmsg` |
| `server/conf` parse side | banked `confparse` — `conflex` took the disjoint lexer instead |
| `stree`, `proxyproto`, `jwtvalidate`, `ldapdn`, `jsversioning`, `subjecttransform`, `archiveio` | all banked in batch2 |
| gateway reply predicates (`isGWRoutedReply*`) | no direct tests; gateway integration only |
| leafnode validators (`validateLeafNodeAuthOptions`, proxy check) | coverage only via ProcessConfigFile integration; thin |
| `DirJWTStore.Pack/Merge` | filesystem-coupled, not a pure codec |
| `ackReplyInfo` + consumer reply decoders | exercised only by JetStream cluster tests (30–120s each); viable but costly — held in reserve |
| `isJSONObjectOrArray`/`isEmptyRequest` | real but thin (2 small predicates, one table test) — reserve |

## Units in authoring order

| # | unit | closure | surface | rejects before accept | Inferable y/d/p/n | ratio |
|---|---|---|---|---|---|---|
| 1 | conflex | conf lexer value cluster (30 stubs, lex.go) | lexer | confparse (banked) | 0/5/6/4 | 0.06 |
| 2 | protoparse | parser.go state machine + protoSnippet + args (5) | protocol parser | — | 0/0/7/8 | 0.12 |
| 3 | mqttwire | mqtt.go packet read/write codec (18) | wire codec | — | 0/6/1/0 | 0.45 |
| 4 | mqtttopics | topic↔subject map + validators (12) | string transform | — | 0/3/4/3 | 0.31 |
| 5 | wsframe | websocket frame codec (10) | wire codec | — | 0/4/3/2 | 0.52 |
| 6 | wshandshake | upgrade helpers + origin check (6) | predicates | — | 0/3/3/2 | 0.44 |
| 7 | raftcodec | raft wire encode/decode incl. vote pair (15) | wire codec | — | 2/2/5/0 | 0.59 |
| 8 | consumerstate | consumer/stream state codec, RLE deletes (13, store.go+filestore.go) | record codec | — | 3/2/5/2 | 0.51 |
| 9 | hdrsurgery | client.go header splice + JS-ack classifiers (13) | header parse | — | 4/2/3/1 | 0.45 |
| 10 | exportauth | accounts.go export-approval predicate chain | predicates | — | 3/1/4/0 | 0.57 |
| 11 | mqttpersist | mqtt retained-msg codec + flags (3) | record codec | — | 0/3/4/0 | 0.44 |
| 12 | msgtrace | msgtrace.go header-map sampler + conn-name precedence | transform | — | 3/3/3/0 | 0.51 |
| 13 | mondecode | monitor.go query decoders + myUptime + JWT redact (7) | arg parse | — | 1/2/2/0 | 0.47 |
| 14 | optsparse | opts.go scalar parsers: duration/URL/storage/compression (8) | config parse | — | 4/1/2/0 | 0.57 |
| 15 | leafmsg | leafnode.go LMSG/LHMSG arg parsers + R/N/L key builders (4) | protocol parse | leafnode validators (weak cov) | 2/0/4/0 | 0.34 |
| 16 | schedcodec | scheduler.go MsgScheduling codec + parseMsgSchedule (3) | record codec | cron.go (banked, scaffolded) | 0/1/5/0 | 0.54 |
| 17 | blkcodec | filestore cmp-metadata sniff + StoreCompression codec + msgBlock.decode w/ collision fallback (8) | record codec | — | 1/0/4/0 | 0.47 |
| 18 | subjvalid | sublist.go subject-syntax predicates + SubjectsCollide (13) | predicates | gslsublist (dropped duplicate symbol) | 2/0/5/0 | 0.36 |
| 19 | msgrecord | filestore record decoder msgFromBuf* + size accounting + errBadMsg (8) | record codec | — | 1/0/6/1 | 0.59 |
| 20 | sourcescodec | sources.db codec + JSStreamSource header v1/v2 (5, filestore+stream) | record codec | ipqueue, ackreply (reserve) | 1/0/4/0 | 0.52 |

## Notable mechanics

- `mkunit.sh` gained function-level test snipping (`snipspec`) after
  `parser_test.go` proved to hold cross-file helpers; `verifyunit.sh`
  restores snipped test files before gold/cheat runs.
- `msgrecord`: a first cheat panicked inside a `mb.mu`-held call site
  and deadlocked the next permutation (600s timeouts). Fix: return a
  distinct `errBadMsg` instead of panicking — failures must be fast and
  assertion-shaped, not hangs.
- `subjvalid`: post-hoc overlap audit caught `tokenizeSubjectIntoSlice`
  — a byte-identical copy banked under `gslsublist`. Removed and
  regenerated before acceptance.

## Diminishing returns

Search cost per accepted unit stayed roughly flat through unit 18:
units 15–20 each took 1–2 candidate inspections, all verified on first
or second cheat attempt. The pure-parser seam is not exhausted but is
thinning: the remaining strong candidates (`ackReplyInfo`, leafnode
validators, `isJSONObjectOrArray` cluster, DirJWTStore Pack/Merge) are
either integration-only coverage or I/O-coupled, and the overlap bank
now covers most obvious pure surfaces (util, cron, gsl, ipqueue, avl,
thw, stree). A 21st+ unit would start dipping below the quality bar —
stopping at 20 as instructed.
