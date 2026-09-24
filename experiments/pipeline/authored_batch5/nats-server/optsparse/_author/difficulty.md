# difficulty — optsparse

Easy-medium individually, fussy in the aggregate: the bare-int-seconds
legacy path (warn not fail), int-vs-string listen asymmetry, dedup
warnings vs error accumulation, K/M/G/T as bit shifts, compression's
string/bool/map trichotomy with four threshold-key synonyms. A cheat that
handles the obvious string forms fails the legacy int path and the
warning-vs-error split.
