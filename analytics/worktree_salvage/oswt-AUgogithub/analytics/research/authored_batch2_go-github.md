# authored_batch2 — go-github (14 units)

Task-author output. Source: `experiments/pipeline/work/go-github/orig`
(`github.com/google/go-github/v92`, pinned `d6ef767cb5327ec534440efc9466f2bd15e3a6e8`).
Units live under `experiments/pipeline/authored_batch2/go-github/<unit>/_author/`.
No hidden tests were written; a separate verifier session owns `tests/hidden/`.

All excised trees pass `go vet ./...` and `go test -count=1 ./github` under the Go 1.26
toolchain (`GOTOOLCHAIN=auto`). Excision patches apply cleanly to the pristine tree; gold
patches restore the impl files byte-identically and touch no `*_test.go`; cheat patches
compile, differ from gold, and hardcode only the worked-example shapes.

Overlap: `uv run python scripts/check_unit_overlap.py --extra <worktree>` → **CLEAN**
(14 new units vs 0 existing; two INFO-level shared files with disjoint symbol sets:
`github/messages.go` webhooksig/eventdispatch, `github/copilot.go` copilotpoly/ndjsonmetrics).

| unit | closure (symbols) | files | lines | details | predicted_flip | commitments | compiles |
|---|---|---|---|---|---|---|---|
| webhooksig | webhook HMAC signature validation + payload size limit + event metadata (genMAC, checkMAC, messageMAC, ValidateSignature, ValidatePayload, ValidatePayloadFromBody, readPayloadBody, WebHookType, DeliveryID) | github/messages.go, github/messages_test.go | 550 | 10 | L2 | sha-prefix dispatch, malformed-sig errors, constant-time compare, payload cap, content-type rules, header lookups | yes |
| eventdispatch | event-type registry + payload decode (ParseWebHook, MessageTypes, EventForType, Event.ParsePayload, Event.Payload, HookDelivery.ParseRequestPayload) | github/messages.go, github/event.go, github/repos_hooks_deliveries.go + 5 test files | 844 | 6 | L2 | unknown-type behavior differs per entry point (error vs generic-map fallback vs unsupported-event error), fresh-prototype semantics, sorted list, panic-on-error deprecated wrapper | yes |
| auditentry | audit-log scalar/array normalization + catch-all fields (AuditEntry.UnmarshalJSON, unmarshalStringOrArray, unmarshalIntOrArray, AuditEntry.MarshalJSON) | github/orgs_audit_log.go + 2 test files | 698 | 6 | L2 | string-array joins `", "` vs int-array takes first, empty array → nil, null → nil, defined-key subtraction + null-drop in catch-all, marshal collision error | yes |
| treeentryjson | git tree entry wire shape (TreeEntry.MarshalJSON) | github/git_trees.go, github/git_trees_test.go | 367 | 4 | L2 | sha+content-both-nil triggers delete shape, explicit `"sha":null`, size/content/url dropped in delete shape, omitempty normal shape | yes |
| projectsjson | Projects V2 polymorphic JSON (ProjectV2Item.UnmarshalJSON, ProjectV2ItemContent.MarshalJSON, ProjectV2ViewSortBy.{Un,}MarshalJSON) | github/projects.go + 2 test files | 1463 | 11 | L2 | 3-way discriminator, 3-part decode gate, unknown-discriminator empty holder, member priority, all-nil `null`, tuple wire form, string/number ID union, int64 precision, exact-2 length, per-element errors | yes |
| pubkeyjson | secrets public-key key_id union (PublicKey.UnmarshalJSON) | github/actions_secrets.go + 5 test files | 783 | 5 | L2 | verbatim string (leading zeros), number→decimal string, absent/null → nil, other type → error, field passthrough | yes |
| envreview | environment reviewer dispatch + update encoding (RequiredReviewer.UnmarshalJSON, CreateUpdateEnvironment.MarshalJSON) | github/repos_environments.go + 2 test files | 448 | 7 | L2 | User/Team dispatch, missing/null/non-string type errors, unknown type resets state, inner decode error preserves type, wait_timer→0 vs can_admins_bypass→true defaults, always-present null keys | yes |
| copilotpoly | Copilot `any` fields (CopilotSpace.UnmarshalJSON, CopilotSeatDetails.UnmarshalJSON) | github/copilot.go + 2 test files | 1877 | 8 | L2 | owner User/Org dispatch + hooks_url structural fallback, null→nil, non-object→error, assignee 3-way dispatch, missing-type error distinct from unknown-type, inner decode errors | yes |
| ndjsonmetrics | NDJSON metrics stream decode (decodeNDJSONMetrics) | github/copilot.go, github/copilot_test.go | 553 | 4 | L2 | value-boundary (not line) decode, empty body → nil slice, mid-stream error discards partial results, order preservation | yes |
| rulesetjson | ruleset rule polymorphism (RulesetReviewer.UnmarshalJSON, RepositoryRulesetRules.{Un,}MarshalJSON, RepositoryRule.UnmarshalJSON, BranchRules.UnmarshalJSON) | github/rules.go + 4 test files | 5703 | 8 | L2 | `{type,parameters}` array wire shape, assign vs append+metadata, empty-params vs missing-params handling, silent drop of unknown types on all 3 decoders, fixed-order emission, reviewer ID number/string union | yes |
| refescape | per-segment ref URL escaping (refURLEscape; stub returns ref verbatim — tree stays green but wrong) | github/git_refs.go + 2 test files | 224 | 4 | L2 | slash separators preserved (split-escape-join), per-segment percent-encoding, `%`→`%25` re-encoding, empty-segment preservation | yes |
| customprop | custom-property tri-state value (CustomPropertyValue.UnmarshalJSON) | github/orgs_properties.go + 4 test files | 469 | 4 | L2 | string/[]string/nil shapes, per-element string check, other-type error, absent → nil | yes |
| teamsupdate | update-team conditional nullability (UpdateTeamRequest.MarshalJSON) | github/teams.go, github/teams_test.go | 178 | 3 | L2 | flag never serialized, flag-off omitempty, flag-on explicit `null` for exactly the two parent keys | yes |
| netconfig | network-configuration request validation (validateComputeService, validateNetworkName, validateNetworkSettingsID, validateNetworkConfigurationRequest) | github/orgs_network_configurations.go + 2 test files | 529 | 6 | L2 | inclusive 1-100 name bounds, exact charset, nil-vs-enum compute check, exactly-one settings ID, `validation failed:` prefix pre-HTTP, check order | yes |

