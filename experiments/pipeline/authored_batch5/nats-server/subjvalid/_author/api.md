# API — subjvalid

Module: `example.internal/msgkit/v2`, package `server`, file
`server/sublist.go`.

Excised symbols — the subject-syntax predicate/tokenizer layer:

- `subjectIsLiteral(subject string) bool`
- `isValidSubject(subject string, checkRunes bool) bool` and exported
  wrappers `IsValidSubject`, `IsValidPublishSubject`,
  `IsValidLiteralSubject`, plus `isValidLiteralSubject(tokens
  iter.Seq[string])`.
- `ValidateMapping(src, dest string) error` — destination-subject
  validation incl. `{{...}}` mapping-function tokens.
- `SubjectsCollide(subj1, subj2 string) bool` with helpers
  `analyzeTokens`, `tokensCanMatch`, `isSubsetMatchTokenized`.
- Tokenizers: `tokenAt(subject, index uint8)` (1-based),
  `numTokens(subject)`.

Callers: subject validation across client SUB/PUB handling, stream
config validation, subject-mapping setup, and interest-collision
checks (SubjectsCollide is used to detect overlapping wildcard
interest). `tokenAt` backs `streamNameFromSubject`/
`consumerNameFromSubject` in jetstream_api.go.

Retained as scaffolding: the `Sublist` match trie itself (distinct
from the banked `gsl` package), `mappingDestinationErr`, the
mapping-function regexes, `NewSubjectTransform`.
