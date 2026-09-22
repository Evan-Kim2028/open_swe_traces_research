# Contract — subjvalid

Subject-syntax predicates and tokenizers. Every commitment below is
covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Subject validity.** Empty subjects and empty tokens are invalid;
   whitespace inside a token is invalid; `>` is valid only as a whole
   final token; wildcard characters embedded in longer tokens are
   literal; the rune check additionally rejects NUL and invalid UTF-8.
   Covered by `TestDetail01`.
2. **Literal detection.** A subject is non-literal only when a wildcard
   character forms a complete dot-bounded token. Covered by
   `TestDetail02`.
3. **Literal validity.** Literal validation rejects empty tokens and
   single-character wildcard tokens but does not check whitespace.
   Covered by `TestDetail03`.
4. **Token indexing.** `tokenAt` is one-based; index zero and
   out-of-range return empty. Covered by `TestDetail04`.
5. **Mapping destination.** An empty destination is allowed; invalid
   destination tokens report a mapping-destination error; mustache
   tokens must name a known mapping function; the pair must build a
   valid transform. Covered by `TestDetail05`.
6. **Collision.** Equal literal subjects collide; wildcard-vs-literal
   uses subset matching; wildcard-vs-wildcard applies the token-count
   rules before pairwise matching. Covered by `TestDetail06`.
7. **Token matching.** Empty tokens never match; a leading wildcard
   character matches anything; otherwise tokens must be equal. Covered
   by `TestDetail07`.
8. **Token count.** `numTokens` counts separators plus one, with empty
   yielding zero. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — full validity table including the whole-token `>` rule and literal-embedded wildcards |
| TestDetail02 | 2 | partially — bounded-both-sides literal rule asserted |
| TestDetail03 | 3 | partially — the whitespace asymmetry asserted |
| TestDetail04 | 4 | yes — asserted exactly |
| TestDetail05 | 5 | partially — error identity asserted (ErrUnknownMappingDestinationFunction / ErrInvalidMappingDestination), not prose |
| TestDetail06 | 6 | partially — length-rule cases asserted both directions |
| TestDetail07 | 7 | partially — leading-char position-only rule asserted |
| TestDetail08 | 8 | yes — asserted exactly |
