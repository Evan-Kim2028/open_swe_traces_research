# Closure — resultview

`expr/result_type.go` — result type + view machinery: canonical identifier
parsing, origin tracking, view lookup, finalize/default-view construction,
recursive view projection.

Symbols stubbed: `NewResultTypeExpr`, `IsErrorResult`,
`CanonicalIdentifier`, `(*ResultTypeExpr).{Dup,Origin,ID,Rename,View,
HasMultipleViews,ViewHasAttribute,Finalize,useExplicitView,
ensureDefaultView,projectIdentifier}`, `Project`, `project`,
`projectSingle`, `projectCollection`, `projectedUserType`,
`projectRecursive`, `(*ViewExpr).EvalName`, `hashTypeAndView`.

Tests removed: `expr/project_test.go` and `expr/result_types_root_test.go`
deleted (pin projection identity, attribute non-aliasing, canonical
identifier).
