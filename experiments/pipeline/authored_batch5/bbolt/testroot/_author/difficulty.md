# Difficulty — testroot

predicted_flip: L0
details: 5

Missed edges: exit-code asymmetry (0 on skip, non-zero on missing
privilege); the gate reads the flag value live rather than caching; the
skip path terminates the process rather than returning.

Hardness driver: a single nine-line predicate, but its contract lives at
the process boundary — exit codes and stderr, assertable only through a
subprocess harness. Easy for a solver, easy for a cheat that special-cases
the flag-unset path; the hidden suite's leverage is the flag-set branches.
