# Verified hidden suites — authored_batch3/kops (20 units)

Date: 2026-09-21. Branch: `vf3kops`. Worktree: `oswt-VF3kops`.

Method: each unit got a black-box suite at `tests/hidden/<pkg>/..._bb_test.go`
(external `_test` package, exported API only) plus `tests/test.sh` (hidden-file
integrity checksums + `go test`). Suites were written against `DETAILS.md` +
kept code only — `gold.patch` was never opened. Behavior that DETAILS could
not settle was probed empirically on the Docker image (`ladder-base:kops`),
never read from gold.

Per-unit verification (four checks via `_tools/verify_hidden.sh`):

| unit | tests | excised | gold | cheat | A12 |
|---|---|---|---|---|---|
| certdesc | 6 | FAIL | PASS | FAIL (build: bytes/fmt) | PASS |
| distros | 8 | FAIL | PASS | FAIL (assert) | PASS |
| fidownload | 6 | FAIL | PASS | FAIL (build) | PASS |
| fieldmap | 6 | FAIL | PASS | FAIL (assert) | PASS |
| fieldpath | 11 | FAIL | PASS | FAIL (assert) | PASS |
| filemodes | 4 | FAIL | PASS | FAIL (build: strconv) | PASS |
| fivalues | 8 | FAIL | PASS | FAIL (build: strconv) | PASS |
| iamsubj | 7 | FAIL | PASS | FAIL (build: fmt/containers) | PASS |
| igrole | 8 | FAIL | PASS | FAIL (assert) | PASS |
| jsonstream | 10 | FAIL | PASS | FAIL (build: fmt) | PASS |
| kopscodecs | 9 | FAIL | PASS | FAIL (build: kubeyaml/unstructured) | PASS |
| reflectfmt | 12 | FAIL | PASS | FAIL (assert: errors.As wrap) | PASS |
| sshfinger | 5 | FAIL | PASS | FAIL (build: fmt/md5/x509) | PASS |
| strvals | 13 | FAIL | PASS | FAIL (assert) | PASS |
| tablesfmt | 7 | FAIL | PASS | FAIL (build: bytes/tabwriter) | PASS |
| tfhcl2 | 14 | FAIL | PASS | FAIL (build: reflect/sort) | PASS |
| tfliterals | 13 | FAIL | PASS | FAIL (assert: sort) | PASS |
| tfwriter | 10 | FAIL | PASS | FAIL (assert: provider reuse) | PASS |
| vfspaths | 8 | FAIL | PASS | FAIL (build: s3) | PASS |
| zonespec | 10 | FAIL | PASS | FAIL (build: strings) | PASS |

All 20 units satisfy all four checks; docker_rc=0 on every run (full sweep
re-run 2026-09-21, all rows reproduced).

12 cheat patches fail at compile time — they replace panic bodies without
restoring blanked imports (`_ "bytes"` etc.). That is a cheat-patch authoring
defect, not a suite defect: the check still reads FAIL. The other 8 cheats are
caught by real assertions.

## Per-unit detail

Convention below: `yes/doc` = asserted exactly; `partially` = derivable part
asserted; `no` = shape only (error returned / error mentions the field /
output has the required shape). "Refused" = a clause not asserted, with reason.

### certdesc (pkg/pki) — 6 tests
Inferable: no, partially, partially, no, no, partially.
- D1 no: DN round-trip asserted as set-equality of key=value pairs, not string order.
- D2 partially: one test bit per kept usage-table entry via `IssueCert`+parse, not a literal table.
- D3 partially: parse→IssueCert→re-parse round-trip for a composed usage list.
- D4 no: EKU asserted as "renders a non-empty name" + round-trip membership, not exact strings.
- D5 no (table kept): canonical type spellings `rsa`, `ca`, `client`, `server`, `serving`, `timestamping` asserted only for the ones named in the kept `wellKnownCertificateTypes`.
- D6 partially: comma-join has no space; `Type:"CA"` issuance is self-signed (nil keystore OK — verified via kept `issue.go`).

