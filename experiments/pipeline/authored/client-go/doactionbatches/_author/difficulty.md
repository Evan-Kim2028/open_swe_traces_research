# Why hard — doactionbatches

concurrent fan-out with ordering (primary alone first), two independent batch limits, regroup-on-error semantics, and pre-split for new regions. Wrong batching = oversized requests; wrong regroup = lost keys.
