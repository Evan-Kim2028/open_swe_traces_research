# Difficulty — unidiff

predicted_flip: L2
details: 12

Missed edges: `,count` omitted only for 1; mode-suffix on index only when mode unchanged;
metadata-only headers emitting no path lines; ctxPrefix being the TRIMMED line; 2*ctx merge
threshold; zero-context numbering branch; trailing "" line dropped in split; octal modes.

Hardness driver: diff is a familiar format whose exact rules (`@@` counts, merge window,
marker line) are routinely misremembered.
