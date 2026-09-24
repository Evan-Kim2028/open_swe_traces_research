# Exported API — createfork

`RepositoriesService.CreateFork(ctx, owner, repo, opts)` POSTs a fork.
GitHub answers `202 Accepted` while the fork is created asynchronously;
the response body still carries the pending repository's JSON. Callers
rely on:

- `*AcceptedError` returned for the 202.
- The repository payload inside that response being decoded into the
  returned `*Repository`, so callers get the deferred fork's metadata
  alongside the error.
