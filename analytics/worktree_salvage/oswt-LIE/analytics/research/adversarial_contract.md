# Adversarial contract: is the instruction the information channel?

2026-09-19. Package + preflight only. Composer trials are a parent-session job.
Staging: `/home/evan/Documents/open_swe_traces_research/experiments/dose_response/sweep_lie/`.
Log: `outputs/LIE.log`. Lever: `uv run python scripts/build_lie_dirs.py`.

## Question

The L0-L6 ladder treats the instruction as the thing that makes a unit solvable.
That was never isolated. Today's four L2 non-flippers (`helm/depresolver`,
`helm/ignorerules`, `helm/repindex`, `kops/clustervalid`) failed because the
contract contradicted gold, and the model followed the prose. Four accidents
are not a result. This experiment lies on purpose.

Take the eight units that flipped L0(0/3) → L2(pass). Build **L2-lie**: byte-identical
to the preflight-PASS L2 dir except one coverage-table row and its invariant prose,
rewritten to assert the opposite of gold. Hidden tests still assert gold. A model
that believes the lie must fail that property.

## Three readings (pre-registered, no trials yet)

- **Still passes 3/3.** The model did not use that part of the contract. It inferred
  the behaviour from the tree, the callers, or the in-tree tests. L0→L2 flips are
  the contract telling the model *where to look*, not *what to do*. That reframes
  every ladder result we have.
- **Fails on exactly the lied-about property.** The contract is load-bearing and
  outranks the code. A contract defect is a silent task-killer, which is what the
  four L2 non-flippers already suggested.
- **Fails on something else / degrades generally.** The lie confused it rather than
  misinformed it.

## Prediction (before any Composer trial)

I predict **fails on exactly the lied-about property for 7 of 8**, and
**assetsremap still passes 3/3**.

Reasoning. At L0 these units reimplemented the bulk and missed one edge. Stating
that edge in the L2 table flipped 7 of them 3/3. The tree was sitting there the
whole time and they did not use it for that fact. Inverting the same fact is the
sharp test: if they now implement the lie, the contract is the channel; if they
still implement gold, they were never reading that sentence.

Six lies are the L0 miss itself (jsonrenders wrap, streamrenders empty Content-Length,
addonparse empty-input, issuecert SAN trim, memfs exclusive create, templater
`mainTemplate` collision). Those should fail the matching hidden test and pass the
rest.

oidcdisc is slightly softer. L0 was mixed (2/3 Get-missing, 1/3 404-when-no-OIDC).
The table row I can invert cleanly is the 404. I still expect a fail on
`TestOIDCDiscoveryDocumentProperty` because that row is now a direct instruction
to return 200.

assetsremap is the odd one out. L0 and 1/3 of L2 failed registry converge, so I
refused to lie about converge (a 2/3 "got it right" is noise). The lie is comma
→ `%2C` in file remap, which they may already have inferred from callers. If this
one still passes, that is evidence the contract is load-bearing *only for facts
the tree does not pin*, not for every row.

I do not expect general degradation. The rest of the contract, the excised tree,
and the hidden suite are unchanged.

## Packaging

Source L2 dirs: `oswt-closureK/experiments/pipeline/tasks_composerver/<repo>/<unit>-L2`
(preflight-PASS). Dest name: `<repo>-<unit>-L2lie`. Copy is hardlinked; `instruction.md`
is a new inode; `validation.json` is copied (not linked) so preflight cannot mutate
source. `task.toml` is the Cursor allowlist already on the L2 dir.

Checksums are `dir_digest` (sorted relpath + file sha256, skip symlinks), the same
function the preflight cache uses. Independent recomputation matched the package
report. After preflight the only extra diff is `validation.json` (the gate writes
A1/A8/A3 + a `preflight` block). `tests/hidden/` and `environment/src` stayed
byte-identical.

