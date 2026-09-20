# Contract (L2) — filemodes

`ParseFileMode` treats "" as the default and parses octal; on parse failure it returns the default alongside the error. `FileModeToString` renders `"0"`+octal. `EnsureFileMode` compares only `ModePerm` bits (Lstat), chmods on drift, and reports `changed` truthfully. `fileHasHash` treats a missing file as (false,nil) — absent means no-match, not an error — and a mismatched hash as (false,nil).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestWriteFile` | WriteFile→EnsureFileMode path: file lands with requested mode |