### distros (util/pkg/distributions) — 8 tests
Inferable: yes, yes, no, no, no, no, yes, no.
- D3 no: `HasDNF` asserted as a per-project boolean partition consistent with the rpm family, no version literals.
- D4 no: `IsSystemd` asserted `true` for every kept package var (shape: constant predicate).
- D5 no: `DefaultUsers` asserted non-empty for DETAILS-listed projects only; unknown project errors. Refused: enumerating each project's user list (table content not inferable).
- D6 no: asserted as a boolean partition over kept vars, not a member list.
- D8 no: nftables asserted only as "rpm-family ⇒ force", members not pinned.

### fidownload (upup/pkg/fi) — 6 tests
Inferable: partially, no, partially, partially, no, yes.
- D2 no: dispatch asserted as "https URL downloads bytes" (httptest on localhost) and "unsupported scheme errors" — error string not pinned.
- D5 no: non-2xx asserted as "returns error"; status-retry and backoff counts not pinned.

### fieldmap (pkg/apis/kops) — 6 tests
Inferable: partially, partially, partially, yes, yes, no.
- D1/D2 partially: direction of the mapping asserted through one round-trip (`Human→Internal→Human` identity) plus one table entry visible in kept callers.
- D3 partially: unmapped paths pass through verbatim — asserted on a synthetic path.
- D6 no: `HumanPath` returns the v1alpha2 spelling for a known input — asserted only that output differs in the documented direction; exact spelling comes from kept constants.

### fieldpath (util/pkg/reflectutils) — 11 tests
Inferable: yes, no, no, partially, partially, partially, no, yes, yes, no, (11th covers Matches asymmetry detail).
- D2 no: `/` separator asserted only as "parses to same path as `.`".
- D3 no: `a..b` collapses — asserted as parse-success + rendered equality.
- D7 no: wildcard-in-pattern matches concrete index, wildcard-in-target does not — asserted as a boolean asymmetry.
- D10 no: unknown element type → subprocess-death test for `klog.Fatalf` (exit status, not message).
- Divergence: DETAILS D4 implies `[key]` parses; under gold bare map keys are NOT parseable — `[key]` is only the rendered form (`Extend`/`ReflectRecursive` produce MapKey elements programmatically). Tests construct MapKey paths via `Extend`, never parse them.

### filemodes (upup/pkg/fi) — 4 tests
Inferable: partially ×4.
- All four asserted as behavior-shapes (default on empty, octal render round-trips through `ParseFileMode`, mode-mask comparison detects drift, missing file → `(false, nil)`); exact default literal and format widths not pinned beyond what kept constants expose.

### fivalues (upup/pkg/fi) — 8 tests
Inferable: yes, partially, yes, yes, no, partially, yes, yes.
- D5 no: `DebugPrint` nil/typed-nil asserted as "renders a non-empty sentinel, and nil ≠ typed-nil rendering" — sentinel literals not pinned.
- D6 partially: marshal-failure returns the error as a *string field*, asserted via an unmarshalable func field.

### iamsubj (pkg/model/iam) — 7 tests
Inferable: yes, yes, partially, partially, no, no, no.
- D5 no: AWS path asserted as "pod gains exactly one projected volume" — volume name not pinned.
- D6 no: every container gains a read-only mount into the projected volume — asserted by count + readOnly flag, not mount path.
- D7 no: `SecurityContext.FSGroup` asserted set-and-nonzero, value not pinned.

### igrole (pkg/apis/kops) — 8 tests
Inferable: yes, no, partially, no, yes, partially, no, no.
- D2 no: `controlplane`/`control-plane` asserted to resolve identically in both directions; the underscore spelling `control_plane` was tried and REMOVED — gold does not support it and DETAILS commits only to dash/no-dash.
- D4 no: `master` asserted as "lenient resolves, strict rejects" — no literal.
- D7 no: empty input asserted "returns empty, no error".
- D8 no: error asserted as "non-nil, mentions the bad line" — prefix literal not pinned.

