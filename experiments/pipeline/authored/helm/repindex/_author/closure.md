# Closure — repindex

Package: pkg/repo/v1. Files: index.go.

Removed: 10 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`NewIndexFile()`, `LoadIndexFile(path)`, `IndexDirectory(dir, baseURL)` -> *IndexFile. Methods: `Add(md, filename, baseURL, digest)`, `MustAdd(...)` (validated), `Has(name, version)`, `Get(name, version)` -> *ChartVersion, `SortEntries()`, `Merge(*IndexFile)`, `WriteFile`/`WriteJSONFile`. `ChartVersions` sorts.
