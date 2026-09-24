# Exported API — tagparse

Package `plumbing/object` (module `example.internal/gitkit/v6`) — annotated-tag object codec.

`Tag{Hash, Name, Tagger Signature, Message, Signature, SignatureSHA256, TargetType, Target}`;
`GetTag`, `DecodeTag`, `Tag.Decode(EncodedObject) error`, `Tag.Encode(EncodedObject) error`,
`Tag.EncodeWithoutSignature(EncodedObject) error` (signature-verification payload),
`Tag.Commit/Tree/Blob/Object`, `TagIter`. `ErrMalformedTag` kept.

Callers: tag porcelain, object storage, PGP verification. In-tree tests removed: 1.
