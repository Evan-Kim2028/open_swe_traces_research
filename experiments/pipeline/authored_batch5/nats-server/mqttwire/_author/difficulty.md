# Difficulty — mqttwire

predicted_flip: L2
details: 9

Missed edges: readPacketLen measures the whole packet from pstart (fixed
header + varint + payload) not just the remainder, and skips the pbuf stash
on ErrMaxPayload; pbuf carry-over prepend in reset; varint's complete-flag
(vs error) contract for truncated values and the m > 0x200000 cap; zero
lengths short-circuit without consuming; alias-vs-copy on readBytes;
fixed-header flag table including the "unknown types pass" default; PI=0
rejection.

Hardness driver: a byte-level codec whose surface looks like standard MQTT
but whose split-buffer bookkeeping (pstart/pbuf) and bound semantics are
invented per line.
