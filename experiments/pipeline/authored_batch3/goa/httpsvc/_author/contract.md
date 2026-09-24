# Contract (L2) — httpsvc

An HTTP service expression is the transport view of one service: its base
paths, parent linkage, endpoints, file servers and mapped errors.

The canonical endpoint is the one named by the service's canonical-endpoint
setting, defaulting to the `show` endpoint when unset. The service's parent
resolves through the root's service list by the declared parent name. Its
generic name in diagnostics is an unnamed-service placeholder or the quoted
service name.

Full paths are computed per declared base path: with no declared paths the
result is the root base path; a declared path starting with a double slash is
cleaned on its own; every other path is joined onto each full path of the
parent canonical endpoint's first route — cleaned — or onto the root base path
when there is no parent, and a trailing slash on the declared path is
preserved through the join.

Whether the service describes JSON-RPC is decided by a marker metadata key on
the service — not by its routes. Preparation runs JSON-RPC route creation
first for such a service: when no explicit route is configured, every JSON-RPC
endpoint shares one POST route on the service's first base path (`/` when
none). Preparation then pulls any API-level HTTP error mapping matching a
declared service error that the service does not already map — the pulled
mapping is duplicated, not shared — and prepares every mapped error response.

Validation runs in a fixed chain: attributes (parameters, then headers), then
the parent, then the canonical endpoint, then the errors, then transport
compatibility. The parent check rejects a missing parent by name, a parent
without a canonical endpoint, and a parent that is itself a child of this
service. Error validation re-checks the root's HTTP error mappings too, so the
same broken mapping is reported once per service. Transport compatibility
requires every route of every JSON-RPC endpoint to use POST. Finalization
defaults an empty path list to the single root path.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | canonical endpoint is the named one, defaulting to `show` |
| `TestDetail02` | full paths: none → root path; `//` cleaned alone; otherwise joined onto the parent canonical endpoint's first route, trailing slash preserved |
| `TestDetail03` | parent resolves through the root's service list by name |
| `TestDetail04` | JSON-RPC is decided by the service's marker metadata, not by explicit routes |
| `TestDetail05` | prepare pulls matching API-level error mappings the service lacks, duplicated; JSON-RPC routes are created first |
| `TestDetail06` | with no explicit route every JSON-RPC endpoint shares POST on the first base path (`/` when none) |
| `TestDetail07` | missing parent, parent without canonical endpoint, and parent-is-also-a-child each error naming the service |
| `TestDetail08` | validation chain order: attributes (params then headers), parent, canonical endpoint, errors, transports |
| `TestDetail09` | root-level error mappings are re-validated per service |
| `TestDetail10` | every JSON-RPC endpoint route must use POST |
| `TestDetail11` | finalize defaults an empty path list to the root path |
| `TestDetail12` | generic name is an unnamed-service placeholder or the quoted service name |
