# VERIFIER.md — tasks_bigL0

Hidden black-box property suites for the two excised L0 units.
Seed `20260919`. Built with `affordance.py` at L0 (bugreport.md) and L2
(contract.md). Dockerfile `FROM ladder-base:client-go-obf`. No Harbor launch.

Build:

```
uv run python scripts/build_bigl0.py
```

Gold/cheat patches were applied blind (`patch -p1`); the suite was never
edited to match gold. If gold failed a property, that property was widened
or dropped from over-specification — never the gold tree.

## `keyspacecodec-obf`

- **L0:** `experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf-L0`
- **L2:** `experiments/harbor_nex/tasks_bigL0/keyspacecodec-obf-L2`
- **Image:** `bigl0-keyspacecodec-obf:l0`
- **Packages:** `internal/apicodec`, `internal/locate`

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| A raw single-key lookup of user key `key` in keyspace 0x1092 must be sent with the 4-byte raw-mode header `0x72 0x00 0x10 0x92` followed by the user key, not the bare user key. Encoding twice on the same original request yields the same prefixed bytes (the original is not overwritten). | `TestCodecContractExamples / raw get 0x1092 + encode-twice` |
| User ranges whose start or end is empty expand to the keyspace interval: empty start becomes the keyspace prefix, empty end becomes the next keyspace prefix; both empty becomes the whole keyspace; interior start/end keep their user suffix under the same prefix. | `TestCodecContractExamples / empty+partial+interior ranges` |
| Constructing a v2 codec rejects a keyspace id that does not fit in 24 bits and rejects an unknown mode; the prefix is a mode byte plus the 24-bit id; the exclusive end prefix is that 32-bit value plus one, with carry across bytes; the last raw id wraps the mode byte from `r` to `s`. | `TestCodecContractExamples / construction + last-id wrap + carry` |
| Epoch-not-match region lists are clipped to the keyspace and decoded to user keys: a region covering the whole keyspace becomes empty/empty, a region wholly outside the keyspace is dropped, and a region overlapping the keyspace is truncated to the overlap then stripped of the header. | `TestCodecClipProperties / epoch region list` |
| A codec built for keyspace 4242 reports that same id. | `TestCodecContractExamples / keyspace 4242` |
| An MPP dispatch must carry keyspace id 4242 and API version 2 on its task meta, and its coprocessor ranges must be encoded the same way as ordinary user keys. Compact payloads carry the same API version and keyspace id on RPC context. | `TestCodecContractExamples / MPP + compact RPC context` |
| Bucket split keys that mix the previous, current, and next keyspace, including empty sentinels, decode to the user keys in this keyspace (`a`,`b`,`c`) with empty sentinels at the clipped bounds — never the mem-comparable encoded forms. | `TestCodecContractExamples / buckets a,b,c` |
| A 4-byte header starting with txn `x` or raw `r` followed by 0x010203 yields keyspace id 0x010203; a header whose mode byte is neither, or that is shorter than 4 bytes, errors and yields the all-ones null id 0xffffffff. | `TestCodecContractExamples / WillowNode parse` |
| API v2 splits a well-formed key into a 4-byte header and the remaining user bytes; API v1 is identity (no header); an invalid v2 mode byte errors with empty results. | `TestCodecContractExamples / LumenSeal v1/v2` |
| A request type with no key payload (store safe-ts) still encodes without error under the transactional v1 codec. | `TestCodecContractExamples / v1 store-safe-ts` |
| A truncated or otherwise non mem-comparable region key is a fatal decode: locating that region must fail without incrementing the backoff counter. | `TestLocateMalformedRegionNoBackoff` |
| Encoding a user key or range and then decoding it returns the original bytes; the same round-trip holds for region-boundary keys. Byte order is preserved inside a keyspace. | `TestCodecKeyRangeRoundTrip / 10k seeded round-trip + order` |
| Bucket split keys from the previous keyspace clip to an empty user start, from the next keyspace to an empty user end, interior keys drop the header, complement keys are not leaked, and interior order is preserved. | `TestCodecClipProperties / 10k seeded buckets` |
| Round-trip, range, and request/response symmetry hold for transactional ids and keys the contract examples never name (not 0x1092/0x010203, not `key`/`a`/`b`/`c`). | `TestCodecUnmentionedRandom / unseen ids and keys` |

