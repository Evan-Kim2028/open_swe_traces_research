# authored_batch4 go-github second pass — 20 new units

Second mining pass over `github.com/google/go-github/v92` (the bank's
highest-yield repo). All 20 units are in
`experiments/pipeline/authored_batch4/go-github/<unit>/_author/` with
gold.patch, cheat.patch, api.md, DETAILS.md (annotated), bugreport.md,
closure.md, and excised/excision.patch.

Tooling: `experiments/pipeline/authored_batch4/_tools/mkunit4.py` (JSON spec
→ excision+gold+cheat patches) and `verify4.sh` (apply excision → build →
test green; apply gold → closure tests pass; apply cheat → closure tests
fail; size ratio). Every unit below passed all four gates.

"Attempts" counts generation+verify cycles for that unit, including rejected
cheats and missed deltests — not candidate reading.

## Overlap check

`scripts/check_unit_overlap.py` — required a one-line fix first:
`EXCISED_RE` truncated `excised:` markers at the `.`, so two units excising
*different methods on the same receiver type* in the same file collided
(`AuditEntry.MarshalJSON` vs `AuditEntry.UnmarshalJSON`, and pre-existing
`sha1`/`commitraw` on `RepositoriesService.*`). Widened the symbol charset
to include `.`; after the fix the check reports **CLEAN — 29 new units vs
21 existing, 0 overlaps**, all file-shares symbol-disjoint.

Manual grep over `experiments/pipeline/authored*/{*,*/*}/_author/closure.md`
for every excised symbol found no collision (only an unrelated kops
`parseSSHPublicKey` mention in sshfinger).

### Candidates rejected for overlap

Surveyed before authoring; each collides with an existing unit:

| Candidate | Existing unit |
|---|---|
| `Client.checkRequestAPIVersionBeforeDo` | apiversion |
| `New*StreamConfig` family (S3/Azure/GCS/Hec/Splunk/Datadog/Hub) | auditstream |
| `parseBoolResponse` | boolresp |
| `isBranchNotProtected` | brnotprot |
| `PullRequestReviewRequest.isComfortFade` | comfortfade |
| `setCredentialsAsHeaders` | credhdrs |
| `parseURL`, `WithEnterpriseURLs` | enturls |
| error formatting/comparison helpers | errfmt, erris |
| `Response.populatePageValues` | pagevalues |
| `CustomProperty.DefaultValue{Bool,String,Strings}` | propvalues |
| rate-limit parse/check helpers | ratecat, ratehdrs |
| `CheckResponse` classification | respclassify |
| `DeploymentProtectionRuleEvent.GetRunID` | runidre |
| URL authorization/origin policy | urlpolicy |
| `RepositoriesService.GetCommitSHA1` | sha1 |
| `RepositoriesService.{GetCommitRaw,CompareCommitsRaw}` | commitraw |
| `RepositoryContent.GetContent` | getcontent |
| `PullRequestsService.GetRaw` | pullraw |
| `putRequestBuffer` | putbuf |
| `RepositoriesService.Create` | repocreate |
| `stringifyValue` | stringify |
| `RepositoriesService.createWebSubRequest` | websub |
| `Alert.ID` | alertid |

Rejected for other reasons: `addOptions` (github.go) — feasible closure but
excising it breaks hundreds of list-endpoint tests; not overlap, just not a
clean cut. Event-payload-only `Event.ParsePayload` — first accepted closure
for eventpayload, widened to include `ParseWebHook` when the cheat couldn't
stay under the size floor.

## Units in authoring order

