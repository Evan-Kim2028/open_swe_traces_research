# Difficulty — protoparse

predicted_flip: L2
details: 16

Missed edges: split-buffer arg/payload stashing via scratch + clonePubArg
re-parse; the `as + pa.size - LEN_CR_LF` index jump (size excludes CRLF,
msgBuf includes it); `\r` drop=1 semantics incl. dangling-`\r` at buffer end;
mcl×16 for non-client kinds with early argBuf checks; PING/PONG/+OK absorbing
junk bytes until `\n`; CONNECT/INFO optional space vs `-ERR` required space;
per-kind op gates (R/A vs CLIENT, L vs LEAF/ROUTER) and the authSet gate with
NoAuthUser ordering (clear timer + connectReceived BEFORE checkAuth);
conditional vs unconditional payload jump (PUB/HPUB vs MSG/HMSG);
protoSnippet's len-1 clamp; getHeader's skip-first-line lazy cache.

Hardness driver: a 700-line byte-machine whose state constants are all
visible but whose buffer-split bookkeeping, auth ordering, and boundary
arithmetic must be reconstructed exactly.
