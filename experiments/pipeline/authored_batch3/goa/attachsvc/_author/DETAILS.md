# Commitments — attachsvc

1. Each service must be a member of the design's service list, the
   design's name lookup must return the SAME service (same-name
   collision rejected), and a service may not be provided twice. In-tree
   coverage: deleted tests only. Inferable: partially.
2. Every method of a selected service must point back at that service.
   In-tree coverage: deleted tests only. Inferable: partially.
3. Each named type must be a member of the design's type list; its
   expression-set entry is the type's ATTRIBUTE, not the type. In-tree
   coverage: deleted tests only. Inferable: no.
4. HTTP collection keeps only transports whose ServiceExpr is selected
   and verifies each transport's Root equals the transport tree being
   collected — ordinary HTTP uses the HTTP root, JSON-RPC uses its own
   embedded HTTP root. In-tree coverage: deleted tests only.
   Inferable: no.
5. Every collected endpoint must point back at its transport and use a
   method declared on the transport's service; file servers likewise.
   In-tree coverage: deleted tests only. Inferable: partially.
6. gRPC collection applies the same endpoint checks but has NO root
   check and no file servers. In-tree coverage: deleted tests only.
   Inferable: no.
7. Set order: types, services, methods, HTTP services, HTTP endpoints,
   HTTP file servers, JSON-RPC services, JSON-RPC endpoints, JSON-RPC
   file servers, gRPC services, gRPC endpoints. In-tree coverage:
   deleted tests only. Inferable: no.
8. Selected services are bound to the owning root before the pipeline
   runs. In-tree coverage: none. Inferable: partially.
9. Pipeline: prepare every preparer in order, then validate — the ROOT
   itself validates first, errors accumulate rather than short-circuit
   — then finalize every finalizer. In-tree coverage: deleted tests
   only. Inferable: partially.