| dest | hidden sha256 | environment/src sha256 | diffs vs L2 (post-preflight) |
|---|---|---|---|
| gin-jsonrenders-L2lie | `fb2fda185b5bf7cf97bcd3fe13836707146ba0307f7c969e22af7b6111395354` | `41de7fb7782c5be5d3c66c0c28012957082d99b5ea272a4070574393fe577ad7` | instruction.md, validation.json |
| gin-streamrenders-L2lie | `11fa7f7db46f0fc419e5b3fc5557fa8c309a6b09d457ec2cce542b9b143035a9` | `3ae3940414fd5b8324a3847bba74dc6195c666eeae21dc7a098f585a33877b8e` | instruction.md, validation.json |
| kops-addonparse-L2lie | `1405c71f40c905ffddacb979e6fdc4c5170c2ecdeff45e4abb9e988ad03edac7` | `1a832fb9110ae1aab254dc03f97c29ffa59f9e4c35b17a9a8d01c9108e35890d` | instruction.md, validation.json |
| kops-issuecert-L2lie | `b3bc81b98f0ff9a283ef11dec9b155150ce6361b8e060500ad9e899071737d07` | `801dfb5bdc2c99752656402320cbd51f3b2cdeee5d399faf21d4c55ccba8c0c4` | instruction.md, validation.json |
| kops-memfs-L2lie | `b9a182395a1dff5574ac65fe279973655aab22e93ee90708ad2633fa82229683` | `0976462b453cf478662a2970f7c08d1f77d893e5c8344c6a8e2cd3859ad145f8` | instruction.md, validation.json |
| kops-oidcdisc-L2lie | `c497eff0b22fe190cd01d574b630070083a1f6cd4cbe5ec77e1d59360a44c7de` | `f938858ca397a60fe1a36209f12460c18bc9aeb067ef64e3a84b79cc1c19b9c5` | instruction.md, validation.json |
| kops-templater-L2lie | `c9b774f59bcd6bc6a28dbba97d58a4756bd7285822130a70648b89f5c76abe0d` | `6e903b309e5bd28ef7a8313d6863f4f7ae666ab176ed517aba871dcd8ed4cba8` | instruction.md, validation.json |
| kops-assetsremap-L2lie | `5c3ff74ad622b302668f6c53eb7bdb4812912e6fd0eaf91e8b18d55e3c582d93` | `b3031ce6a579fbf2a8938bf86ff2f319c6295053768e3644b4104a07976334d8` | instruction.md, validation.json |

Preflight reused the source L2 `openswe-audit:*` images (src trees are identical, so
a rebuild would have been 6× ~7.6 GB of kops layers for no reason). Suite runs were
fresh, not cached.

## Preflight (bare fail, gold pass, cheat fail)

The lie must not change this. Gold still implements gold; the hidden suite still
tests gold.

| dest | bare | gold | cheat | seconds | verdict |
|---|---|---|---|---|---|
| gin-jsonrenders-L2lie | fail | pass | fail | 22 | PASS |
| gin-streamrenders-L2lie | fail | pass | fail | 22 | PASS |
| kops-addonparse-L2lie | fail | pass | fail | 95 | PASS |
| kops-issuecert-L2lie | fail | pass | fail | 32 | PASS |
| kops-memfs-L2lie | fail | pass | fail | 110 | PASS |
| kops-oidcdisc-L2lie | fail | pass | fail | 56 | PASS |
| kops-templater-L2lie | fail | pass | fail | 75 | PASS |
| kops-assetsremap-L2lie | fail | pass | fail | 57 | PASS |

8/8 PASS. Bare evidence is the excision panic (`WriteJSON`, `ParseAddons`,
`IssueCert`, …). Gold `REWARD=1`. Cheat fails on an assertion or a nil deref in
the cheat body. Same shape as the source L2 preflight in
`oswt-closureK/analytics/research/preflight_report_2026-09-19.md`.

## Per-unit lie

Each row is a commitment the model got right at L2 (7 of 8 were 3/3; assetsremap
was 2/3, so the lie is *not* the 1/3 miss).

### gin/jsonrenders

L0 0/3 `TestJsonpJSONCallbackProperty` wrap mismatch. L2 3/3.

**Commitment.** Non-empty callback writes `JSEscapeString(callback)+'('+json+');'`.

**Before (prose).** `JsonpJSON uses content type 'application/javascript; charset=utf-8': with an empty callback it writes the raw JSON; otherwise it writes JSEscapeString(callback) + '(' + json + ');'.`

