# Closure — validcode

`codegen/validation.go` — validation code rendering: per-constraint
templates, nil-guard decisions, alias handling, error paths, format
constants.

Symbols stubbed: `validateAttribute`, `validationAttributeNeedsNilGuard`,
`validationCode`, `literalValidationPath`, `parameterValidationPath`,
`validationPath.child`, `renderValidationPath`, `hasValidations`,
`toSlice`, `oneof`, `constant`, `protobufUnionPayloadRequiresPresence`.

Tests removed: `codegen/validation_test.go`,
`validation_union_context_test.go`, `validation_protobuf_union_test.go`
deleted; plan tests trimmed from `validation_plan_test.go`.
