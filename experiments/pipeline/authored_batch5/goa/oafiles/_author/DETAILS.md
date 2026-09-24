# Commitments — oafiles

1. `Files` returns exactly two codegen files at `gen/<path>.json` and `gen/<path>.yaml`, each rendering the named section over the spec. In-tree coverage: v2/v3 files tests (trimmed). Inferable: doc — Files' comment.
2. `toJSON` emits indented JSON when `openapi:json:prefix` or `openapi:json:indent` meta is set, compact otherwise; marshal failure panics. In-tree coverage: v2 files tests (trimmed). Inferable: partially — the panic-on-bug choice is a contract decision.
3. `toYAML` emits YAML where scalars matching a `YYYY-MM-DD…` date shape are quoted so readers keep them as strings, covering both `key: value` and `- value` line forms. In-tree coverage: `TestToYAMLQuotesDateShapedStrings` (trimmed). Inferable: partially — the date regex shape is a choice.