**After.** `JsonpJSON uses content type 'application/javascript; charset=utf-8': it always writes the raw JSON and never wraps it in callback(...); the callback value is ignored.`

**Before (row).** `` `TestRenderJsonpJSON(+Error,+Error2,+Fail)` | callback escaping, '(' json ');' wrapping, empty-callback passthrough, javascript content type ``

**After.** `` `TestRenderJsonpJSON(+Error,+Error2,+Fail)` | callback is ignored; output is always raw JSON with javascript content type (no wrapping) ``

**Hidden test that asserts the truth.** `TestJsonpJSONCallbackProperty` (empty-callback passthrough still holds under the lie, so a follower fails this one test, not `TestJsonpJSONEmptyCallbackProperty`).

### gin/streamrenders

L0 0/3 `TestDataContentLengthProperty` empty data must not set Content-Length. L2 3/3.

**Commitment.** Data writes Content-Length only when the payload is non-empty.

**Before (prose).** `Data writes Content-Length only when the payload is non-empty and then writes the bytes.`

**After.** `Data always writes Content-Length (including when the payload is empty, as "0") and then writes the bytes.`

**Before (row).** `` `TestRenderData/TestRenderDataContentLength/TestRenderDataError` | bytes written with Content-Length only when non-empty ``

**After.** `` `TestRenderData/TestRenderDataContentLength/TestRenderDataError` | bytes written with Content-Length always set, including "0" for empty payloads ``

**Hidden test.** `TestDataContentLengthProperty`.

The unchanged redirect sentence still overclaims: gold panics outside 300-308 except 201, the contract says all 3xx. Hidden tests skip 309-399, so this did not block the L2 flip. If a lie trial fails `TestRedirectStatusProperty` instead of Content-Length, that is this latent defect.

### kops/addonparse

L0 0/3 `TestAddonParseEmptyInvalidProperty` empty/invalid input. L2 3/3.

**Commitment.** Empty/whitespace/comment-only input yields an empty addons list, not an error.

**Before (prose).** `Empty/whitespace/comment-only input yields an empty addons list (not an error).`

**After.** `Empty/whitespace/comment-only input is an error.`

**Before (row).** `` `TestParseAddonsEmpty` | empty input is not an error ``

**After.** `` `TestParseAddonsEmpty` | empty input is an error ``

**Hidden test.** `TestAddonParseEmptyInvalidProperty`.

### kops/issuecert

L0 0/3 `TestIssuecertClientServerProperty` `IP SAN: []` on `" 10.0.0.1 "`. L2 3/3.

**Commitment.** Alternate names are trimmed, then split into IP vs DNS; empties skipped.

**Before (prose).** `Subject is copied; each alternate name is trimmed, IPs vs DNS split, empties skipped.`

**After.** `Subject is copied; each alternate name is used as-is without trimming — a token is an IP SAN only if the entire string parses as an IP, otherwise it is a DNS SAN (empty strings are kept as empty DNS names).`

**Before (row).** `` `TestIssueCert` (clientServer) | DNS + IP SANs; both client and server EKU ``

**After (row).** `` `TestIssueCert` (clientServer) | untrimmed alternate names: whitespace-padded IPs are DNS SANs; both client and server EKU ``

**Hidden test.** `TestIssuecertClientServerProperty` (`AlternateNames: []string{" 10.0.0.1 ", "host.example.com", ""}`).

### kops/memfs

L0 0/3 `TestMemFsCreateWriteProperty` create: file already exists. L2 3/3.

**Commitment.** Create is exclusive: a second Create returns exists.

**Before (prose).** `Create is exclusive: a node that already has contents returns exists; Write always replaces contents.`

**After.** `Create is not exclusive: a second Create overwrites contents the same way Write does; it does not return exists.`

**Before (row).** `` `TestMemFsCreateFile` | exclusive create vs overwrite write; exists on second create ``

**After.** `` `TestMemFsCreateFile` | create overwrites like write; a second create succeeds ``

**Hidden test.** `TestMemFsCreateWriteProperty`.

### kops/oidcdisc

L0 mixed: 2/3 `TestOIDCMemoryStoreListProperty` get missing, 1/3 `TestOIDCDiscoveryDocumentProperty` no oidc spec status 200. L2 3/3.

