# Commitments — methodval

1. Validation merges payload, streaming_payload, result, then
   streaming_result ONLY when it is a different object than result —
   then requirements, errors, interceptors. In-tree coverage: none.
   Inferable: partially.
2. A payload field marked as a security attribute must be String or a
   named String type, must not set a field-type override, and must not
   declare a default value (missing credentials stay missing). In-tree
   coverage: deleted tests only. Inferable: partially — the reasons are
   stated in the messages.
3. The effective security requirements are the method's own, else the
   service's, else the API's. In-tree coverage: none. Inferable: yes.
4. Each scheme kind demands its own payload tag: basic auth needs
   username AND password; API key needs the tag namespaced by that
   scheme's name; bearer, JWT and OAuth2 each need their own tag.
   In-tree coverage: deleted tests only. Inferable: partially.
5. Every scope listed in a requirement must exist in at least one of
   that requirement's schemes — searched only in schemes of the five
   credential kinds. In-tree coverage: none. Inferable: no.
6. Inverse check: a payload carrying a credential tag with NO matching
   scheme kind errors (one message per kind; the API-key check is a
   prefix match). In-tree coverage: none. Inferable: no.
7. "Security attribute" = the five fixed credential tags plus any tag
   with the API-key prefix. In-tree coverage: none. Inferable: no.
8. A named type counts as String only after unwrapping user types to
   their underlying attribute. In-tree coverage: none. Inferable: no.
9. Method+service+API interceptors are merged into the method lists in
   precedence order, deduplicated by name — method wins. In-tree
   coverage: none. Inferable: partially.
10. Credential tags are found by walking the payload's own meta, then
    recursively through its base user types, then its type. In-tree
    coverage: none. Inferable: no.
11. Finalize defaults nil payload/streaming-payload/result to empty
    attributes, finalizes result-type results, inherits service errors
    whose names are not already declared, finalizes errors (authored
    user types get plain finalize, others get the method-typed variant),
    and inherits requirements from service then API — except that an
    explicit no-security declaration produces a single no-kind scheme.
    In-tree coverage: none. Inferable: no.
12. Inherited requirements are duplicated, not shared. In-tree coverage:
    none. Inferable: no.
13. Streaming predicates: payload streams on client and bidirectional
    kinds, result streams on server and bidirectional kinds; mixed
    results requires both result objects present AND different objects.
    In-tree coverage: none. Inferable: partially — names say which,
    bidirectional inclusion is the detail.
