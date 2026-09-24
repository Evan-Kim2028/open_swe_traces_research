# Exported API — svcexpr

```go
type ServerExpr struct {
    Name, Description string
    Services []string
    Hosts    []*HostExpr
    Meta     MetaExpr
}

type HostExpr struct {
    Name, ServerName, Description string
    URIs      []URIExpr
    Variables *AttributeExpr
    Meta      MetaExpr
}

type URIExpr string

func (s *ServerExpr) EvalName() string
func (s *ServerExpr) Validate() error
func (s *ServerExpr) Finalize()
func (s *ServerExpr) Schemes() []string

func (h *HostExpr) Validate() error
func (h *HostExpr) Finalize()
func (h *HostExpr) EvalName() string
func (h *HostExpr) Attribute() *AttributeExpr
func (h *HostExpr) Schemes() []string
func (h *HostExpr) HasHTTPScheme() bool
func (h *HostExpr) HasGRPCScheme() bool
func (h *HostExpr) URIString(u URIExpr) (string, error)

func (u URIExpr) Params() []string
func (u URIExpr) Scheme() string
```

## Pre-existing callers

`APIExpr.Schemes`/`DefaultServer` walk servers and hosts; the DSL `Server`,
`Host` and `URI` functions build these expressions and the eval engine runs
`Validate`/`Finalize`. codegen reads `URIString`, `Params`, `Schemes` and the
`Has*Scheme` predicates when planning generated servers and clients.
