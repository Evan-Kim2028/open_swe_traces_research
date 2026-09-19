# Contract (L2) — kindsorter

Rendered manifests are partitioned and ordered before apply. SortManifests parses each multi-document YAML manifest file, splits hook-annotated documents out of the manifest list, and orders the remaining manifests by resource kind using the given ordering — stable, so equal kinds keep file order; kinds absent from the ordering sort after every known kind, and two unknown kinds order alphabetically by kind name (so a Namespace precedes an unknown kind on install). Hook documents carry a numeric weight annotation (default 0, parsed from their metadata; unparseable weight is treated as 0) and are sorted by weight, then by kind ordering; hook delete-policy annotations are preserved on the hook. Malformed documents contribute errors; empty documents are skipped. Separately, release lists sort by name (version suffix tie-break on the numeric revision), by creation time (oldest first), and by revision number; Reverse applies a sort then inverts it. Manifest files sort into the result keyed by name for deterministic output.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestKindSorter` | kind ordering applied |
| `TestKindSorterKeepOriginalOrder` | stable within kind |
| `TestKindSorterNamespaceAgainstUnknown` | namespace vs unknown kinds |
| `TestKindSorterForHooks` | hooks weight+kind order |
| `TestSortManifests` | partition hooks from manifests |
| `TestSplitManifests` | multi-doc split |
| `TestSortByName` | name sort with revision tiebreak |
| `TestSortByDate` | date sort |
| `TestSortByRevision` | revision sort |
| `TestReverseSortByName` | reverse name |
| `TestReverseSortByDate` | reverse date |
| `TestReverseSortByRevision` | reverse revision |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/release/v1/util/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
