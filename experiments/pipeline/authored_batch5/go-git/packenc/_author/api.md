# Exported API — packenc

Package `plumbing/format/packfile` (module `example.internal/gitkit/v6`) —
PACK file writer: header/entry/footer emission, delta headers, and the
`ObjectToPack` write-side bookkeeping record.

`NewEncoder(w, storer, useRefDeltas, ...EncoderOption)`, `Encoder.Encode(hashes,
packWindow) Hash`, `WithObjectSelector`, `ObjectSelector` iface; `ObjectToPack`
methods `IsWritten/MarkWantWrite/WantWrite/BackToOriginal/SetOriginal/
SaveOriginalMetadata/CleanOriginal/Type/Hash/Size/IsDelta/SetDelta`.

Callers: push/fetch pack assembly via transport. In-tree tests removed: 19.
