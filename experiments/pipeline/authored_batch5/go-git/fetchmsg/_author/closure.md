# Closure — fetchmsg

Package: `plumbing/protocol/packp`. File: `fetch.go`.

Removed (15 functions stubbed): `MalformedResponseError.Error`,
`FetchArgs.Encode`, `FetchArgs.Decode`, `FetchOutput.Decode`,
`fetchSectionRank`, `FetchOutput.decodeMetadataSection`,
`FetchOutput.Encode`, `FetchOutput.decodeAcknowledgments`,
`FetchOutput.decodeShallowInfo`, `FetchOutput.decodeWantedRefs`,
`FetchOutput.decodePackfileURIs`, `FetchOutput.encodeAcknowledgments`,
`FetchOutput.encodeShallowInfo`, `FetchOutput.encodeWantedRefs`,
`FetchOutput.encodePackfileURIs`.

Kept: all type declarations (`FetchArgs`, `FetchOutput`, `Acknowledgments`,
`ShallowInfo`, `WantedRefs`, `PackfileURIs`, `MalformedResponseError`), the
`maxSectionLines` var with its doc comment, the section-shape doc comments,
and the `parseFullHash` helper in `lsrefs.go` (visible — full-length hash
parsing).

Tests deleted: all 23 `*_test.go` in the package (suite is cross-cutting —
fetch/conformance/fuzz exercises the whole v2 grammar).
