# DETAILS — gethdr

1. Header lookup is case-insensitive: any letter-casing of the queried key
   matches the stored key. Inferable: yes — HTTP headers are
   case-insensitive by spec and both public methods promise it.
2. A key absent under every casing returns the empty string. Inferable:
   yes — map-lookup contract.
3. The stored value is returned verbatim (no trimming or rewriting).
   Inferable: partially — returning the value is forced, the no-rewrite
   part is conventional.
