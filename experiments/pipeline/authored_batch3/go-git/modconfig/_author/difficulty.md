# Difficulty — modconfig

predicted_flip: L2
details: 12

Missed edges: silent-skip only for name/path errors; name fallback to path on marshal;
remove-on-empty branch options; `false` rebase accepted; bidirectional newline escaping;
empty-string → unset divergence; validation ordering; drive-letter and NUL checks beyond
canonical git.

Hardness driver: several "extra" rules layered over the upstream baseline that only the
doc comments hint at.
