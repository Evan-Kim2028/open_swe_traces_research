# Contract (L2) — kopscodecs

Encode/decode between runtime objects and versioned wire forms. `ToVersionedYaml`/`ToVersionedJSON` emit the object in its registered external version with `apiVersion`/`kind` set; the `WithVersion` variants let the caller pick the target version. `Decode` reads a serialized object back into the internal type using the scheme. `rewriteAPIGroup` rewrites the group in serialized output — used to present legacy objects under the current group. Objects of an unregistered kind fail loudly; a missing version falls back to the scheme's preferred version.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestToVersionedYaml` | object serializes to YAML with apiVersion/kind |
| `TestToVersionedJSON` | object serializes to JSON equivalently |
| `TestRewriteAPIGroup` | group string is rewritten in the emitted document |
