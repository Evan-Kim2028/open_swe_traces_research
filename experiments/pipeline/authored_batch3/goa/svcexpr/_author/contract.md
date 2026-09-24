# Contract (L2) — svcexpr

Servers, hosts and URIs: schemes, parameters, validation and defaults.

A URI's scheme is decided by string prefix in a fixed order — the secure
HTTP prefix is checked first, then the secure gRPC prefix, then plain gRPC —
and everything else, including a bare or malformed string, reports the plain
HTTP scheme. A URI's parameters are every `{…}` group in order; a wildcard
parameter reports its name keeping the wildcard marker.

Resolving a parameterized URI on a host errors naming the URI when it is not
one of the host's, and substitutes each parameter with its variable's default
value, falling back to the variable's first enum value when there is no
default.

Host validation masks `{…}` placeholders before parsing: an empty URI list
errors, an unparseable URI errors naming it, an empty scheme errors as a
missing scheme, and any scheme outside the four supported ones (HTTP, secure
HTTP, gRPC, secure gRPC) errors as invalid. Every URI variable must be a
primitive AND carry a default value or a non-empty enum — a validation object
with zero enum values does not count — and each violation names the variable.
Both the host's and the server's scheme listings deduplicate and return
sorted names; a host has the HTTP family when any URI is http or https, and
the gRPC family when any is grpc or grpcs. Host finalization, and the
variables accessor, lazily install an empty object attribute when the
variables are unset.

Server finalization defaults an empty service list to every service in the
design, and an empty host list to one usable default host carrying both an
HTTP-family and a gRPC-family URI. Then, per listed service, a host missing
the HTTP family gains a default localhost HTTP URI when that service serves
HTTP or JSON-RPC, and likewise for the gRPC family — evaluated per service,
so the first append satisfies later services.

Server validation merges every host's validation errors and reports each
listed service absent from the design by name. The server's diagnostic name
is a fixed label carrying its name; the host's names both itself and its
owning server.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | scheme decided by prefix — secure HTTP, secure gRPC, gRPC — everything else reports HTTP |
| `TestDetail02` | every `{…}` group extracted in order; a wildcard parameter keeps its marker |
| `TestDetail03` | resolving errors naming the URI when unlisted; parameters substitute their default else first enum value |
| `TestDetail04` | host validation masks placeholders: empty list, malformed URI, missing scheme, invalid scheme each error |
| `TestDetail05` | each URI variable must be primitive and carry a default or a non-empty enum, named on violation |
| `TestDetail06` | host and server scheme listings deduplicate and sort |
| `TestDetail07` | HTTP family = http or https URI; gRPC family = grpc or grpcs |
| `TestDetail08` | host finalize and the variables accessor lazily install an empty object attribute |
| `TestDetail09` | server finalize defaults empty services to all services and empty hosts to one default host with HTTP and gRPC URIs |
| `TestDetail10` | finalize appends a default localhost URI for each scheme family a listed service needs but the host lacks |
| `TestDetail11` | server validation merges host errors and reports each absent listed service by name |
| `TestDetail12` | the server's name is a fixed label plus its name; the host's names itself and its owning server |
