# difficulty — mqttpersist

Medium. Three functions forming one record format. Traps: flags are hex
with an optional leading `-` delete marker; the slicer tolerates spaces
around `:` both directions and capacity-limits values; decode falls back
to legacy JSON when RFlags is absent; topic comes from the subject, not
the header. A cheat that writes plain `key: value` headers and decodes
symmetrically still fails on the golden delete-marker buffer and the
flag-validation path.
