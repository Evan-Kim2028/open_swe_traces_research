# Contract — featflags

Process-global feature flags parsed from the `KOPS_FEATURE_FLAGS` env var.
Every commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **`Enabled` precedence.** An explicit value set via `ParseFlags` beats the
   registered `defaultValue`; when neither is set the flag is disabled.
   Covered by `TestDetail01`.
2. **Registration idempotence.** `new` returns the same `*FeatureFlag` for a
   repeated key; the first non-nil default wins — a later default does not
   overwrite it, and a later non-nil default fills a previously nil one.
   Covered by `TestDetail02`.
3. **Parse grammar.** `ParseFlags` accepts `+Name`, `-Name`, and bare `Name`
   (enable); whitespace around items is trimmed and empty items are skipped.
   Covered by `TestDetail03`.
4. **Unknown names tolerated.** Unknown flag names are logged and skipped —
   parsing them neither errors nor disturbs known flags. Covered by
   `TestDetail04`.
5. **Order matters.** Within one `ParseFlags` call, later items override
   earlier ones for the same flag. Covered by `TestDetail05`.
6. **Init-once env.** The env var is parsed once at package load; mutating it
   afterwards has no effect until `ParseFlags` is invoked explicitly with the
   new value. Covered by `TestDetail06`.
7. **`Get` lookup (shape).** `Get` returns the flag for a registered name;
   for an unknown name it returns a non-nil error that names the missing
   flag. The exact message is an implementation detail. Covered by
   `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — asserts first-non-nil-wins per api.md |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | yes |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | no — shape only (non-nil error naming the flag) |
