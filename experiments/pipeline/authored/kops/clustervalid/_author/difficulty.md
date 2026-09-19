# Difficulty — clustervalid

predicted_flip: L4
Nine functions but a dense state machine: detached vs warm-pool vs bastion join rules, Ready∧¬NetworkUnavailable, max-unready tolerance applied to both node and pod passes, and the static-pod set on control-plane nodes. A mid model given the full contract still drops one of those filters; signatures plus test names (L4) pin which failure kind each case expects.

Hardness driver: multi-pass validation with interacting exemptions (detached/warm-pool/bastion/max-unready).
