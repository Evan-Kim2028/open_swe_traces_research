# Difficulty — iamsubj

predicted_flip: L3
details: 7
Commitments: empty/false service-account contract on node roles, echo semantics on the generic type, the four-role factory with asymmetric flag plumbing, cloud-provider gating, and a precise pod-mutation recipe (ARN spelling, volume name, audience, expiry, mount path, env names, FSGroup defaulting).

Hardness driver: the pod mutation is a recipe of a dozen literals — env names, paths, volume name, audience, expiry — every one arbitrary; and the fsGroup-only-when-unset defaulting plus the per-container loop are easy to subtly break.