**Commitment.** Discovery document is 404 unless at least one endpoint has an OIDC spec.

**Before (prose).** `The document is 404 unless at least one endpoint in the universe has an OIDC spec;`

**After.** `The document is 200 even when no endpoint in the universe has an OIDC spec (an empty/default discovery document);`

**Before (row).** `` `TestOIDCDiscovery` | well-known document issuer/jwks_uri and 404 when no OIDC spec ``

**After.** `` `TestOIDCDiscovery` | well-known document issuer/jwks_uri and 200 with a default document when no OIDC spec ``

**Hidden test.** `TestOIDCDiscoveryDocumentProperty`.

### kops/templater

L0 0/3 `TestTemplaterSnippetNameProperty` snippet named `mainTemplate` must be rejected. L2 3/3.

The collision invariant lives in the prose. The closest coverage row was `TestRenderSnippet` (include named snippet). Both were rewritten.

**Before (prose).** `Each snippet is parsed under its filename; colliding with `mainTemplate` is an error.`

**After.** `Each snippet is parsed under its filename; a snippet named `mainTemplate` is accepted and replaces the main template.`

**Before (row).** `` `TestRenderSnippet` | include named snippet ``

**After.** `` `TestRenderSnippet` | a snippet named mainTemplate is valid and overrides the main template ``

**Hidden test.** `TestTemplaterSnippetNameProperty`. Include still works; the remaining prose still says `Include executes a named snippet with the given context map.`

### kops/assetsremap

L0 0/3 and 1/3 of L2: `TestAssetsRemapRegistryConvergeProperty`. Not used. Lie is a property the passing L2 trials got right.

**Commitment.** Commas in a remapped file-repository path are `%2C`.

**Before (prose).** `If a file repository is set, join repository path with the canonical path; commas in the escaped path become `%2C`.`

**After.** `If a file repository is set, join repository path with the canonical path; commas in the path are left as literal commas (not percent-encoded).`

**Before (row).** `` `TestRemapURLPathDelimiterEscaping` | commas in file paths are `%2C` ``

**After.** `` `TestRemapURLPathDelimiterEscaping` | commas in file paths stay as literal commas ``

**Hidden test.** `TestAssetsRemapFileContractTableProperty` (want `s3://artifact-bucket/prefix%2Cprod/...`).

## Reproduce packaging

```
uv run pytest -q tests/test_adversarial_contract.py tests/test_preflight.py
uv run python scripts/build_lie_dirs.py --package-only
uv run python scripts/build_lie_dirs.py --preflight-only
```

Parent trial launch (from `open_swe_traces_research`, Cursor allowlist already on the dirs):

```
harbor run --path experiments/dose_response/sweep_lie --agent cursor-cli --model cursor/composer-2.5 \
  --n-concurrent 4 --n-attempts 3 --max-retries 1 \
  --jobs-dir experiments/dose_response/jobs --job-name sweep_lie --yes
```

## Bank-wide contract-vs-gold (50 preflight-clean units)

Solver-free. Dump: `uv run python scripts/audit_contract_gold.py` →
`outputs/contract_gold_dump/`. Helm/kops/gin from `oswt-closureK`, client-go/goa
from `oswt-closureS`. Every coverage row re-read against gold.patch and the hidden
suite.

A unit is defective if any coverage row is **false** of gold, or any hidden
assertion is **missing** from both the table and the prose. Vague rows are
reported and not counted in the defect rate unless they would send a solver the
wrong way.

The four L2 non-flippers are the calibration:

| unit | class | gold vs contract |
|---|---|---|
| helm/depresolver | false | contract: "sha256 over the canonicalized (sorted) requirement set". gold: `json.Marshal([2][]*chart.Dependency{req, lock})`. Hidden `TestDRHashReqStable` asserts an exact digest of that encoding. |
| helm/ignorerules | false | contract: "`!` is a literal"; "`**` does" cross `/`. gold: `p.negate = true` on `!`; `**` is `errors.New("double-star (**) syntax is not supported")`. |
| helm/repindex | false | contract: absolute URL used as-is; existing entry replaced. gold: `filepath.Split` then `URLJoin`; `append`, so duplicates accumulate. Hidden `TestRIContractTable`: `duplicate add did not append`. |
| kops/clustervalid | missing | constructor must error on an empty InstanceGroup list (`TestClusterValidConstructorProperty`: `empty ig list`). No row, no prose. |

