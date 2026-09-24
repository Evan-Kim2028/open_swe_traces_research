# Contract (L2) — secschemes

Security schemes, requirements and flows: names, validation, copying and the
no-security escape.

Two name spellings exist for the same scheme kinds and they disagree on
exactly one: the kind's own string form covers every kind including the
no-security kind, while the scheme's type-name form exists only for the five
credential kinds and spells the basic-auth kind with its longer name. Both
lookups reject an unknown kind loudly; the type-name lookup additionally
rejects the no-security kind, which only the string form covers.

A requirement's diagnostic name is a fixed base name extended with the first
scheme's name when that scheme is named — requirements with no schemes or an
unnamed first scheme share the one fixed name. A scheme's diagnostic name is
its type name with a security suffix appended; a flow's is its grant-type
string with a flow prefix prepended. A flow's grant-type strings are the four
standard OAuth2 grant names — authorization code, implicit, password, client
credentials — and an unknown flow kind is rejected loudly.

Flow validation parses all three URLs — token, authorization, refresh — and
produces one error per genuinely malformed URL naming which URL failed; an
empty URL is accepted. Scheme validation merges the errors of every flow, in
flow order.

Copying a requirement preserves its scope list while deep-copying each scheme.
Copying a scheme copies its scalar fields — kind, scheme name, location,
element name, description, bearer format — shares the scopes, flows and
metadata, and points the copy's authored reference at the source's authored
scheme, so a copy-of-a-copy still resolves to the declared original; a scheme
never copied resolves to itself. A scheme's hash is a deterministic value
sensitive to each of its scheme name, location and element name.

A requirement list disables security when ANY scheme in ANY requirement is the
no-security kind; the effective-requirements lookup returns nothing in exactly
that case and the input list otherwise.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | two injective name spellings exist; they disagree on exactly one kind |
| `TestDetail02` | both lookups reject unknown kinds loudly; the type-name form also rejects the no-security kind that the string form covers |
| `TestDetail03` | a requirement's name is the fixed base plus the first scheme's name when named; degenerate scheme lists share the fixed name |
| `TestDetail04` | a scheme's name is its type name plus the security suffix; a flow's is the flow prefix plus its grant-type string |
| `TestDetail05` | grant-type strings are the four standard OAuth2 names; unknown flow kinds rejected loudly |
| `TestDetail06` | one error per malformed URL naming the URL; empty URLs accepted |
| `TestDetail07` | scheme validation merges every flow's errors in flow order |
| `TestDetail08` | copying a requirement shares the scope list but deep-copies each scheme |
| `TestDetail09` | copying a scheme copies scalars, shares scopes/flows/meta, and resolves authored to the declared original even through a copy |
| `TestDetail10` | a never-copied scheme resolves to itself |
| `TestDetail11` | any no-security kind in any requirement disables security; effective requirements return nothing exactly then |
| `TestDetail12` | the hash is deterministic and sensitive to scheme name, location and element name |
