# spec-reimpl-bb — black-box property verifier

Date: 2026-09-18. Family: `spec-reimpl-bb` (does not replace `spec-reimpl`).
Question: A0/A1 for codec excision should measure **behavior**, not whether the
solver guessed white-box helper names (`ThornSlot`, `NimbusCore`, `*codecV2`).

Build:

```
uv run python scripts/build_spec_reimpl_bb.py \
  --obf-task experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf \
  --dest experiments/harbor_nex/tasks_unsolv \
  --upstream experiments/harbor_nex/tasks_iter9/client-go-keyspacecodec/environment/src
```

Hidden test: `src/openswe_traces/synth/testdata/codec_bb_prop_test.go`
(copied into `tests/hidden/internal/apicodec/codec_bb_prop_test.go`).
Seed `20260918`, 10 000 cases per property test. Callers enumerated from the
unobfuscated ITER_9 tree (`EncodeRequest` etc. in `client.go`, `pd_codec.go`,
`region_cache.go`) and mapped through
`experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf/mapping.json`.

## Exported API (caller-facing only)

Constructors: `NimbusPack` (NewCodecV2), `ZestRing` (NewCodecV1).
Package: `WillowNode` (ParseKeyspaceID), `LumenSeal` (DecodeKey), `JadeSeal`
(IsDecodeError).

`YarrowJoin` methods actually reached from outside `apicodec`:

| original | obfuscated | outside callers |
|---|---|---|
| EncodeRequest | MistCore | `internal/client/client.go`, `kvclient/test_util.go` |
| DecodeResponse | CedarPath | same |
| EncodeRegionKey | JadePort | `locate/pd_codec.go`, `region_cache.go` |
| EncodeRegionRange | SablePack | `pd_codec.go` ScanRegions |
| DecodeRegionRange | ThornUnit | `pd_codec.go` processRegionResult |
| DecodeBucketKeys | AmberGate | `pd_codec.go` processRegionResult |
| GetAPIVersion | QuartzSlot | `locate/region_request.go` |
| IsDecodeError | JadeSeal | `region_cache.go` |
| EncodeKey / DecodeKey / EncodeRange / DecodeRange / DecodeRegionKey / GetKeyspace / GetKeyspaceID | HazePipe / MistUnit / CedarUnit / QuartzPort / EmberSlot / LumenRef / ThornRef | on the interface held by those callers; used on the request/response path |

White-box names **not** used in the hidden suite: `ThornSlot`
(`codecV2.encodeKeyRanges`), `NimbusCore` (`decodeRegionError`), `*codecV2`,
`memCodec`, `PebbleUnit`, `IvoryLink`. Epoch clip is exercised through
`CedarPath` on a `RawGet` response; ranges through `CedarUnit`/`QuartzPort`.

## Properties → contract sentence

| property | contract sentence |
|---|---|
| TestCodecKeyRangeRoundTrip / key+range encode then decode | Encoding a user key or range and then decoding it returns the original bytes; the same round-trip holds for region-boundary keys. |
| TestCodecKeyRangeRoundTrip / monotonicity | If user key A is byte-wise less than B, both the ordinary encoding and the region-boundary encoding of A are byte-wise less than those of B. |
| TestCodecClipProperties / epoch region list | Epoch-not-match region lists are clipped to the keyspace: whole-keyspace → empty/empty, wholly outside dropped, overlap kept as user keys, order preserved. |
| TestCodecClipProperties / buckets | Previous keyspace → empty start, next → empty end, interior drops the header, complement not leaked, interior order preserved. |
| TestCodecContractExamples / raw get 0x1092 | Raw get of `key` in 0x1092 wires as `0x72 0x00 0x10 0x92` plus the user key. |
| TestCodecContractExamples / header parse+split | Txn/raw `.. 0x01 0x02 0x03` → id 0x010203; mode `t` is invalid; v2 split vs v1 identity. |
| TestCodecContractExamples / empty range + construction | Empty ranges in 0x1092 expand to `[0x72 0x00 0x10 0x92, 0x72 0x00 0x10 0x93)`; last raw id wraps `r`→`s`. |
| TestCodecContractExamples / MPP + buckets + v1 + fatal | MPP 4242 advertises id/API v2/encoded ranges; buckets `a,b,c` plus neighbors; v1 store-safe-ts encodes; truncated region key is a fatal decode. |
| TestCodecUnmentionedRandom / three unseen properties | Round-trip, range, and request/response symmetry on transactional ids/keys the instruction never names (not 0x1092/0x010203, not `key`/`a`/`b`/`c`). |

## Prove

Local + Docker image `harbor-obf-client-go-keyspacecodec:latest` (`--network=none`).
Details: `experiments/harbor_nex/tasks_unsolv/spec-reimpl-bb-validation.json`.

| impl | black-box suite |
|---|---|
| gold (obf gold.patch) | **pass** (host 0.12s; docker A0/A1/A2 pass) |
| buggy (excised identity stubs) | **fail** (bare user key on raw get; no clip) |
| off-by-one header (`prefix[:3]`) | **fail** (wire `72 00 10 6b 65 79` vs `72 00 10 92 6b 65 79`) |
| Composer 2.5 A0 patch (`spec-reimpl-A0__aojSyzA`, replayed from trajectory edits) | **compiles**, **fails** `TestCodecKeyRangeRoundTrip` (empty-range round-trip) and `TestCodecClipProperties` (epoch decode out-of-keyspace). `TestCodecContractExamples` and `TestCodecUnmentionedRandom` pass. |

A0’s original white-box fail mixed naming (A1: unexported `thornSlot`) with real
behavior bugs. Against this suite the A0 tree is a **behavioral** miss on
range/epoch clip, not a naming miss: worked examples and unseen random
round-trip/req-resp mostly work; clip and empty-range properties do not.

## Harbor

A0-only path: `experiments/harbor_nex/tasks_unsolv_bb_A0/`
(contains only `spec-reimpl-bb-A0`). Wrapper: `experiments/harbor_nex/run_bb_a0.sh`.

```
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks_unsolv_bb_A0 \
  --agent cursor-cli \
  --model cursor/composer-2.5 \
  --n-concurrent 1 \
  --n-attempts 1 \
  --max-retries 2 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name composer25-bb-A0 \
  --yes
```

`CURSOR_API_KEY` from `/home/evan/Documents/eval_tasks/.env` (not printed).
Log: `experiments/harbor_nex/composer25-bb-A0.log`.
Job dir: `experiments/harbor_nex/jobs/composer25-bb-A0` (wrapper pid 2974117, trial `spec-reimpl-bb-A0__XqubaQK`).
