# Commitments — exgen

1. Example returns the LAST user-supplied example when present, then
   honors "openapi:example"/"swagger:example" meta = "false" by
   returning nil, then picks the generator path in priority order:
   length > enum > format > pattern > min/max, falling back to the
   type's own Example. In-tree coverage: deleted tests only.
   Inferable: partially — ordering is documented in prose.
2. Pattern and min/max constraints retry up to maxAttempts times: a
   format-driven example that fails the pattern or bounds is redrawn.
   In-tree coverage: deleted tests only. Inferable: no.
3. NewLength picks a count inside [MinLength, MaxLength], biased toward
   the bound when only one side is set, and clamps to a hard maximum.
   In-tree coverage: deleted tests only. Inferable: no — the bias and
   clamp value are arbitrary.
4. byLength dispatches on the UNALIASED type: strings get Characters,
   bytes get []byte(Characters), maps build count entries, arrays build
   count elements. In-tree coverage: deleted tests only. Inferable:
   partially.
5. byFormat maps each supported ValidationFormat to a matching random
   producer (email, hostname, date, date-time, IPv4/IPv6, URI, MAC,
   CIDR, regexp, RFC1123, UUID, JSON literal) and panics on unknown
   formats. In-tree coverage: deleted tests only. Inferable: partially
   — the format list is documented, per-format values are not.
6. patgen walks a simplified regexp AST (literal, capture, concat,
   alternate, star, plus, quest, repeat, char class, any) generating a
   satisfying string. In-tree coverage: deleted tests only. Inferable:
   no.
7. byMinMax draws a number inside the effective bounds, honoring
   exclusive bounds and sign flips when only a maximum is given.
   In-tree coverage: deleted tests only. Inferable: no.
8. checkPattern/checkMinMaxValue verify a candidate example and are
   used by the retry loop. In-tree coverage: deleted tests only.
   Inferable: partially.
