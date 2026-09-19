# Exported API — ignorerules

`Empty() *Rules`; `(*Rules).AddDefaults()`; `ParseFile(file)`/`Parse(io.Reader)` -> *Rules; `(*Rules).Ignore(path, os.FileInfo) bool`.
Callers: chart loaders/packaging when walking a chart directory.
