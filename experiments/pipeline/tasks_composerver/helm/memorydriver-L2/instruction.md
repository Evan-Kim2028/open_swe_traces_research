# Contract (L2) — memorydriver

An in-process release store keeps every release revision of every release name in memory, keyed `<name>.v<version>`. Records for one name are always held sorted ascending by version; inserting a version that already exists is an error. Get returns the stored release for an exact key or a not-found error. List returns the newest revision of every release name (optionally filtered by a caller predicate). Query returns the newest revisions whose labels contain all the given key/value pairs. Create stores a new revision under its computed key and fails if it already exists. Update replaces the record at its key, preserving sort position, and fails when absent. Delete removes the record at the key and returns the removed release. All operations are safe under concurrent goroutine access (read lock for reads, write lock for writes).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestMemoryName` | driver reports its name |
| `TestMemoryCreate` | create stores retrievable revision |
| `TestMemoryGet` | get by key or not-found |
| `TestMemoryList` | list yields newest per name |
| `TestMemoryQuery` | label match on newest revisions |
| `TestMemoryUpdate` | update replaces in place |
| `TestMemoryDelete` | delete removes and returns |
| `TestRecordsAdd` | sorted insert + duplicate rejection |
| `TestRecordsRemove` | remove by key |
| `TestRecordsRemoveAt` | remove by index |
| `TestRecordsGet` | record lookup by key |
| `TestRecordsIndex` | index of key in sorted order |
| `TestRecordsExists` | existence check |
| `TestRecordsReplace` | replace keeps position |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/storage/driver/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
