# Why hard — formmapping

predicted_flip: L5
a 550-line reflection engine with the densest invariant set in the repo: lazy pointer allocation that doubles as cycle protection, tag-option parsing (default=, parser=, collection_format), per-kind conversion with subtle empty-string coercion, JSON fallback for struct/map fields, custom unmarshaler precedence, and map-target special cases. Most invariants are invisible in signatures; a contract-only rewrite still needs a hidden test file to converge.
