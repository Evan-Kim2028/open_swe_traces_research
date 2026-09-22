# Contract (L2) — attachsvc

`RootExpr.EvaluateAttachedServices` re-runs the design pipeline over a chosen
subset of services (and optionally named types) that must already belong to the
receiving design. Membership is strict: every selected service must be the
identical object held in the design's service list — a same-named look-alike is
rejected, as is a service absent from the list or the same service supplied
twice — and every named type must be a member of the design's type list. Every
method of a selected service must point back at that service, and each named
type contributes its attribute expression to the evaluation set.

Transport collection keeps only the transports mounted on the selected
services. For plain HTTP, and for JSON-RPC whose transports live under an
embedded HTTP tree of their own, each collected transport must be rooted in the
transport tree being walked — a transport rooted in a different tree is
rejected. Every collected HTTP endpoint must point back at its transport and
use a method declared on the transport's service; file servers are checked the
same way. gRPC collection applies the same endpoint checks but has no root
check and no file servers.

Before the pipeline runs, each selected service is bound to the receiving
design, so evaluating a service through a design that does not own it fails.
The pipeline then prepares every preparable expression, validates, and
finalizes every finalizable expression. Preparation, validation and
finalization each walk the collected sets in a fixed order: types, services,
methods, HTTP services, HTTP endpoints, HTTP file servers, JSON-RPC services,
JSON-RPC endpoints, JSON-RPC file servers, gRPC services, gRPC endpoints.
Validation checks the design root first, then every set in that order, and
accumulates all violations into one error rather than stopping at the first;
finalization runs only after validation succeeds.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | service membership: the identical listed object is required; foreign, duplicate and absent services are rejected |
| `TestDetail02` | every method of a selected service must point back at that service |
| `TestDetail03` | named types must be members of the design's type list; foreign types rejected |
| `TestDetail04` | collection keeps only transports of selected services and requires each transport's root to be the tree being walked, for plain HTTP and for JSON-RPC's embedded HTTP tree |
| `TestDetail05` | each collected HTTP endpoint points back at its transport and uses a method declared on the transport's service; file servers likewise |
| `TestDetail06` | gRPC collection applies the same endpoint checks with no root check and no file servers |
| `TestDetail07` | sets are processed in a fixed order — types before services before methods before the transport sets — visible in accumulated validation order |
| `TestDetail08` | selected services are bound to the receiving design before the pipeline runs, so a foreign design rejects them |
| `TestDetail09` | the root validates first and violations accumulate rather than short-circuiting; finalize runs last |
