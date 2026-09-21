# Verified: authored_batch5 nats-server hidden suites

2026-09-22. All 20 authored nats-server units now carry a hidden
`tests/hidden/**_bb_test.go`, a `_author/contract.md`, and a generated
`tests/test.sh`. Every unit passes all four Docker gates:

`EXCISED=FAIL GOLD=PASS CHEAT=FAIL A12=PASS` — the excised tree fails,
the gold patch passes, the cheat patch fails, and `gold.patch` modifies
no test file. Log: `outputs/VFbatch5natsserver.log`.

Method: assertions were written from each unit's `DETAILS.md`, `api.md`,
the excised stub's doc comments, and solver-visible kept code/tests.
`gold.patch` was never opened or used as a specification. Ambiguous
shapes were settled with scratch probes run against the pristine `/app`
tree inside `ladder-base:nats-server`. `Inferable: no` lines were
asserted as shape (error presence/kind, field named in the error,
bounds, state transitions), never literals.

## Results

| Unit | Tests | Inferable (doc-yes / partially / no) | Gates |
|---|---|---|---|
| blkcodec | 6 | 1 / 4 / 0 | all pass |
| conflex | 16 | 5 / 6 / 4 | all pass |
| consumerstate | 13 | 5 / 5 / 2 | all pass |
| exportauth | 8 | 4 / 4 / 0 | all pass |
| hdrsurgery | 10 | 6 / 3 / 1 | all pass |
| leafmsg | 7 | 2 / 4 / 0 | all pass |
| mondecode | 5 | 3 / 2 / 0 | all pass |
| mqttpersist | 7 | 3 / 4 / 0 | all pass |
| mqtttopics | 10 | 3 / 4 / 3 | all pass |
| mqttwire | 9 | 6 / 1 / 0 | all pass |
| msgrecord | 8 | 1 / 6 / 1 | all pass |
| msgtrace | 9 | 6 / 3 / 0 | all pass |
| optsparse | 8 | 5 / 2 / 0 | all pass |
| protoparse | 16 | 0 / 7 / 8 | all pass |
| raftcodec | 10 | 4 / 5 / 0 | all pass |
| schedcodec | 7 | 1 / 5 / 0 | all pass |
| sourcescodec | 6 | 1 / 4 / 0 | all pass |
| subjvalid | 8 | 2 / 5 / 0 | all pass |
| wsframe | 10 | 4 / 3 / 2 | all pass |
| wshandshake | 8 | 3 / 3 / 2 | all pass |

181 `TestDetailNN` tests across 20 units; every test maps to a numbered
DETAILS row and every row's asserted commitment is paired in the unit's
`contract.md` coverage table.

## What was asserted for the `Inferable: no` rows (23 total)

- **conflex** (4): second `.` in a float re-routes to string — asserted
  only that `127.0.0.1:4222` lexes as a single string item; bare-string
  emission order — asserted observable item kinds (escaped parts keep
  string type, bool spellings produce Bool, `$x` produces Variable with
  `$` stripped); map/value terminators and error shapes — asserted
  item sequence and that an error item appears, not message text.
- **consumerstate** (2): encoder preallocation — asserted only encoded
  length and round-trip, not the buffer cap; v1-compat — asserted the
  observable decode result (Delivered adjusted up by floor-1, v2/v1
  timestamp direction), not the format rationale.
- **hdrsurgery** (1): `removeHeaderStatusIfPresent` — asserted removal
  leaves `emptyHdrLine`→nil and that non-status headers are untouched;
  did not pin the internal `\r`-position precondition.
- **mqtttopics** (3): `//`/`/.` collapse direction — asserted
  round-trip pairs only; `*` handling and sparkplug birth/death topics —
  asserted accepted/rejected topic sets, not the scan mechanics.
- **msgrecord** (1): `errBadMsg.Error()` detail text — asserted the
  error occurs and includes the block basename, not the prose.
- **protoparse** (8): `PING`/`PONG`/`+OK` byte absorption — asserted
  `PING garbage\r\n` parses and garbage is dropped; `\r` handling in
  arg states — asserted arg content and `argBuf` never containing `\r`;
  split-buffer state, per-kind op gates, `clonePubArg` copy semantics,
  `protoSnippet` clamp, header-cache and parser-reset — all asserted on
  observable parser outputs (ops dispatched, errors raised, state
  cleared), never on internal constants.
