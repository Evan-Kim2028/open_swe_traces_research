# Commitments — oamarshal

1. Extension map entries are merged into the encoded object as top-level keys alongside the struct fields. In-tree coverage: `TestExtensions` (trimmed). Inferable: doc — the method comments state merge semantics.
2. Numeric values survive round-tripping without float corruption — decoding uses `json.Number` semantics so large integers keep their exact digits. In-tree coverage: `TestMarshalJSONPreservesLargeIntegers` (trimmed). Inferable: partially — UseNumber is an implementation choice pinned by the test.
3. Marshal and merge errors propagate to the caller rather than being swallowed. In-tree coverage: `TestExtensions`, `TestValidations` (trimmed). Inferable: yes.
