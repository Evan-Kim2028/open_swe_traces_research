# Missing behavior

A bulk erase of a key interval is half-open: the start key is included, the
exclusive end is not. An empty end means unbounded. Keys strictly before the
start must remain. After the erase, a scan of the same interval is empty.

Worked case: six pairs named db/key1/key2/key3/key4/kv are stored, then the
interval from `key3` through unbounded is erased. A scan from `key3` returns
no keys. `key2` is still readable. expected those suffix keys gone, actual
they are still present (a no-op erase).

Coverage the hidden checks enforce:

| property | requirement |
|---|---|
| half-open interval | Keys at or after the start and before the exclusive end disappear; keys strictly before the start stay. |
| worked case | After filling the six named pairs, deleting from key3 through unbounded leaves that suffix empty while key2 remains. |
| unseen key | A key that is not one of the named examples still disappears when its own interval is erased. |

Reproduce with:

```
go test -count=1 -timeout 15m ./rawkv/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
