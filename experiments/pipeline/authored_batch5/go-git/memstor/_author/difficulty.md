# Difficulty — memstor

predicted_flip: L2
details: 12

Missed edges: dual-map bookkeeping; store-on-unknown-type error split;
type-filtered lookup; deferred storage on writer close; tx read
isolation; CAS semantics on CheckAndSetReference; SetIndex ModTime
stamping; lazy default config/index; ErrStop-as-EOF; module memoization;
format-change guard on populated storage; not-supported stubs.

Hardness driver: individual methods are each simple — the surface is
wide rather than deep, and the cheat version "works" until CAS, type
maps and tx isolation are exercised.
