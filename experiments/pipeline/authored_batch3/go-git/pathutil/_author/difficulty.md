# Difficulty — pathutil

predicted_flip: L2
details: 12

Missed edges: `~1`–`~4` digit range; `.git` valid under WindowsValidPath; colon early-exit
for ADS; ignored-codepoint skipping between needle bytes; needle must end the component;
three-of-four needle combos; tilde forms and failure passthrough; reserved names not
policed in tree paths.

Hardness driver: platform-canonicalisation predicates where every pattern boundary is a
deliberate security choice documented upstream.