| # | unit | closure | surface | attempts | Inferable y/d/p/n | cheat ratio |
|---|------|---------|---------|----------|-------------------|-------------|
| 1 | websig | `messageMAC`, `checkMAC`, `signatureFromHeader` (messages.go) — webhook HMAC prefix dispatch + verify | crypto predicate | 2 | 4/1/0/2 | 0.17 |
| 2 | payloadbody | `readPayloadBody`, `ValidatePayloadFromBody` — capped body read + content-type/form extraction | parsing | 1 | 3/4/1/1 | 0.11 |
| 3 | eventpayload | `Event.ParsePayload` + `ParseWebHook` — typed webhook dispatch | polymorphic decode | 4 | 3/3/1/1 | 0.38 |
| 4 | reviewerid | `RulesetReviewer.UnmarshalJSON` — int/string `id` coercion | scalar coercion | 2 | 5/1/0/1 | 0.39 |
| 5 | reporule | `RepositoryRule.UnmarshalJSON` — 20-way type→parameters dispatch | polymorphic decode | 1 | 3/0/2/1 | 0.11 |
| 6 | branchrules | `BranchRules.UnmarshalJSON` — flat array → typed rule slices | demux decode | 1 | 5/0/0/1 | 0.06 |
| 7 | rulesetcodec | `RepositoryRulesetRules.MarshalJSON`+`UnmarshalJSON` — struct↔rules-array codec | codec | 2 | 3/0/2/2 | 0.05 |
| 8 | auditentry | `AuditEntry.MarshalJSON` — AdditionalFields merge + collision error | serialization | 1 | 3/0/1/1+ | 0.45 |
| 9 | auditcoerce | `AuditEntry.UnmarshalJSON` — org/org_id scalar∥array + AdditionalFields diff | coercion | 3 | 2/2/2/1 | 0.29 |
| 10 | pubkey | `PublicKey.UnmarshalJSON` — numeric `key_id`→string | scalar coercion | 2 | 2/1/1/2 | 0.39 |
| 11 | treeentry | `TreeEntry.MarshalJSON` — sha:null delete marker | serialization | 1 | 4/0/1/0 | 0.17 |
| 12 | envdefaults | `CreateUpdateEnvironment.MarshalJSON` — WaitTimer→0, CanAdminsBypass→true defaults | serialization | 1 | 1/3/0/1 | 0.27 |
| 13 | reqreviewer | `RequiredReviewer.UnmarshalJSON` — User/Team `reviewer` dispatch | polymorphic decode | 2 | 3/0/2/1 | 0.42 |
| 14 | propval | `CustomPropertyValue.UnmarshalJSON` — value string∥[]string∥null | coercion | 2 | 2/1/2/1 | 0.25 |
| 15 | viewsort | `ProjectV2ViewSortBy` codec — `[field_id, direction]` tuple, int64∥string | codec | 1 | 1/2/2/1 | 0.28 |
| 16 | projectitem | `ProjectV2ItemContent.MarshalJSON` + `ProjectV2Item.UnmarshalJSON` — content_type union | union codec | 1 | 2/1/1/2 | 0.53 |
| 17 | pkgversion | `PackageVersion.GetBody{,AsPackageVersionBody}`/`GetMetadata`/`GetRawMetadata` — RawMessage typed views | coercion predicates | 2 | 4/0/0/1+ | 0.19 |
| 18 | copilotspace | `CopilotSpace.UnmarshalJSON` — User/Organization `owner` (+hooks_url heuristic) | polymorphic decode | 1 | 2/1/1/2 | 0.44 |
| 19 | copilotseat | `CopilotSeatDetails.UnmarshalJSON` — User/Team/Org `assignee` dispatch | polymorphic decode | 2 | 2/0/3/1+ | 0.25 |
| 20 | ndjson | `decodeNDJSONMetrics[T]` — streaming NDJSON → `[]*T` | stream parse | 1 | 2/2/1/0 | 0.38 |

(+ one Inferable value differs in wording from the strict 4-way table where
noted; every commitment line carries an `Inferable:` annotation.)

All 20 cheats validated by `scripts/ops/cheat_validity.py` — every ratio
below the 0.6 implementation floor; each cheat fails the closure suite while
the excised tree stays green.

## Search-cost trend

Attempts per unit: 2,1,4,2,1,1,2,1,3,2,1,1,2,2,1,1,2,1,2,1 — mean ≈ 1.75,
median 1.5. The two outliers were mechanical, not search failures:
eventpayload needed a closure widening (cheat had no size headroom under
4-line gold) and auditcoerce needed a whole-file test deletion discovered
late. **Search cost per accepted unit was not rising** — units 15–20 went in
at 1–2 attempts each, same as units 1–10. The go-github polymorphic-JSON
seam is still not exhausted (the bank's yield profile held: every unit here
is parsing/predicate/serialization, no client-orchestration glue). Stopped
at the requested count, not at diminishing returns.
