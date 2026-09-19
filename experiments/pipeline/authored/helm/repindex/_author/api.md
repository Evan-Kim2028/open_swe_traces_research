# Exported API — repindex

`NewIndexFile()`, `LoadIndexFile(path)`, `IndexDirectory(dir, baseURL)` -> *IndexFile. Methods: `Add(md, filename, baseURL, digest)`, `MustAdd(...)` (validated), `Has(name, version)`, `Get(name, version)` -> *ChartVersion, `SortEntries()`, `Merge(*IndexFile)`, `WriteFile`/`WriteJSONFile`. `ChartVersions` sorts.
Callers: repo commands, downloader, chartrepo.DownloadIndexFile.
