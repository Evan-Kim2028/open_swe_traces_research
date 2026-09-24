# Difficulty — tfliterals

predicted_flip: L3
details: 13
Commitments: twelve small renderers each with an exact spelling (`data.` prefix, `, ` joins, `[a, b]`, `? :` ternary, `${count.index}` suffix), sanitizer application on names, unescaped string quoting, ignored Write parameters, and sort-in-place + adjacent-dedup semantics.

Hardness driver: every function is a one-liner with an exact output spelling — nothing is structurally hard, but there are ~13 independent spellings to recover and several (unescaped quotes, ignored args, sorted side-effect) invite "obvious" wrong choices.
