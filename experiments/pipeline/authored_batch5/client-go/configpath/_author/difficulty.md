# Difficulty — configpath

predicted_flip: L1
details: 5

Missed edges: scheme check lowercases (`TiKV://` parses) while the error
text still claims `kvstore`; `disableGC` is case-insensitive but only for
`true`/`false`/empty; empty host splits to `[""]`; the txn-scope failpoint
overrides even a configured scope.

Hardness driver: a thin parser whose behavior is all in the guard details;
the case-fold and the stale error string are the easy misses.
