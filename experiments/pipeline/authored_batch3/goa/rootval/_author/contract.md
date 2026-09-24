# Contract (L2) — rootval

The root expression owns the whole design and drives its evaluation.

Walking happens in a fixed set order: the API first — a nil API is created on
the spot, named after the first service, else the default name `API` — then
servers, user types, result types, services (each bound back to the design
before being walked), methods, then the HTTP transport tree (its root, then
services, endpoints, file servers), then the JSON-RPC transport tree the same
way, then the gRPC root, services and endpoints. Inside each HTTP-family and
gRPC service list a child service is stably sorted after the parent named by
its parent-name setting, regardless of declaration order.

Name lookup resolves declared types first, then result types, then nothing.
Validation requires a non-nil API and then checks: user type names unique by
resolved name; result type names unique across *declared* result types —
generated result types are exempt; every user type and every result type gets
default-value validation scoped so the diagnostic names the offending type;
each declared error is default-validated exactly once across the root,
service and method levels, deduplicated by the error's identity.

Two cross-cutting checks follow. An authored type used under two or more
distinct static error names is rejected — naming the type and the names —
unless some possibly-nested attribute carries the error-name marker; the
collection covers root, service and method errors, groups uses by the type's
origin, and reports in a deterministic sorted order. And when both an HTTP
and a JSON-RPC transport tree exist, their mounted routes are compared per
server — a server with an empty service list, and the synthetic all-services
server assumed when none is declared, hosts every service — and an ordinary
HTTP route and a JSON-RPC route sharing method and path collide, with
parameter names normalized out of the comparison so differently-named
parameters still collide; the error lands on the JSON-RPC route.

Type mappings are deduplicated on the pair (the user type's origin, the
reflected external type): a repeated pair errors with a per-direction message
— a conversion names user type then external type, a creation names external
then user. A relocated user type — one carrying a package-path metadata key —
may depend only on declared types that also carry a package path; the type
itself, undeclared or generated dependencies, and already-located dependencies
are exempt. The diagnostic names the relocated type and its dependency,
reports the dependency path through objects, arrays and maps, and carries a
fix hint.

Finalization defaults a missing API, creates a default server hosting every
service when none is declared, then finalizes the root errors and the
servers — a declared server is preserved.

Metadata merges append only values missing under an existing key and copy the
whole value list for an absent key; asking for a key's last value returns the
last element and an ok flag, false for an absent key or an empty list.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | walk order: API, servers, user types, result types, services, methods, HTTP tree, JSON-RPC tree, gRPC root/services/endpoints |
| `TestDetail02` | a nil API is created from the first service name, else the default name `API` |
| `TestDetail03` | within HTTP and gRPC service lists a child stably sorts after its named parent |
| `TestDetail04` | name lookup resolves declared types first, then result types |
| `TestDetail05` | duplicate names rejected: user types by resolved name; declared result types too, generated ones exempt |
| `TestDetail06` | every user type and result type gets default-value validation scoped to name the type |
| `TestDetail07` | each declared error is default-validated once across root/service/method, deduplicated by identity |
| `TestDetail08` | one authored type under two or more static error names is rejected unless a nested attribute carries the error-name marker |
| `TestDetail09` | shared-route detection only when both HTTP and JSON-RPC trees exist; routes collide on method plus parameter-normalized path; error lands on the JSON-RPC route |
| `TestDetail10` | a server with an empty service list hosts every service |
| `TestDetail11` | type-map dedupe keys on (user type origin, reflected external type) with per-direction messages |
| `TestDetail12` | relocated types may depend only on declared types carrying a package path; the diagnostic reports the dependency path and a fix hint |
| `TestDetail13` | finalize defaults the API, creates a default server when none is declared, then finalizes root errors and servers |
| `TestDetail14` | metadata merge appends only missing values to existing keys, copies whole lists for absent keys; last-value lookup returns last element plus ok |
