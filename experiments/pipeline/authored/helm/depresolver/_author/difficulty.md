# Difficulty — depresolver

predicted_flip: L4
Four functions but each hides a branch lattice: per-repository-scheme dispatch, alias name preservation, local-path digest-from-disk (version read from the chart, not the constraint), and canonical-sort hashing where order must not matter. TestResolve is one big table of cases — a mid model gets the happy paths and misses alias/file:// semantics; it needs signatures + test names (L4).

Hardness driver: scheme-dispatched resolution + canonical hash.
