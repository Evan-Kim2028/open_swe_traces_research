# Difficulty — sigcodec

predicted_flip: L2
details: 12

Missed edges: last-bracket anchoring; untouched-on-malformed decode;
optional time segment; negative-hours-negate-minutes; unix clamp at 0;
zone-preserving encode; String without time; type dispatch errors;
blob-as-passthrough-object; ErrStop-clean ForEach; error propagation
from Next; type assertion on GetBlob.

Hardness driver: timezone arithmetic and bracket anchoring are
invisible-correct — wrong versions parse most signatures fine and only
break on adversarial inputs.
