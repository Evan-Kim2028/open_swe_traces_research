# Difficulty — wsframe

predicted_flip: L2
details: 10

Missed edges: continuation frames write 0 opcode bits; mask key fallback to
math/rand on crypto failure; wsMaskBufs carries key position across
buffers; unmask's 8-byte lane + running mkpos across split reads;
wsGet returning pos+avail (not bytes read from r); close-status validity
table (1004/1005/1006/1015 + reserved 1016–2999); close body truncation to
size-5 + "..."; header pool contract (nbPoolGet 14-byte buf); mpay<=0 →
MAX_PAYLOAD_SIZE then ×8 capped at 64MB.

Hardness driver: RFC-6455 layout is documented, but pool semantics,
streaming mask position, and the close-status validity table must be
reconstructed exactly.
