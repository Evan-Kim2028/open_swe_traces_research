# Closure — svcerrors

`expr/service.go` — service + error expressions: lookups, eval names,
inline-error contract consistency, error-name field rules, generated error
type construction and origin reuse.

Symbols stubbed: `(*ServiceExpr).{Method,EvalName,Error,Hash,Validate,
validateInlineMethodErrors,Finalize}`, `standardErrorUsesGeneratedConstructor`,
`(*ErrorExpr).{Validate,Finalize,finalizeMethodType}`,
`previousInlineMethodErrorOrigin`.

Tests removed: `expr/service_test.go` deleted (pins inline-error sharing,
qualifier diffs, service/error validate messages).