### jsonstream (pkg/jsonutils) — 10 tests
Inferable: yes, yes, partially, partially, partially, no, no, no, partially, no.
- D6 no: `}` in field-value state asserted as "no error, `}` emitted, writer still usable". Refused: DETAILS claims the path entry is popped; gold leaves it (`Path()` stays `"k"`). Not asserted — documented divergence.
- D7 no: scalar rendering asserted as round-trip via `encoding/json` decode, not byte-exact format.
- D8 no: top-level scalar → error (non-nil only).
- D10 no: field-name emission asserted via `Path()` contents, not token-stringification internals.

### kopscodecs (pkg/kopscodecs) — 9 tests
Inferable: yes, partially, no, partially, partially, no, no, yes, no.
- D3 no: unstructured input asserted "encodes without version mutation" (apiVersion preserved).
- D6 no: legacy `kops/v1alpha2` asserted as "recognized as the kops API" — i.e. NOT silently decoded as foreign unstructured; a typed-path error is accepted because gold errors there too. Refused: asserting successful rewrite+decode — gold diverges (errors on typed decode of unrewritten bytes).
- D7 no: `kops/v1alpha3` asserted NOT silently typed-decoded.
- D9 no: encode error asserted "mentions object type" via `*unstructured.UnstructuredList`.
- Conflict/divergence: kept comment says group `clustkit` — the real registered group is `kops.k8s.io` (visible in kept `register.go`); the comment is an obfuscation artifact. Tests use `kops.k8s.io` and `kops/v1alpha2` (the real legacy spelling, confirmed in kept-tree evidence).

### reflectfmt (util/pkg/reflectutils) — 12 tests
Inferable: doc, doc, partially, no, partially, no, doc, doc, partially, yes, partially, no.
- D1 doc: root-visited-first asserted; relaxed from "second visit is field A" — gold visits root ptr AND deref'd struct at empty path before fields.
- D4 no: JSONNames asserted "json tag name appears in path" — full path spelling not pinned.
- D6 no: nil ptr/interface asserted "visited once, no descent".
- D12 no: `IsMethodNotFound` asserted as direct type-assert semantics — a `fmt.Errorf("%w")`-wrapped MethodNotFoundError must return false. This is the assertion that catches the cheat (`errors.As`).
- Note: `ReflectRecursive` requires non-nil `*ReflectOptions` on descent; suite passes `&ReflectOptions{}` (kept signature exposes it).

### sshfinger (pkg/pki) — 5 tests
Inferable: partially, yes, no, no, partially.
- All expected fingerprints computed in-test from independently generated keys (md5 over PKIX DER for RSA/AWS; SHA256-OpenSSH form for ed25519; md5 over wire encoding for OpenSSH). No literal fingerprints.
- D3 no: ECDSA → AWS fingerprint errors (shape only); RSA/ed25519 distinct encodings verified by construction.

### strvals (third_party/forked/helmstrvals) — 13 tests
Inferable: yes, partially, partially, partially, doc, no, partially, partially, partially, doc, no, no, no.
- D6 no: nested map in list element asserted via two-pair form `a.b[0].c=v,a.b[0].d=w`. Divergence: single `a.b[0].c=v` DROPS the element under gold — documented; the committed behavior (nesting works) is asserted only where gold upholds it.
- D11 no: `=v` asserted "no panic, empty key absent from result" (silently skipped under gold).
- D12 no: wrong-shape existing dest entry asserted "error returned".
- D13 no: `a={x,y}b=c` asserted both keys present (pushback after `}` confirmed empirically).

### tablesfmt (util/pkg/tables) — 7 tests
Inferable: yes, yes, yes, partially, no, partially, no.
- D5 no: non-slice items → subprocess-death test (exit status, not panic text).
- D7 no: cells asserted tab-separated + header present; exact padding/whitespace refused.

