# Commitments — rootval

1. Walk order: API, servers, user types, result types, services (each
   first bound to the design), methods, then HTTP services/endpoints/
   file servers, then JSON-RPC, then gRPC root/services/endpoints.
   In-tree coverage: none. Inferable: partially — order is observable.
2. A nil API is created from the first service name, else "API".
   In-tree coverage: none. Inferable: partially.
3. Within HTTP and gRPC service lists, a child service sorts AFTER the
   parent named by its ParentName (stable). In-tree coverage: none.
   Inferable: no.
4. UserType looks in declared types first, then result types. In-tree
   coverage: none. Inferable: yes.
5. Duplicate checks: user types by resolved Name(); result types too
   but ONLY for declared ones — generated result types lack the
   "openapi:typename" meta and are skipped. In-tree coverage: deleted
   tests only. Inferable: no.
6. Every user type and result type gets default-value validation
   scoped "type %q" / "result type %q". In-tree coverage: none.
   Inferable: partially.
7. Each declared error is default-validated exactly once across
   root/service/method levels — deduplicated by pointer identity.
   In-tree coverage: none. Inferable: no.
8. One authored type used under two or more distinct static error
   names is rejected unless some (possibly nested) attribute carries
   the error-name marker meta; collection covers root, service and
   method errors, groups by the type's ORIGIN, and reports in a
   deterministic order with sorted names. In-tree coverage: none.
   Inferable: no.
9. Shared-route detection runs only when both HTTP and JSON-RPC
   transports exist; with no declared servers a synthetic server
   hosting every service is assumed; routes compare by method and by
   path with parameter names normalized — "/{*x}" wildcards normalize
   to "/{*wildcard}" and other params to "/{parameter}". The error
   lands on the JSON-RPC route. In-tree coverage: deleted tests only.
   Inferable: no.
10. A server with an empty service list hosts every service. In-tree
    coverage: none. Inferable: doc — stated in the field docs.
11. Type-map dedupe keys on (user type ORIGIN, reflected external type)
    with a per-direction message — "conversion" names user-then-external,
    "creation" names external-then-user. In-tree coverage: deleted tests
    only. Inferable: partially.
12. Relocated types (struct:pkg:path) may depend only on declared types
    that also carry a package path — self, undeclared/generated types,
    and already-located deps are exempt; the diagnostic reports the
    dependency path (objects "a.b", arrays "[]", maps "[key]"/"[value]")
    and a fix hint. In-tree coverage: deleted tests only. Inferable: no.
13. Finalize defaults the API and creates a default server when none is
    declared, then finalizes root errors and servers. In-tree coverage:
    none. Inferable: partially.
14. MetaExpr.Merge appends only MISSING values onto existing keys and
    copies whole slices for absent keys; Last returns the last element
    plus an ok flag. In-tree coverage: deleted tests only. Inferable:
    partially.