- **wsframe** (2): `unmask` lane XOR — asserted cross-read masked bytes
  unmask identically to contiguous unmask; `wsGet` — asserted zero-copy
  vs read-through split and `pos` advancing only over consumed buffer
  bytes.
- **wshandshake** (2): `wsGetHostAndPort` — asserted host lowercased and
  default ports 443/80 by TLS flag; same-origin explicit-port rule —
  asserted the two asymmetric outcomes (explicit 443 vs implicit default)
  rather than the comparison internals.

## Corrections made while probing (DETAILS over-claims and my own errors)

Where probing showed a DETAILS line or a first-draft assertion
over-claimed, the test was brought back to the derivable commitment:

- **mqtttopics**: `*` is literal in MQTT topics (only `+`/`#` are
  wildcards), and `+` converts only as a complete `/`-delimited level —
  `"foo.bar.+"` stays literal while `"a/+/b"` → `"a.*.b"`.
- **wsframe D3**: `nbPoolGet` returns a larger slab (512B), not a
  14-byte buffer — asserted `len == header length` and
  `cap >= wsMaxFrameHeaderSize`. **D9**: `pos` is the read cursor;
  `buf[pos:pos+needed]`.
- **wshandshake D3**: the header-name argument is a direct map key
  (canonical form required); only *token* matching is case-insensitive.
  **D8**: request `host.com:443`+TLS vs origin `https://host.com`
  *passes* (443==443); the committed failure direction is implicit-port
  request vs explicit-port origin — asserted accordingly.
- **raftcodec D7**: `encodePeerState` always emits the trailing u16 —
  "optional" in the row is decode-side tolerance. **D10**: a
  trailing-slash path is accepted (`path.Base` strips it); removed the
  over-strict rejection, kept padded/extended/empty rejections.
- **subjvalid D5**: `mappingDestinationErr.Is` exposes only
  `ErrInvalidMappingDestination`; the unknown-function case is asserted
  as that identity plus "unknown function" in the message. A wildcard
  dest against a literal src passes token rules and fails later in
  `NewSubjectTransform` — asserted only that an error is reported.
- **protoparse**: `parseState` is embedded in `client` (`c.header`, not
  `c.ps`). Route/gateway/leaf ops panic on underconfigured dummy
  clients — the suite asserts the op-gate (client-kind rejects) rather
  than wiring full routes.
- **consumerstate**: minimum record length is magic+version only.
- **exportauth**: user JWTs must be signed by the account key.
- **leafmsg**: probe showed one allocation in arg parsing — the
  zero-alloc wording was not asserted; observable arg layout is.
- **hdrsurgery**: a prefix equal to the whole key has quirky residual
  behaviour — asserted strict-prefix removal only.
- **msgtrace**: a mid-block line without `:` merges into the next key —
  avoided asserting a "stops at malformed line" edge; `xsnap` is not
  snappy while `s2`/`S2` are; CLIENT always supports tracing.

## Excision-patch repairs (authoring defects, not test changes)

Two units' excision patches removed all uses of a retained `math`
import in kept test files, leaving a dangling import that broke every
build. Fixed inside the excision patch itself (blank import
`_ "math"`), matching the same fix the author applied to production
files: `raftcodec` (`server/raft_test.go`) and `schedcodec`
(`server/filestore_test.go`).

## Cheat-patch detection points

Each cheat was caught by a different committed behaviour: mqttwire
(`WriteUint16` mutation), msgtrace (CLIENT must always support
tracing), optsparse (explicit `false` values must be tracked),
sourcescodec (v2 header parse), mqttpersist (invalid retained flags),
msgrecord (corrupt-message error detail), mqtttopics (field-append
order), subjvalid (`numTokens("")`), wsframe (64MiB cap), wshandshake
(explicit-port same-origin), protoparse (error must quote the proto
excerpt), raftcodec (excised panic), schedcodec (excised panic), and
panic-on-stub for the codec units.

## Caveat

Probes ran against the pristine tree's *behaviour* only (compile +
run), reading declarations/doc comments that remain solver-visible in
the excised tree. No `gold.patch` content was opened or used.
