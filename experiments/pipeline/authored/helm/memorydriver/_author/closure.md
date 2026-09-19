# Closure — memorydriver

Package: pkg/storage/driver. Files: memory.go, records.go.

Removed: 11 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`NewMemory() *Memory` (driver.Driver). Methods: `Name() string`, `SetNamespace(ns)`, `Get(key)`, `List(filter func(rls) bool)`, `Query(keyvals map[string]string)`, `Create(key, rls)`, `Update(key, rls)`, `Delete(key)`. Record keys are `<name>.v<version>`.
