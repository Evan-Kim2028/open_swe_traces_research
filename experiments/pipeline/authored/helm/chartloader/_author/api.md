# Exported API — chartloader

`Loader(name)` -> ChartLoader; `Load(name)`, `LoadFile(name)`, `LoadArchive(io.Reader)`, `LoadDir(dir)`, `LoadFiles([]*archive.BufferedFile)` -> *chart.Chart; `FileLoader.Load`, `DirLoader.Load`; `LoadValues(io.Reader)` -> map[string]any; `MergeMaps(a,b)` -> map[string]any.
Callers: pkg/chart/loader (v2 shim), pkg/action install/lint/template paths.
