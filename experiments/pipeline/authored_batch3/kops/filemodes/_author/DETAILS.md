# Details — filemodes

1. `ParseFileMode` returns the default for an empty string and parses base-8 otherwise; on a parse failure it returns the DEFAULT mode alongside the error. Inferable: partially — octal is conventional, default-on-error is a choice.
2. `FileModeToString` renders `"0"` + octal with no width padding — mode `07` renders `07`, `0644` renders `0644`. Inferable: partially — the leading-zero spelling is a choice.
3. `EnsureFileMode` compares only `stat.Mode() & os.ModePerm` (type bits ignored) via `os.Lstat`, chmods on drift, and reports `changed` honestly. Inferable: partially.
4. `fileHasHash` treats a missing file as (false, nil) — not an error — and a mismatched hash as (false, nil). Inferable: partially.
