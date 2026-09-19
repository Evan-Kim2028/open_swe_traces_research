# Missing behavior

A batch erase takes a list of keys and removes each of them. Keys that were
not listed must remain. After the erase, a lookup of a listed key is empty.

Worked case: store db, key1, key2, key3 then erase that list. A lookup of the
first listed key is empty. expected listed keys gone, actual they are still
present (a no-op batch erase).

Coverage:

| situation | required result |
|---|---|
| every listed key after a batch erase | lookup is empty |
| a companion key that was not listed | still readable |
| worked names db, key1, key2, key3 then erase that list | first listed key missing |
| an unseen random key as the sole listed key | vanishes |

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
