# Difficulty — conflex

predicted_flip: L2
details: 16

Missed edges: digit-started tokens reclassifying to string; `-`-at-offset-4 date
gate with strict Zulu template; second-dot IP reroute; suffix+`b/i` runs and the
`]`-is-not-a-terminator array quirk; bool six-spelling + `$`-strip order; raw
single-quote vs escaped double-quote; lone-`/` swallowing; newline-separated
array items; `)`-on-bare-line block terminator.

Hardness driver: a Rob-Pike state-machine lexer whose transition table is
visible but whose per-state rune sets and emission rules must be reconstructed.
