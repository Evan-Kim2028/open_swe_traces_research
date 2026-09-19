# Contract (L2) — chartloader

A chart can be loaded from a directory, a .tgz/tar.gz archive, or an explicit list of buffered files; all three paths funnel into a single file-list loader. Chart.yaml is required and must parse to metadata; values.yaml (if present) is unmarshalled into the chart's value table; files under templates/ become templates, files under charts/ become nested subcharts recursively. A byte-order mark at the start of any UTF-8 file is stripped. Paths are validated: absolute paths, parent-directory escapes, and backslashes are rejected; directories, symlinks, and device files are skipped rather than followed. Loading a directory walks it in lexical order and rejects a chart whose total uncompressed size exceeds a fixed byte budget. File order is preserved for template merging: later files with the same top-level key override earlier ones in MergeMaps. An archive that is truncated or not tar/gzip yields an error, not a partial chart. LoadValues parses a values document stream (YAML or JSON) into a map and tolerates an empty stream at the boundary of the read buffer.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestLoadDir` | directory load produces a complete chart |
| `TestLoadDirExceedsBudget` | oversize charts rejected |
| `TestLoadDirWithDevNull` | device files skipped |
| `TestLoadDirWithSymlink` | symlinks not followed |
| `TestBomTestData` | BOM handling on disk fixtures |
| `TestLoadDirWithUTFBOM` | BOM stripped in dir load |
| `TestLoadArchiveWithUTFBOM` | BOM stripped in archive load |
| `TestLoadFile` | single .tgz file load |
| `TestLoadFiles` | explicit buffered-file list load |
| `TestLoadFilesOrder` | file order preserved |
| `TestLoadFileBackslash` | backslash paths rejected |
| `TestLoadV3WithReqs` | dependencies nested under charts/ |
| `TestLoadInvalidArchive` | truncated archive errors |
| `TestLoadValues` | values stream parsed |
| `TestLoadValuesEOFBoundary` | empty-at-boundary stream tolerated |
| `TestMergeValuesV3` | merge precedence of later files |
