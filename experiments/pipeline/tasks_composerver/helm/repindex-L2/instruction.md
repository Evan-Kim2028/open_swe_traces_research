# Contract (L2) — repindex

A repository index maps chart names to every published version entry, each carrying metadata, download URLs, and a sha256 digest. Adding an entry resolves its download URL against the repository base URL: a filename that is already an absolute URL is used as-is, a bare filename is joined under the base, and an existing entry for the same name/version is replaced, not duplicated. Entries are kept sorted by chart name, then by semantic version descending (newest first); pre-releases sort below their release. Lookup by version accepts an exact version or a semantic-version range (`>`, `>=`, `<`, `<=`, `-` ranges, wildcards, `*`), and a range resolves to the highest matching published version; an unparseable version constraint is an error, and a name or version with no match yields a specific not-found error. Merging a second index unions all entries without creating duplicates. Indexing a directory scans for chart archives and adds each. The serialized index round-trips through YAML and JSON and sorts entries on load; duplicate or empty entries in loaded data are tolerated/deduped as specified.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestIndexFile` | end-to-end index add/get/sort |
| `TestLoadIndex` | yaml index load |
| `TestLoadIndex_Duplicates` | duplicate entries deduped |
| `TestLoadIndex_EmptyEntry` | empty entry tolerated |
| `TestLoadIndex_Empty` | empty index tolerated |
| `TestLoadIndexFileAnnotations` | annotations preserved |
| `TestLoadUnorderedIndex` | entries sorted on load |
| `TestMerge` | indexes merge without dupes |
| `TestIndexDirectory` | directory scanned for archives |
| `TestIndexAdd` | add joins url under base |
| `TestIndexWrite` | yaml write round-trip |
| `TestIndexJSONWrite` | json write round-trip |
| `TestAddFileIndexEntriesNil` | nil-safe add |
| `TestIsVersionRange` | range detection |
| `TestLoadIndex_DuplicateChartDeps` | dup deps tolerated |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/repo/v1/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
