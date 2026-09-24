# Commitments — svcexpr

1. A URI's scheme is decided by string prefix: "https" is checked first,
   then "grpcs", then "grpc", and EVERYTHING else — including a bare or
   malformed string — reports "http". In-tree coverage: none. Inferable:
   no — the fallback hides malformed input instead of failing.
2. URI parameters are extracted as every `{...}` group; a wildcard
   `{*name}` parameter reports its name INCLUDING the leading `*`, while
   the validator's variable-masking regex strips it. In-tree coverage:
   none. Inferable: no — the two paths disagree on the wildcard.
3. Resolving a parameterized URI errors with "uri %s not found in host"
   when the URI is not one of the host's, and substitutes each parameter
   with the variable's default value, falling back to the FIRST enum
   value when there is no default. In-tree coverage: none. Inferable: no.
4. Host validation masks `{var}` placeholders before parsing: an empty
   URI list is "host must define at least one URI", a parse failure is
   "malformed URI %q", an empty scheme is "missing scheme for URI %q,
   scheme must be one of 'http', 'https', 'grpc' or 'grpcs'", and any
   other scheme is "invalid scheme for URI %q, ...". In-tree coverage:
   none. Inferable: partially — the message shapes are visible, the
   masking trick is not.
5. Every URI variable must be a primitive AND have a default value or a
   non-empty enum (a validation object with zero enum values does not
   count). In-tree coverage: none. Inferable: no.
6. Both scheme aggregations deduplicate and return sorted names.
   In-tree coverage: none. Inferable: partially — sorted output is the
   natural guess, dedup is the detail.
7. HasHTTPScheme is true when any URI is http or https; HasGRPCScheme for
   grpc or grpcs. In-tree coverage: none. Inferable: doc.
8. Host finalize and the variables accessor both lazily install an empty
   object attribute when variables are nil. In-tree coverage: none.
   Inferable: no.
9. Server finalize: empty Services defaults to every service in the
   design; empty Hosts defaults to one host "svc" described as "Service
   host" carrying URIs "http://localhost:80" and "grpc://localhost:8080".
   In-tree coverage: none. Inferable: no — the literal URIs are
   arbitrary.
10. Server finalize appends "http://localhost:80" to any host missing an
    http scheme when a listed service serves HTTP or JSON-RPC, and
    "grpc://localhost:8080" when it serves gRPC — evaluated per service,
    so the first append satisfies later services. In-tree coverage:
    none. Inferable: no.
11. Server validation merges every host's validation errors AND reports
    "service %q undefined" for each listed service absent from the
    design. In-tree coverage: none. Inferable: partially.
12. Error names: server is "Server "+name; host is `host "name" of
    server "server"`. In-tree coverage: none. Inferable: partially.
