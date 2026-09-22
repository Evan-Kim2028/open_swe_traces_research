# Commitments — exid

1. Every identity encodes a distinct KIND byte plus each name component
   with its length so different component lists cannot produce the same
   key — ("ab","c") and ("a","bc") differ. In-tree coverage: deleted
   tests only. Inferable: no — the length-prefixed framing is
   implementation detail.
2. Integer components (array index, map index, status code) encode as
   eight big-endian bytes. In-tree coverage: deleted tests only.
   Inferable: no.
3. Seed returns the whole key as base64 RawURLEncoding text. In-tree
   coverage: deleted tests only. Inferable: partially — "encoded" is
   documented, the codec choice is not.
4. HTTP and JSON-RPC endpoints receive different kinds for request,
   response, and error bodies; each distinct status code gets its own
   key. In-tree coverage: deleted tests only. Inferable: partially —
   the separation is documented, the kind values are not.
5. Generated user types carry their own stored identity (returned by
   GeneratedUserTypeExampleIdentity with ok=true); authored types derive
   one from their ID. In-tree coverage: none directly. Inferable:
   partially.
6. Structural descent (Member/ArrayElement/MapKey/MapValue/UnionMember)
   appends a new framed segment to the existing key and panics on an
   empty seed. In-tree coverage: deleted tests only. Inferable: no.
