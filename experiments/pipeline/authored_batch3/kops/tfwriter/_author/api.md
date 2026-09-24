# Exported API — tfwriter

Package `upup/pkg/fi/cloudup/terraformWriter` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/terraformWriter`).

- `func (t *TerraformWriter) InitTerraformWriter()` — initialize the writer's maps.
- `func (t *TerraformWriter) RenderResource(type, name string, e interface{}) error` / `RenderDataSource` — record an object to render.
- `func (t *TerraformWriter) AddOutputVariable(key string, l *Literal) error` / `AddOutputVariableArray` — record output values.
- `func (t *TerraformWriter) EnsureTerraformProvider(name string, args map[string]string) *TerraformProvider`.
- `func (t *TerraformWriter) AddFilePath` / `AddFileBytes` — stage file content under `data/` and return a reference literal.
- `func (t *TerraformWriter) GetResourcesByType()` / `GetDataSourcesByType()` / `GetOutputs()` — grouped, sanitized views.
- `var Files`, `Providers`, `AzureStorageAccountID` fields; `OutputValue{Value, ValueArray}`.

Production callers: `upup/pkg/fi/cloudup/terraform/target_hcl2.go` (via `finishHCL2`), `util/pkg/vfs/*_terraform.go`, every `*tasks` package.
