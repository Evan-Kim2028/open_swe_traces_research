# Contract (L2) — tfhcl2

`finishHCL2` writes `kubernetes.tf` in a fixed section order: `locals`/`output` blocks, `provider` blocks, `resource` blocks, `data` blocks, then the `terraform {}` requirements block. Structs render via `toElement` — fields snake_cased (or `cty`-tagged), nil pointers and empty slices omitted, `[]*Literal`/`[]string` collapse to list expressions, `[]struct` repeats the block. `object.Write` sorts keys and aligns `=` over runs of consecutive single-value fields. Maps render `"k" = v` sorted and aligned, suppressing empty maps; `quote` escapes only `"` and `\`. Provider naming/body rules vary per cloud (google/hcloud/azurerm aliases, region/project/zone/subscription rules, `features` for azurerm). The requirements block pins versions per provider and errors on an unlisted one.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestWriteLocalsOutputs` | locals block + sorted output blocks render exactly |
