# Difficulty — vfspaths

predicted_flip: L2
details: 8
Commitments: 11-way scheme dispatch, host→bucket convention across seven cloud builders, the S3_ENDPOINT-required-vs-optional split, linode's extra checksum pins, azure's inverted env check + first-slash split, memfs-context gating, and attempt-counted retry semantics with cap-clamped growth.

Hardness driver: each builder is a small variation on a theme — the DIFFERENCES (which need the env var, which extra flags, azure's inverted check) are the commitments; the retry loop's no-initial-sleep/last-error-return semantics diverge from the familiar wait.ExponentialBackoff.
