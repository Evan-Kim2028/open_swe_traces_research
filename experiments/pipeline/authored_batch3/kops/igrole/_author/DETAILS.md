# Details — igrole

1. Matching is case-insensitive against the lowercased role names of `AllInstanceGroupRoles` (visible in `instancegroup.go`). Inferable: yes.
2. `controlplane` (no dash) is normalized to `control-plane` in BOTH strict and lenient modes — the replacement is unconditional. Inferable: no — surprising that strict mode still corrects it.
3. Lenient mode strips a trailing `s` from the input AND from each candidate role name — pluralization is tolerated on both sides. Inferable: partially — the two-sided trim is a quirk.
4. `master` resolves to the control-plane role only in lenient mode; strict mode rejects it. Inferable: no — legacy alias is hidden.
5. Failure returns `("", false)`, never an error. Inferable: yes — signature.
6. `ParseRawYaml` rejects documents with unknown fields (strict unmarshal). Inferable: partially — strictness is a choice.
7. `ParseRawYaml` skips unmarshal entirely for empty/whitespace input and returns success. Inferable: no — silent no-op.
8. Parse/marshal errors are wrapped with a fixed "error parsing configuration" / "error converting to yaml" prefix. Inferable: no — literal wrap text.
