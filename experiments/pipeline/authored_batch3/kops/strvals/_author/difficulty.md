# Difficulty — strvals

predicted_flip: L2
details: 13
Commitments: full strvals grammar (dotted nesting, `[i]` indexing, `{a,b}` lists, backslash escapes), typed scalar coercion, empty-vs-missing value at EOF, MaxIndex/MaxNestedNameLevel bounds, recover-to-error wrapping, reuse of pre-existing dest entries.

Hardness driver: one recursive-descent core drives four surface behaviors (nested maps, list growth, brace lists, coercion); empty-value-at-EOF vs missing-value is a quiet distinction; wrong-shaped pre-existing entries take an undocumented panic-recovery path.
