# Commitments — svcerrors

1. Service eval name is "unnamed service" when empty else
   `service "<name>"`; hash is "_service_+<name>". In-tree coverage:
   none. Inferable: partially.
2. Service Validate merges each error's own validation then checks
   inline method errors. In-tree coverage: none. Inferable: partially.
3. A "generated-constructor" error is one whose type is not an authored
   user type OR is the built-in error result type. In-tree coverage:
   deleted tests only. Inferable: no.
4. Inline-error consistency: service errors seed the seen-set first;
   each generated-constructor method error must match the previously
   seen contract — qualifier differences report WHICH settings differ,
   any other difference reports the generic contract message; only
   generated-constructor errors participate. In-tree coverage: deleted
   tests only. Inferable: no.
5. Error-name fields: at most one field per (possibly nested) type may
   carry the error-name marker; it must be String and required.
   In-tree coverage: deleted tests only. Inferable: partially.
6. ErrorExpr.Finalize: an authored non-error-result user type gets the
   error-name marker UNLESS one of its object fields already carries it;
   a non-user-type error wraps its attribute in a user type named after
   the error. In-tree coverage: none. Inferable: no.
7. Method-typed error finalization builds a generated user type keyed by
   method+error example identity, reusing the ORIGIN of the first
   previously-generated same-named error in service method order.
   In-tree coverage: deleted tests only. Inferable: no.
8. The previous-origin search walks methods strictly BEFORE the current
   one and only picks same-named errors whose type is a generated user
   type. In-tree coverage: deleted tests only. Inferable: no.
