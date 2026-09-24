# Bug report — consumerstate

The consumer-state and replicated-stream-state binary codecs are missing.
`encodeConsumerState`, `decodeConsumerState`, `checkConsumerHeader`,
`DecodeStreamState`, `IsEncodedStreamState`, the delete-block helpers
(`DeleteRange`, `DeleteSlice`, `DeleteBlocks`), and the run-length/varint
sizing helpers all panic with `excised`. Consumer state cannot be written
or recovered, and NRG stream snapshots cannot be framed or parsed.

Reproduce:

    go test ./server/ -run 'TestFileStoreConsumerEncodeDecode|TestFileStoreBadConsumerState|TestNoRaceEncodeConsumerStateBug|TestFileStoreEncodedStreamStateWithSources'

Restore both directions faithfully: the versioned headers, floor-relative
sequence deltas, the mints-relative second-resolution timestamps (including
the -1 delta edge), v1 compat rules, the corruption guards, and the
deleted-block tail encoding (seqset AND run-length forms).

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
