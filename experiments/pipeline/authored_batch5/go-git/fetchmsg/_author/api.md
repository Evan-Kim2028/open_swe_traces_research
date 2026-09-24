# Exported API — fetchmsg

Package `plumbing/protocol/packp` (module `example.internal/gitkit/v6`) — v2
fetch command request/response codec.

`FetchArgs{Wants,Haves,Done,ThinPack,NoProgress,IncludeTag,OFSDelta,Shallows,
Deepen,DeepenRelative,DeepenSince,DeepenNot,Filter,WaitForDone}` with
`Encode`/`Decode`; `FetchOutput{Acknowledgments,ShallowInfo,WantedRefs,
PackfileURIs,Packfile}` with `Encode`/`Decode`; section types
`Acknowledgments{ACKs,Ready}`, `ShallowInfo{Shallows,Unshallows}`,
`WantedRefs{Refs}`, `PackfileURIs{URIs}`; `MalformedResponseError{Reason}`.

Callers: transport fetch/negotiate paths build `FetchArgs`, consume
`FetchOutput`. In-tree tests removed: 23.
