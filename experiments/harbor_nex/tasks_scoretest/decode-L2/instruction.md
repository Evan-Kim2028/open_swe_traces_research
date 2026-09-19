# Missing behavior

Versioned lock and value records must round-trip through their binary
encoding. A lock carries a start timestamp, primary bytes, payload, op,
ttl, for-update timestamp, transaction size, and min-commit timestamp. A
value carries a type, start timestamp, commit timestamp, and payload.
Encoding then decoding must restore every field.

Worked cases:

- lock start 47, primary `abc`, payload `de`, ttl 444, min-commit 666
- value start 42, commit 55, payload `de`

A decoder that leaves fields at zero is wrong: expected start 47, actual 0.

Coverage the hidden checks enforce:

| property | contract |
|---|---|
| seeded lock round-trip | A lock record keeps start timestamp, primary, payload, op, ttl, for-update ts, txn size, and min-commit ts after encode then decode. |
| seeded value round-trip | A value record keeps type, start ts, commit ts, and payload after encode then decode. |
| worked records | lock start 47 / ttl 444 / min-commit 666 with primary abc and payload de; value start 42 / commit 55 with payload de. |
| unseen random | A lock whose start timestamp is not 47 still round-trips. |

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/mockstore/mockkv/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
