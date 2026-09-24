# Commitments — grpcresp

1. EvalName is "gRPC response" plus " of <parent>" when a parent is
   set. In-tree coverage: none. Inferable: partially.
2. Prepare installs empty mapped attributes for headers and trailers
   when nil and prepares the message attribute. In-tree coverage: none.
   Inferable: partially.
3. Validate rejects responses where a message attribute name collides
   with a header or trailer mapping, where metadata mapped to headers
   or trailers does not exist in the result type, and where message
   attributes used by both headers and trailers conflict. In-tree
   coverage: deleted tests only. Inferable: no — the collision set and
   per-view lookup are implementation detail.
4. A response message computed from the result removes attributes
   mapped to headers and trailers; when nothing remains the message is
   empty. In-tree coverage: deleted tests only. Inferable: partially.
5. Finalize propagates view/type finalization into the message and
   inherits headers/trailers the endpoint declares at service level.
   In-tree coverage: none. Inferable: no.
6. Dup deep-copies message, headers, trailers, meta, and the Tag pair;
   scalar fields copy by value. In-tree coverage: none. Inferable:
   partially.
