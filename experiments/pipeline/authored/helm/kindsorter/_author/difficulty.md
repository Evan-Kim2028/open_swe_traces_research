# Difficulty — kindsorter

predicted_flip: L2
The ordering table is data (InstallOrder/UninstallOrder vars stay in the tree) and the sort contract is fully describable in prose; the only real traps are the unknown-kind and namespace-ordering rule plus hook weight parsing, both stated in the contract. A mid model that reads the retained ordering vars and the contract should converge at L2. Still multi-file and stable-sort-sensitive, so not trivial.

Hardness driver: ordering contract driven by retained data tables.
