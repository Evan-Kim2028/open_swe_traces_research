# Difficulty — provenance

predicted_flip: L3
The crypto plumbing (openpgp entity traversal, clearsign block decode) is unfamiliar enough that a mid model flails at L0-L2, but the contract pins the observable contract precisely: digest-in-sums verification, status-object reporting, armor handling. The message-block format is the main reconstruction risk — test names at L3 plus the contract's format description get it there.

Hardness driver: crypto pipeline with strict wire format.
