package expr

type (
	// JSONRPCExpr contains the API level JSON-RPC specific expressions.
	JSONRPCExpr struct {
		HTTPExpr
	}
)

// EvalName returns the name printed in case of evaluation error.
func (*JSONRPCExpr) EvalName() string {
	panic("excised: JSONRPCExpr.EvalName")
}

// Prepare copies shared HTTP request settings to the JSON-RPC API. JSON-RPC
// error mappings stay separate because their codes belong to JSON-RPC, not
// HTTP.
func (j *JSONRPCExpr) Prepare() {
	panic("excised: JSONRPCExpr.Prepare")
}
