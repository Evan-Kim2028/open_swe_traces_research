# Exported API — compat

```go
func ClientFiles(genpkg string, data *ServicesData) []*codegen.File
func ServerFiles(genpkg string, data *ServicesData) []*codegen.File
func ServerTypeFiles(genpkg string, data *ServicesData) []*codegen.File
func ClientTypeFiles(genpkg string, data *ServicesData) []*codegen.File
func PathFiles(data *ServicesData) []*codegen.File
func ClientEncodeDecodeFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File
func ServerEncodeDecodeFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File
func WebsocketClientFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File
```

## Pre-existing callers

Released plugins and the service-data tests call these to fetch already-planned
files; `requireGeneratedPackage` guards the package identity contract.
