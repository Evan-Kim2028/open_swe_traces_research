# Closure — chartloader

Package: internal/chart/v3/loader. Files: load.go, archive.go, directory.go.

Removed: 11 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`Loader(name)` -> ChartLoader; `Load(name)`, `LoadFile(name)`, `LoadArchive(io.Reader)`, `LoadDir(dir)`, `LoadFiles([]*archive.BufferedFile)` -> *chart.Chart; `FileLoader.Load`, `DirLoader.Load`; `LoadValues(io.Reader)` -> map[string]any; `MergeMaps(a,b)` -> map[string]any.
