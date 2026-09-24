# Bug report — raftcodec

The NRG wire codec is missing. Encoding or decoding any raft appendEntry,
appendEntryResponse, voteRequest, peerState, snapshot frame, or snapshot
file name panics (`panic("excised: ...")`). Nodes cannot serialize or
parse cluster-protocol messages; WAL peer-state recovery and snapshot
install are broken.

Reproduce:

    go test ./server/ -run 'TestNRGAppendEntry|TestNRGPeerState|TestNRGMalformed|TestNRGVote'

Restore the codec to full fidelity: exact byte layouts, bounds-checked
decoding that returns the declared error values instead of panicking,
borrowed-vs-copied slice semantics, the compat `lterm` tail, the peer
interning, the snapshot checksum, and strict snapshot-name parsing.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
