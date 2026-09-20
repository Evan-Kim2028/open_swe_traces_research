# Contract (L2) — igrole

`ParseInstanceGroupRole` accepts the user-facing role spellings (case-insensitive, plural-tolerant) and yields the typed `InstanceGroupRole` — master/control-plane, node, bastion, and api-server each map to their constant; unknown strings error. `ParseRawYaml` splits a multi-document YAML stream into its documents and unmarshals the one matching the requested kind into the target. `ToRawYaml` re-serializes a document list back to a multi-doc stream. Documents with non-matching kinds are skipped, not errors.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `Test_ParseInstanceGroupRole` | role spellings map to constants; unknown roles error |
| `TestParseConfigYAML` | multi-doc stream selects the requested kind |
