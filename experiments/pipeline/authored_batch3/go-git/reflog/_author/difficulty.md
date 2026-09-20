# Difficulty — reflog

predicted_flip: L2
details: 12

Missed edges: partial-slice-plus-error from batch decode; blank-line skipping; optional tab;
last-index signature split; unvalidated tz digit ranges; sub-minute offset loss; no-tab encode
for empty message; 64-byte error quoting.

Hardness driver: forgiving-reader choices (skip/optional/partial) that naive strict parsers get
wrong, plus normalization semantics on encode.
