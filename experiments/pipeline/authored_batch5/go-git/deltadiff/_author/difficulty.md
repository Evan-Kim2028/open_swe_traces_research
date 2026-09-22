# Difficulty — deltadiff

predicted_flip: L3
details: 13

Missed edges: LEB128 size pair ordering; 127-byte insert chunking; copy
opcode flag-byte layout; 64KB copy splitting; sub-block literal fallback;
negative-length small-source sentinel; tail-as-remainder shortcut;
insert-before-copy flush; backward scan with equal-key overwrite; chain
truncation at 64; pow2 table sizing; MemoryObject type/size on GetDelta.

Hardness driver: byte-exact opcode emission over a hash index whose
compression quality is unobservable from I/O — only malformed vs valid vs
optimal distinguish attempts.
