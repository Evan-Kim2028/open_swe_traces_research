# Commitments — oatags

1. Both `swagger:tag:` and `openapi:tag:` prefixes are recognized; output order follows sorted meta keys and tags dedupe by name. In-tree coverage: `TestTagsFromExpr` (trimmed). Inferable: doc — parseTags' comment commits to sorted stable output.
2. `<tag>:desc` sets Description; `<tag>:url` and `<tag>:url:desc` populate ExternalDocs. In-tree coverage: `TestTagsFromExpr` (trimmed). Inferable: partially — key spellings are design choices.
3. `summary`, `parent`, and `kind` are emitted only for the 3.2 spec; earlier versions must not carry them. In-tree coverage: `TestTagsFromExpr` (trimmed). Inferable: doc — TagsFromExpr's comment.
4. `extension:` sub-keys become `x-*` extensions merged into output via the marshal methods. In-tree coverage: `TestExtensions` (trimmed). Inferable: partially.
