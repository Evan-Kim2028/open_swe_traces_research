# Details — tfwriter

1. Name legalization: `.` → `-`, `/` → `--` (double dash), `:` → `_`, and a leading digit prefixes the whole name with `prefix_`. Inferable: no — the exact replacement map is arbitrary.
2. Legalized-name collisions are detected at query time, not insert time — two raw names that legalize identically produce a "duplicate" error from the Get*ByType calls. Inferable: partially — detection point is a choice.
3. `GetResourcesByType`/`GetDataSourcesByType` group items by type then by legalized name; same-name different-type entries do NOT collide. Inferable: partially.
4. `AddOutputVariable` on an existing key is a "duplicate variable" error; `AddOutputVariableArray` on a key holding a scalar is a "both an array and a scalar" error — two different messages for asymmetric orderings. Inferable: no.
5. `AddOutputVariableArray` on an absent key creates the entry and appends; repeat calls accumulate. Inferable: yes.
6. `GetOutputs` legalizes output keys too — two keys legalizing to the same name is an error — and dedups+sorts `ValueArray` while leaving scalar `Value` untouched. Inferable: partially.
7. `EnsureTerraformProvider` returns the existing provider when arguments match exactly, and aborts the process (klog.Fatalf) when they differ — not an error return. Inferable: no.
8. `InitTerraformWriter` initializes only the file and output maps; resource/data-source/provider collections lazy-init on first use. Inferable: partially.
9. `AddFilePath` stages bytes under `data/<type>_<name>_<key>` and returns a literal for the quoted `${path.module}/data/<id>` path; `AddFileBytes` wraps it in `file()`/`filebase64()` by flag. Inferable: partially.
10. `RenderResource`/`RenderDataSource` always succeed and preserve insertion order. Inferable: yes.