## Existing closures avoided

- `redirect-until-found` (gensmoke `author_batch/units/`): touches `github/github.go`
  (`Client.bareDoUntilFound`, `Client.roundTripWithOptionalFollowRedirect`,
  `Client.checkRedirectHost`) and `github/repos_contents.go`
  (`RepositoriesService.getArchiveLinkWithoutRateLimit`, `getArchiveLinkWithRateLimit`,
  `GetArchiveLink`). Avoided: all `github.go` URL/redirect/rate-limit helpers and
  `repos_contents.go` were deliberately excluded from the unit slate.
- Within the batch, same-file pairs were kept symbol-disjoint per the checker granularity:
  `messages.go` (signature validation vs event dispatch — no shared funcs) and `copilot.go`
  (polymorphic struct decoders vs the NDJSON stream helper).
- `projects.go`: the sort-criterion and item-content closures were merged into one unit
  (`projectsjson`) because the checker extracts bare method names (`UnmarshalJSON`,
  `MarshalJSON`) from patch decls — two same-file units sharing a method name collide.
  The same reasoning produced one `rulesetjson` covering all `rules.go` JSON funcs.

## Difficulty design notes

- Every unit's DETAILS line marks which commitments are inferable from the remaining tree;
  the deliberately non-inferable ones are the L0 drivers: asymmetric unknown-type behavior
  (eventdispatch, copilotpoly, rulesetjson silent-drop), silent field dropping
  (treeentryjson, auditentry catch-all), opposite marshal defaults (envreview 0 vs true),
  gated decode (projectsjson), structural-fallback dispatch (copilotpoly hooks_url),
  value-vs-line boundaries (ndjsonmetrics), and verbatim-vs-normalized IDs (pubkeyjson).
- `refescape` uses a return-input stub rather than panic, keeping the whole in-tree suite
  green — nothing in the remaining tree hints that per-segment escaping is required.
