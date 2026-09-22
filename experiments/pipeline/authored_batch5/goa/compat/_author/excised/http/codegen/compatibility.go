// This file keeps released HTTP generator entry points available to plugins
// while all rendering uses the one transport plan retained by Apikit.
package codegen

import (
	_ "fmt"

	"example.internal/apikit/v3/codegen"
	"example.internal/apikit/v3/expr"
)

// ClientFiles returns the planned client files. genpkg must match the package
// used to create data.
func ClientFiles(genpkg string, data *ServicesData) []*codegen.File {
	panic("excised: ClientFiles")
}

// ServerFiles returns the planned server files. genpkg must match the package
// used to create data.
func ServerFiles(genpkg string, data *ServicesData) []*codegen.File {
	panic("excised: ServerFiles")
}

// ServerTypeFiles returns the planned server type files. genpkg must match the
// package used to create data.
func ServerTypeFiles(genpkg string, data *ServicesData) []*codegen.File {
	panic("excised: ServerTypeFiles")
}

// ClientTypeFiles returns the planned client type files. genpkg must match the
// package used to create data.
func ClientTypeFiles(genpkg string, data *ServicesData) []*codegen.File {
	panic("excised: ClientTypeFiles")
}

// PathFiles returns the planned request path files.
func PathFiles(data *ServicesData) []*codegen.File {
	panic("excised: PathFiles")
}

// ClientEncodeDecodeFile returns the planned client encoder and decoder file
// for service. genpkg must match the package used to create data.
func ClientEncodeDecodeFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File {
	panic("excised: ClientEncodeDecodeFile")
}

// ServerEncodeDecodeFile returns the planned server encoder and decoder file
// for service. genpkg must match the package used to create data.
func ServerEncodeDecodeFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File {
	panic("excised: ServerEncodeDecodeFile")
}

// WebsocketClientFile returns the planned WebSocket client file for service.
// genpkg must match the package used to create data.
func WebsocketClientFile(genpkg string, service *expr.HTTPServiceExpr, data *ServicesData) *codegen.File {
	panic("excised: WebsocketClientFile")
}

// requireGeneratedPackage rejects a package argument that does not describe
// the HTTP data supplied by the same generation run.
func requireGeneratedPackage(genpkg string, data *ServicesData) {
	panic("excised: requireGeneratedPackage")
}
