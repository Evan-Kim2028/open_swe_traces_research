# Commitments — httpsvc

1. The canonical endpoint is the named one, defaulting to "show" when
   unset. In-tree coverage: none. Inferable: doc — the canonical-name
   convention is documented.
2. FullPaths: no declared paths yields the root path; a path starting
   "//" is cleaned on its own; otherwise each path joins the parent
   canonical endpoint's first route full paths (joined, cleaned) — or
   the root path when there is no parent — and a trailing slash in the
   declared path is preserved. In-tree coverage: deleted tests only.
   Inferable: no.
3. Parent resolves through the root's service list by ParentName.
   In-tree coverage: none. Inferable: yes.
4. JSON-RPC is decided by the service's "jsonrpc:service" meta — not by
   an explicit route. In-tree coverage: none. Inferable: no.
5. Prepare creates JSON-RPC routes first when the service is JSON-RPC,
   then pulls any API-level HTTP error matching a declared service error
   that the service does not already map — duplicated, not shared — and
   prepares every mapped error response. In-tree coverage: none.
   Inferable: no.
6. JSON-RPC route defaults: when no explicit route exists every
   JSON-RPC endpoint shares POST on the service's first path ("/" when
   none). In-tree coverage: none. Inferable: no.
7. Parent validation: missing parent errors; a parent without a
   canonical endpoint errors; a parent that is also a child errors.
   In-tree coverage: deleted tests only. Inferable: partially.
8. Validation chain order: attributes (params then headers), parent,
   canonical endpoint, errors, transports. In-tree coverage: none.
   Inferable: partially.
9. Error validation re-checks root HTTP errors too — the same error may
   be validated repeatedly across services. In-tree coverage: none.
   Inferable: doc — the comment explains the constraint.
10. JSON-RPC route validation requires POST on every route of every
    JSON-RPC endpoint. In-tree coverage: none. Inferable: partially.
11. Finalize defaults an empty path list to ["/"]. In-tree coverage:
    none. Inferable: yes.
12. EvalName is "unnamed service" or `service "<name>"`. In-tree
    coverage: none. Inferable: partially.
