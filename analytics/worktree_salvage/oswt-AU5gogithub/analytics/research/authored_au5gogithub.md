# authored_au5gogithub — 20 new go-github units

Branch `au5gogithub`, source commit `d6ef767c`. Units under
`experiments/pipeline/authored_au5gogithub/go-github/<unit>/_author/`,
verified by `_tools/verify4.sh` (excised tree green, gold passes, cheat
fails). Overlap screened with `scripts/check_unit_overlap.py --new
experiments/pipeline/authored_au5gogithub` (20 new vs 21 existing in
authored+authored_batch3: **0 overlaps**) plus a cross-worktree pass
against batch4 units in `oswt-AU4gogithub2` (29), `oswt-RCgogithub4`
(28), `oswt-VFbatch4gogithub` (29): **0 overlaps**. Log:
`outputs/AU5gogithub.log`.

**Rejections before acceptance: 0 for every unit.** The candidate
queue was the 20-spec list left by the prior session; all 20 passed the
symbol screen, so no candidate was rejected for overlap and there are
no collisions to list. The real per-unit cost was spec repair
(orphaned imports, missing deltests, oversized cheats) — noted per unit
as `repair:` rounds.

## Units (authoring order)

1. **newreq** — `Client.NewRequest`: API-version header + omit-empty
   User-Agent. Surface: request builder (header policy). repair: 0
   (pre-materialized). Inferable: doc 1, partially 1, yes 1. cheat 0.17.
2. **dounit** — `Client.Do`: whitespace-only body tolerance + pooled
   decode. Surface: response decode policy. repair: 0. Inferable:
   doc 1, partially 1, yes 3. cheat 0.57.
3. **createcommit** — `GitService.CreateCommit`: opts-nil default,
   Verification.Signature vs Signer precedence. Surface: request
   serialization. repair: 0. Inferable: yes 2, partially 2. cheat 0.44.
4. **pullmerge** — `PullRequestsService.Merge`: DontDefaultIfBlank ×
   empty commit_message. Surface: request serialization. repair: 1
   (cheat slimmed from 0.60 to 1 line). Inferable: partially 1, no 1,
   yes 1. cheat 0.20.
5. **uploadreq** — `Client.NewUploadRequest`: `..` traversal,
   ErrUntrustedDestination, reader wrapping, GetBody. Surface: request
   validation/builder. repair: 0. Inferable: partially 4, no 1, yes 1,
   doc 1. cheat 0.16.
6. **newformreq** — `Client.NewFormRequest`: body-destination gate,
   API-version header, omit-empty User-Agent. Surface: request
   validation/builder. repair: 0. Inferable: doc 2, yes 2, partially 1.
   cheat 0.54.
7. **authtransports** — `UnauthenticatedRateLimitedTransport` /
   `BasicAuthTransport.RoundTrip`: AllowedOrigins credential gating.
   Surface: credential-scope predicate. repair: 1 (cheat slimmed 0.73→
   0.36). Inferable: doc 2, partially 2. cheat 0.36.
8. **baredo** — `Client.bareDo`: rate-limit gate/writeback, secondary
   retry, ctx-cancel mapping, URL sanitize, AcceptedError.Raw, body
   close. Surface: egress orchestration. repair: 2 (extra deltests for
   AcceptedError.Raw consumers). Inferable: doc 2, partially 3, no 1,
   yes 1. cheat 0.13.
9. **clientclone** — `Client.Clone`: config carry-over, transport
   re-scope for token clones, shared rate map. Surface: config cloning.
   repair: 2 (one more deltest). Inferable: doc 1, yes 1, no 1,
   partially 3. cheat 0.30.
10. **createfork** — `RepositoriesService.CreateFork`: decode
    AcceptedError.Raw into fork. Surface: response decode. repair: 2
    (test-file unimport). Inferable: partially 2, yes 1. cheat 0.25.
11. **downloadasset** — `DownloadReleaseAsset` +
    `downloadReleaseAssetFromURL`: follow redirects via separate client,
    validate, close original body. Surface: redirect orchestration.
    repair: 1 (unimport). Inferable: yes 2, doc 1, partially 2.
    cheat 0.36.
12. **downloadcontents** — `DownloadContentsWithMeta`: fetch
    download_url through credential-scoped client. Surface: download
    plumbing. repair: 1. Inferable: doc 1, partially 3, no 1, yes 1.
    cheat 0.21.
13. **fetchsbom** — `FetchSBOM` + `fetchSBOMFromURL`: pre-signed
    redirect fetch, no credentials, validate+decode. Surface: redirect
    orchestration. repair: 2 + cheat slim (0.72→0.22). Inferable: doc 1,
    yes 2, partially 2. cheat 0.23.
14. **getcontents** — `GetContents`: path escape/trim, file-or-directory
    two-pass decode, combined error. Surface: response decode + path
    escaping. repair: 1. Inferable: yes 2, partially 1, no 1.
    cheat 0.12.
15. **gitrefs** — `CreateRef`/`UpdateRef`/`DeleteRef`: required-field
    guards, `refs/` normalize, path escape. Surface: input validation.
    repair: 1. Inferable: partially 4, yes 1. cheat 0.54.
16. **markdown** — `MarkdownService.Render`: optional Mode/Context
    fields. Surface: request serialization. repair: 1 (whole test file
    → delfiles). Inferable: doc 2, partially 1, yes 1. cheat 0.30.
17. **ratelimitget** — `RateLimitService.Get`: BypassRateLimitCheck +
    per-category writeback into client.rateLimits. Surface: response
    decode + state writeback. repair: 1. Inferable: doc 1, partially 2,
    yes 1. cheat 0.23.
18. **searchq** — internal `search`: per-type preview Accept +
    TextMatch media type + repository_id. Surface: header
    serialization. repair: 2 (impl-file unimports). Inferable: doc 1,
    no 1, partially 2. cheat 0.27.
19. **statsreshape** — `ListCodeFrequency`/`ListPunchCard`: `[][]int` →
    `*WeeklyStats`/`*PunchCard`. Surface: pure reshape (purest unit in
    the batch). repair: 2 + cheat slim (0.68→0.45). Inferable:
    partially 3, yes 1. cheat 0.48.
20. **uploadasset** — `UploadReleaseAsset`/`FromRelease`: dir/nil/
    negative guards, `{?name,label}` strip, media-type inference.
    Surface: input validation + media-type inference. repair: 2 (extra
    deltests). Inferable: partially 3, doc 1, no 1. cheat 0.04.

## Diminishing returns

Search cost per accepted unit was **not rising**. All 20 specced
candidates passed the overlap screen on the first check (0 rejections,
0 collisions); marginal candidate-search cost was ~0 because the queue
was pre-vetted. The dominant cost was spec repair, and it stayed flat:
14/14 drafts failed on orphaned imports at first probe, all were fixed
in ≤2 probe rounds each, and 4 units needed one cheat-slimming pass
(authtransports, pullmerge, fetchsbom, statsreshape). Cost was bounded
rework, not a shrinking candidate pool — github.go's request/response
plumbing and the service files still hold un-excised closures, so the
bank is not exhausted. Stopping at 20 was the requested count, not a
yield signal.
