# Difficulty — distros

predicted_flip: L3
details: 8
Commitments: packageFormat-vs-project predicate split, per-project version gates for DNF with an unknown-project default, a constant systemd predicate, a per-project user table, a host-filesystem probe predicate, and an inverted nftables allowlist.

Hardness driver: pure fact-tables with plausible-wrong defaults (unknown rpm→DNF true; ForceNftables allowlist vs denylist; ubuntu-not-debian); the host-probe predicate is environment-dependent and punishes tests that assume determinism.
