# Closure — renamedet

Package: `plumbing/object`. File: `rename.go`.

Removed (32 functions stubbed): `DetectRenames`; `renameDetector.detect`,
`.detectExactRenames`, `.detectContentRenames`; `bestNameMatch`,
`nameSimilarityScore`, `changeName`, `changeHash`, `changeMode`,
`sameMode`, `groupChangesByHash`, `compactChanges`,
`buildSimilarityMatrix`; `similarityMatrix`/`keyCountPairs` sort impls;
`fileSimilarityIndex`, `newSimilarityIndex`; `similarityIndex.hash`,
`.hashContent`, `.score`, `.common`, `.add`, `.slot`, `.grow`;
`shouldGrowAt`; `newKeyCountPair`, `keyCountPair.key`, `.count`.

Kept: `renameDetector`/`similarityIndex`/`similarityPair`/`keyCountPair`
types, `errIndexFull`, `keyShift`/`maxCountValue`/`maxMatrixSize` consts,
`DiffTreeOptions` consumers in `difftree.go` (intact), every doc comment —
the JGit citations make this a heavily documented port.

Tests deleted: `rename_test.go` (1 — the only suite that reaches it).
