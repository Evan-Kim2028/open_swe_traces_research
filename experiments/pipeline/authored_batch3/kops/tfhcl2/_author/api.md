# Exported API — tfhcl2

Package `upup/pkg/fi/cloudup/terraform` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/terraform`).

- `(*TerraformTarget).Finish()` → writes `kubernetes.tf` — the whole HCL document: locals+outputs, provider blocks, resources, data sources, terraform requirements block.
- In-package surface: `toElement(item)` reflection-driven value→element conversion; `mapToElement(map)`; `writeLocalsOutputs(buf, outputs)`; `writeResources`/`writeDataSources`/`writeProviders`/`writeTerraform` section emitters; `object`/`mapStringLiteral`/`sliceObject` element types with `Write(buf, indent, key)`.

Production callers: `upup/pkg/fi/cloudup/terraform/target.go` (`Finish`), task code feeding `RenderResource`/`RenderDataSource`/`AddOutputVariable` upstream.
