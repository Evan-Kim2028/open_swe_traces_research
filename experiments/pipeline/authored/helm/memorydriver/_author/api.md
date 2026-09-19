# Exported API — memorydriver

`NewMemory() *Memory` (driver.Driver). Methods: `Name() string`, `SetNamespace(ns)`, `Get(key)`, `List(filter func(rls) bool)`, `Query(keyvals map[string]string)`, `Create(key, rls)`, `Update(key, rls)`, `Delete(key)`. Record keys are `<name>.v<version>`.
Callers: storage.Storage via driver.Driver interface; tests drive Memory directly.
