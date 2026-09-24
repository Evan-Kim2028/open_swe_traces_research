# Difficulty — bytesfmt

predicted_flip: L2
details: 7

Missed edges: `<=`/`>` boundary choices mean 1024 bytes prints as
"1024 Bytes" while 1025 prints "1.00 KB"; decimal-count rule (0 / 2 / 1)
depends on exact divisibility and magnitude; the GC-time fallback drops the
last space-field only once.

Hardness driver: three subtly different formatters (FormatBytes prunes,
BytesToString doesn't) that agree on some inputs and diverge on others.
