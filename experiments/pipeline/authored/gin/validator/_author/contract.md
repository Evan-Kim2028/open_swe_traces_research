# Contract (L2) — validator

The default validator lazily constructs its engine on first use (once-only, safe for concurrent use) and registers 'binding' as the field-tag name. ValidateStruct never panics: nil input returns nil; a pointer is dereferenced — if it points at a non-struct the inner value is re-validated by kind; structs and pointers-to-struct go to the engine; slices and arrays are validated element-wise and per-element errors are collected into a SliceValidationError — nil if every element passed. SliceValidationError.Error concatenates non-nil element errors as '[index]: message' joined by newlines, and an empty slice renders as an empty string. Any other kind (ints, strings, maps, ...) is skipped with nil. Engine() triggers the same lazy init and returns the engine so callers can register custom validations.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestValidateNoValidationValues/TestValidatePrimitives` | non-struct kinds validate to nil without panic |
| `TestValidateNoValidationPointers` | pointers to non-struct values are re-validated by kind, not validated as structs |
| `TestValidateAndModifyStruct` | struct values actually reach the engine |
| `TestValidatorEngine` | Engine() returns the live engine for registering custom checks |
| `TestSliceValidationError/TestDefaultValidator` | per-element errors aggregate with [i]: prefixes; all-pass returns nil |
