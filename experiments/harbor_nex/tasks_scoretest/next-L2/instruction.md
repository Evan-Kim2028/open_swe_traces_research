# Missing behavior

A forward iterator over the in-memory buffer visits stored keys in
lexicographic order. Each advance moves to the next greater key. After the
last key, the iterator is invalid. The value at the cursor matches the key
that was stored.

Worked case: store `a`, `c`, `b` (out of order). Visiting yields a, then b,
then c. A no-op advance that never moves is wrong: expected b after a,
actual still a (or a hang).

Coverage the hidden checks enforce:

| property | requirement |
|---|---|
| forward order | Seeded random distinct keys are emitted in sorted order, then the cursor is invalid. |
| worked case | Keys stored as a, c, b are visited as a, then b, then c. |
| unseen singleton | A key that is not a/b/c still sits at the cursor and one advance exhausts the iterator. |

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/unionstore/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
