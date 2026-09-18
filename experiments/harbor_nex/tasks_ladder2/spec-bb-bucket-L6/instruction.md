# Missing behavior

A region covers a half-open key interval: the start is inside, the exclusive
end is not. Optional split keys carve that interval into contiguous buckets.
Looking up a key that belongs to the region must return the unique bucket
whose own half-open interval contains the key. A key outside the region
must not produce a bucket.

When the split list is only the region start and end (no interior splits),
the whole region is one bucket. The version number attached to the split
list is preserved.

Worked cases the hidden tests assert:

- Interval `[a, z)` contains `a` and does not contain `z`.
- One interior split at `m`: `b` is in the left bucket, `m` is in the right.
- No interior splits: `k` is in the bucket whose bounds are the region bounds.
- A key before the region start yields no bucket.
- expected a containing bucket for an in-range key, actual none (lookup
  always returns empty).

Coverage the hidden checks enforce:

- Start inside, exclusive end outside, keys before start outside.
- In-range lookup is non-nil and contains the key; outside is nil; version is kept.
- Left-of-split vs split-key buckets; whole-interval fallback when there are no interior splits.
- Random unseen in-range keys still resolve; hardcoding a few letters fails.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/locate/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestBucketContainsRoundTrip`, `TestLocateBucketProperties`, `TestBucketContractExamples`, `TestBucketUnmentionedRandom`: 10k seeded half-open interval + bucket lookup properties and unseen keys.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
