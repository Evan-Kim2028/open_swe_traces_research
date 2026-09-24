# Difficulty — negotiate

predicted_flip: L3
details: 12

Missed edges: window growth curve differs by transport mode; 256-have
vein budget; three separate termination conditions; wants⊆haves
short-circuit before any request; stateless re-send of accumulated
commons; per-ACK-kind vein resets; detailed-multi-ack preference;
sideband-64k preference; shallow capability gate; fresh-clone format
adoption; absent object-format means sha1-only; shallow-update read
timing.

Hardness driver: a send-everything-once negotiation *works* against a
lenient server — the batching, vein budget and done-round only matter
against real negotiation behavior, which the tests simulate.