### tfhcl2 (upup/pkg/fi/cloudup/terraform) — 14 tests
Inferable: no, partially, no, no, no, partially, no, no, partially, partially, no, no, partially, no.
Divergences found vs DETAILS (all `no` lines, asserted only where derivable):
- D5: `fieldKey` produces `httpport` for `HTTPPort`, not DETAILS's `http_port` — snake-case claim refused; asserted "cty tag wins" + "untagged name lowercases" only.
- D8: `"` is emitted RAW (unescaped) under gold; DETAILS says `"` and `\` are escaped. Asserted only newline/tab pass-through shape; escape claim refused.
- D14: unknown providers (`linode`, `metal`) produce an EMPTY `required_providers` block, not a Fatalf — fatal claim refused; asserted block presence only.
- D1/D9 partially: section order + relative indentation asserted; exact alignment refused (`count` sits outside gold's aligned run — asserted per observed run membership).
Provider table asserted from kept constants: aws, google, hcloud, azurerm, digitalocean, scaleway; extra `files` provider gets `alias = "files"` + `configuration_aliases`.

### tfliterals (upup/pkg/fi/cloudup/terraformWriter) — 13 tests
Inferable: yes ×6, partially ×3, no ×2, doc ×1, mixed.
- D5 no: `LiteralFromStringValue` asserted verbatim round-trip (output contains input string).
- D10 no: `Write` asserted "`= ` + String + newline shape" via regex, not byte-exact.
- D12 doc/D13: `SortLiterals` ascending by `.String`, in-place — this is the assertion the cheat fails.

### tfwriter (upup/pkg/fi/cloudup/terraformWriter) — 10 tests
Inferable: no, partially, partially, no, yes, partially, no, partially, partially, yes.
- D1 no-but-in-contract: sanitize map (`.`→`-`, `/`→`--`, `:`→`_`, leading digit→`prefix_`) pinned — it is solver-facing in the contract, so pinning is fair.
- D4 no: duplicate scalar output key → error; array appends — asserted as error-presence + append-count.
- D7 no: identical provider re-registration returns existing (catches cheat); conflicting → fatal asserted via subprocess-death.
- Render* asserted insertion-order preserved; exact HCL layout not pinned.

### vfspaths (util/pkg/vfs) — 8 tests
Inferable: yes, partially, no, no, no, partially, no, partially.
- D3 no: do/linode/hos/scw → error without `S3_ENDPOINT`, `*vfs.S3Path` with it; s3 works without.
- D4 no: endpoint consumers asserted only as "constructs an S3Path" — endpoint internals not exported.
- D5 no: `AZURE_STORAGE_ACCOUNT` set → error; `azureblob://acct` (no container) → error; valid → `*vfs.AzureBlobPath`.
- D7 no: `RetryWithBackoff` — condition runs before first sleep (Steps=1, Duration=1h completes instantly), Steps counts attempts exactly, exhausted result returns the condition's own `(false, sentinel)` not a timeout error.
- D8 partially: capped steep backoff (Factor=1000, Cap=20ms) completes <3s where uncapped would take ~10s.

### zonespec (dns-controller/pkg/dns) — 10 tests
Inferable: yes, partially, yes, no, yes, partially, partially, no, yes, partially.
- D4 no: first-`/` split asserted as "`a/b/c` → Name=`a.`, ID=`b/c`" — shape over literal.
- D8 no: name-mismatch rule skipped, ID-mismatch returns false immediately — asserted as boolean pair.

## Cross-cutting conflicts and notes

- DETAILS-vs-gold divergences (assertions refused, listed above): fieldpath `[key]` parseable; jsonstream path-pop on `}`-in-F; kopscodecs legacy rewrite success + `clustkit` group comment; strvals single-touch nested element; tfhcl2 `http_port` spelling, quote escaping, unlisted-provider fatal. In every case the test was narrowed to the derivable/fair subset rather than bent to gold — the divergence is documented, not silently resolved.
- Contract-vs-DETAILS: no contract row was found asserting against a DETAILS line; contracts stayed generic (no literals). tfwriter's contract legitimately exposes the sanitize map.
- Cheat-patch defect: 12/20 cheats fail to compile because excision blanks imports (`_ "x"`) that cheat never restores. Suites remain correct — if the import defect were fixed, the same assertions stand on their own.
- No `gold.patch` file was opened at any point; all behavioral questions were resolved by probing compiled behavior in Docker or by reading kept (non-excised) source.
