# Closure — subjvalid

File: `server/sublist.go` (subject-syntax layer only — the Sublist
trie/cache machinery is untouched; the banked `gsl` closure covers
`server/gsl`, a different package).

Stubbed (13 symbols): `subjectIsLiteral`, `isValidSubject`,
`IsValidSubject`, `IsValidPublishSubject`, `IsValidLiteralSubject`,
`isValidLiteralSubject`, `ValidateMapping`, `SubjectsCollide`,
`analyzeTokens`, `tokensCanMatch`, `isSubsetMatchTokenized`, `tokenAt`, `numTokens`.

Import blanked after excision: `unicode/utf8`.

Test coverage snipped (restored for gold/cheat), all in
sublist_test.go: `TestSublistValidLiteralSubjects`,
`TestSubjectIsLiteral`, `TestValidateDestinationSubject`,
`TestSubjectToken`, `TestSublistSubjectCollide`.
