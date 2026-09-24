# Bug report

MQTT connections are broken. Clients can connect (the CONNECT packet
parses) but then packets desynchronise: subscriptions and publishes that
arrive in two TCP reads instead of one are lost or misparsed — the broker
silently swallows the first fragment and treats the second fragment as a
new packet. Variable-length integer fields (packet "remaining length")
mis-decode for values over 127 and a malformed 5-byte varint loops forever
instead of erroring. Length-prefixed strings/byte fields read past the end
of the buffer or return data that aliases the wrong memory. PUBREL and
SUBSCRIBE packets with the wrong fixed-header flags are accepted, and
PINGREQ packets carrying a payload are accepted too. A packet identifier of
0 in PUBACK/PUBREC/PUBREL/PUBCOMP no longer fails. Oversized packets are no
longer rejected against max_payload — the check now measures only the
unread tail rather than the whole packet. QoS and retain flag extraction
from PUBLISH headers return wrong values.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
