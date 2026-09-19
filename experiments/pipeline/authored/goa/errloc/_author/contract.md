# Contract (L2) — errloc

A recorded evaluation error prints as `[file:line] message` when a file is known, otherwise as the underlying message alone. Several such errors print one per line. The file and line attached to an execution error are the first call-stack frame that is not inside this toolkit's own sources and not inside a registered design-package import path; frames whose file name ends in `_test.go` are kept even if the path would otherwise be skipped. Module-cache paths containing `@version` are matched as if that segment were absent. Paths are rewritten relative to the process working directory when possible. A validation error for an expression that still holds its source function uses that function's declaration file and line; an expression with no source function, or a nil source, prints without a location prefix.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRunDSL_ReportErrorLocation` | an error recorded from a source function is tagged with that function's file and the call line |
| `TestRunDSL_ValidationErrorLocation` | a validation failure is tagged with the source function's declaration line |
| `TestValidationErrorsIncludeLocation` | a validation message includes `[file:line]` when a source function exists |
| `TestValidationErrorsWithoutLocation` | the top-level placeholder prints without a location prefix |
| `TestValidationErrorsNilDSL` | a nil source function prints the expression name and message only |
| `TestNormalizeFileForPackageMatch` | `@version` path segments are stripped before package matching |
