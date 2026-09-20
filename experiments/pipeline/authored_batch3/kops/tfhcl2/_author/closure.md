# Closure — tfhcl2

Package: `upup/pkg/fi/cloudup/terraform` (`example.internal/clustkit/upup/pkg/fi/cloudup/terraform`).

Files: `upup/pkg/fi/cloudup/terraform/element.go` (7 funcs), `map.go` (6), `target_hcl2.go` (7) — 20 funcs.

Removed functions (bodies stubbed): `object.IsSingleValue`, `object.Write`, `toElement`, `sliceObject.IsSingleValue`, `sliceObject.Write`, `sliceToElement`, `fieldKey`, `mapStringLiteral.IsSingleValue`, `mapStringLiteral.ToObject`, `mapStringLiteral.Write`, `mapToElement`, `writeIndent`, `quote`, `TerraformTarget.finishHCL2`, `writeLocalsOutputs`, `TerraformTarget.writeProviders`, `sortedKeysForMap`, `TerraformTarget.writeResources`, `TerraformTarget.writeDataSources`, `TerraformTarget.writeTerraform`.

Exported entry point(s): `(*TerraformTarget).Finish` → `finishHCL2` → `kubernetes.tf`; in-package tests reach `writeLocalsOutputs`/`toElement`/`mapToElement` directly.
