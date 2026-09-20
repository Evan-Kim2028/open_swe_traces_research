# Commitments — secschemes

1. The scheme-type name mapping has two DIFFERENT spellings for the same
   kinds: the method used for error names yields "OAuth2", "BasicAuth",
   "APIKey", "Bearer", "JWT", while the kind's own string form yields
   "OAuth2", "Basic", "APIKey", "Bearer", "JWT", "None" (for the
   no-security kind). In-tree coverage: none. Inferable: no — the two
   spellings disagree for exactly one kind.
2. Both name lookups panic on an unknown kind value, and the scheme-type
   method also panics for the no-security kind (the kind's string form does
   not). In-tree coverage: none. Inferable: no.
3. A requirement's error name is "Security" plus "scheme <name>" appended
   directly (no space) when the first scheme has a non-empty name, and just
   "Security" for empty or nil scheme lists. In-tree coverage: none.
   Inferable: no.
4. A scheme's error name is its type name followed by "Security"; a flow's
   error name is "flow " plus its grant-type string. In-tree coverage: none.
   Inferable: partially — follows once the type strings are known.
5. Flow grant-type strings are the OAuth2 names "authorization_code",
   "implicit", "password", "client_credentials"; unknown kinds panic.
   In-tree coverage: none. Inferable: doc — standard grant names.
6. Flow validation parses all three URLs (token, authorization, refresh);
   an EMPTY URL is accepted (parse of "" succeeds) and only genuinely
   malformed URLs produce "invalid <which> URL %q: %s" entries, in field
   order. In-tree coverage: none. Inferable: no.
7. Scheme validation returns the merged errors of every flow, in order.
   In-tree coverage: none. Inferable: partially.
8. Copying a requirement shares the scopes slice (same backing array) but
   deep-copies each scheme. In-tree coverage: none. Inferable: no.
9. Copying a scheme copies scalar fields but SHARES the scopes, flows and
   meta slices; the copy's authored pointer is set to the source's authored
   scheme (so a copy-of-a-copy still points at the declared original).
   In-tree coverage: none. Inferable: no.
10. AuthoredScheme returns the stored authored scheme when set, otherwise
    the receiver. In-tree coverage: none. Inferable: doc — stated in the
    method's comment.
11. HasNoSecurity reports true when ANY scheme in ANY requirement is the
    no-security kind; EffectiveSecurityRequirements returns nil in exactly
    that case and the input slice otherwise. In-tree coverage: none.
    Inferable: partially — names imply it, "any vs first" is the detail.
12. A scheme's hash is SchemeName, In and Name joined with underscores in
    that order. In-tree coverage: none. Inferable: no.