**Defect rate: 20/50 = 40%.** 11/50 have at least one **false** coverage row (the class that killed the four L2 non-flippers). 9/50 are missing-only (a hidden assertion with no row and no prose). Vague rows were recorded in the per-repo JSON and are not in this rate.

If the L2-lie trials show the contract is load-bearing, this 40% is the bank's silent error rate and it matters more than any single Composer result today. The four L2 non-flippers were not a 33% tail of the hard set. They were the subset of this 40% that the model actually hit.

| repo | n | defective | false-row units | missing-only |
|---|---|---|---|---|
| helm | 10 | 8 | 7 | 1 |
| kops | 10 | 2 | 0 | 2 |
| gin | 10 | 3 | 2 | 1 |
| goa | 10 | 2 | 2 | 0 |
| client-go | 10 | 5 | 0 | 5 |
| **bank** | **50** | **20** | **11** | **9** |

Helm is the outlier (8/10). That is not an audit artefact: chartloader, kindsorter, memorydriver, storage, and the three known helm non-flippers all have quoted gold vs hidden contradictions. client-go's 5/10 are all missing, not false: the table is true as far as it goes and the hidden suite tests extra excised API.

| unit | class | gold vs contract |
|---|---|---|
| helm/chartloader | false | "device files skipped" / "backslash paths rejected". gold errors on irregular files; backslash archives load. |
| helm/depresolver | false | HashReq is `json.Marshal([2][]*chart.Dependency{req, lock})`, not a sorted-set sha256. |
| helm/ignorerules | false | `!` sets `p.negate`; `**` is a parse error. |
| helm/kindsorter | false | hooks sort by kind only (weight is parsed, not a key); SortByName has no revision tiebreak. Unknown-hook drop and `_` partials are missing. |
| helm/memorydriver | false | List/Query return every matching record, not newest-per-name. SetNamespace isolation missing. |
| helm/repindex | false | always `filepath.Split` then join; Add appends; empty index errors; load duplicates error. |
| helm/storage | false | List is every revision; prune errors fail Create. Duplicate Create error missing. |
| helm/strvalsparser | missing | negative list indexes error. |
| kops/clustervalid | missing | empty InstanceGroup list must error. |
| kops/templater | missing | missing include snippet must error (panic recovered). Prose says panics are recovered, not that a missing include panics. |
| gin/bodydecoders | false | BSON `BindBody` is `bson.Unmarshal` only; no `validate`. |
| gin/htmlrender | missing | pre-set Content-Type left unchanged (jsonrenders/streamrenders state this; htmlrender does not). |
| gin/streamrenders | false | gold panics outside 300-308 except 201; contract says all 3xx. Hidden tests skip 309-399, so this did not block the L2 flip. **This unit is in the L2-lie set.** The lie is Content-Length, not redirect. If Composer fails redirect instead of `TestDataContentLengthProperty`, that is the latent defect, not the lie. |
| goa/httpxray | false | recorded URL is `scheme://host/path` (query stripped). |
| goa/xrayseg | false | child `Name` is the argument, not copied from parent. `AddAnnotation` unstated. |
| client-go/lockresolver | missing | `ExtractLockFromKeyErr` (excised, tested, never mentioned). |
| client-go/memdbstaging | missing | `Set(nil)` / `Set(empty)` must error. |
| client-go/onregionerror | missing | RegionNotFound/KeyNotInRegion terminal; StoreNotMatch `CloseAddr`. |
| client-go/pdoracle | missing | invalid `prevSecond` errors. |
| client-go/replicaselector | missing | `WithMatchLabels` first-hop. |

Per-repo JSON: `outputs/contract_gold_dump/{helm,kops,gin,goa,client-go}_verdict.json`. Rollup: `outputs/contract_gold_dump/bank_summary.json`. Dumps: `outputs/contract_gold_dump/<repo>-<unit>.md`.

Reproduce the dump (not the judgement pass):

```
uv run python scripts/audit_contract_gold.py --out outputs/contract_gold_dump
```