### Proofs (C5 harness first, then excised / gold / cheat)

| check | result |
|---|---|
| `proof_harness` | pass |
| `harness_ok` | pass |
| `image_harness` | pass |
| `buggy_fails` | pass |
| `docker_a0_buggy_fails` | pass |
| `gold_restore` | pass |
| `docker_a0_gold_pass` | pass |
| `cheat_rejected` | pass |
| `patches_skip_tests` | pass |
| `blackbox_hygiene` | pass |

## `batchcmds-obf`

- **L0:** `experiments/harbor_nex/tasks_bigL0/batchcmds-obf-L0`
- **L2:** `experiments/harbor_nex/tasks_bigL0/batchcmds-obf-L2`
- **Image:** `bigl0-batchcmds-obf:l0`
- **Packages:** `internal/client`

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| Sending a packed entry on a canceled context must fail with cancellation; sending with a zero (or already-expired) timeout must fail with deadline exceeded — not a generic batch-unavailable error. | `TestBatchCancelDeadlineProperty / 10k seeded cancel+deadline` |
| Ten (and other) unforwarded pending entries build as one packed send; extra sizes the worked example does not name still succeed. After reset-equivalent live traffic the send path stays usable (id allocator is not stuck at zero). | `TestBatchPackedSizesAndCancelSkip / extra sizes 7/11/13/16` |
| Interleaving an empty host with forwarded hosts yields one stream per host. Extra hosts and per-host counts beyond the 1/2/3/4 fixture still group by host. | `TestBatchStreamGroupingProperty / extra forwarded hosts` |
| Canceled pending entries are omitted from the live stream; later live sends still complete. | `TestBatchPackedSizesAndCancelSkip / live cancel skip` |
| Canceling a waiter fails that waiter with cancellation (cancel-all of an already-canceled context). | `TestBatchCancelDeadlineProperty / forwarded cancel` |
| With batching on and one connection, three prewrites to the same forwarded host share one stream, so a server metadata checker fires once per host, not once per request. | `TestBatchStreamGroupingProperty / sequential prewrites 1,2,3,4,…` |
| A unary-stream coprocessor call is not packed; each call hits the checker once, and a forwarded host still appears in metadata. | `TestBatchStreamGroupingProperty / coprocessor stream` |
| Forwarded hosts and batch sizes that the contract examples never name still share one stream per host and complete. | `TestBatchUnmentionedRandom / unseen hosts` |

### Proofs (C5 harness first, then excised / gold / cheat)

| check | result |
|---|---|
| `proof_harness` | pass |
| `harness_ok` | pass |
| `image_harness` | pass |
| `buggy_fails` | pass |
| `docker_a0_buggy_fails` | pass |
| `gold_restore` | pass |
| `docker_a0_gold_pass` | pass |
| `cheat_rejected` | pass |
| `patches_skip_tests` | pass |
| `blackbox_hygiene` | pass |

## Rules

| rule | how |
|---|---|
| B4 | exported API / pre-existing constructors only; packaging refuses white-box tokens |
| B5 | seed 20260919, ≥10k cases, adversarial edges, unseen-random inputs |
| A12 | gold/cheat copied unread; `rule_verdicts` flags `*_test.go` hunks |
| C5 | `go` on PATH, `false`≠0, `true`=0 before any REWARD is trusted |
| A1/A8 | excised image fails; gold.patch on that image passes |
| A3 | cheat.patch on the excised image fails |

`task.toml`: `[agent]` allowlist Cursor hosts; `[verifier] network_mode = "no-network"`.

