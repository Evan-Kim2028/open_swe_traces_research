# Contract (L2) — storage

The storage layer wraps a backend driver and enforces release bookkeeping. Create persists a release under a key derived from its name and version; Get/Delete operate on that (name, version) pair. History(name) returns all stored revisions of a name; Last returns the newest. ListReleases returns the newest revision of every release; ListDeployed and ListUninstalled filter those by release status. Deployed(name) returns the newest deployed (or pending-install) revision of a name and errors when none exists; DeployedAll returns every revision whose status counts as deployed. After Create/Update, revisions beyond the history limit are pruned, oldest first — but a revision whose status is deployed (or otherwise protected) is never the one removed; pruning failures must not fail the write. Releases may be stored driver-encoded; corrupted entries are skipped by listing operations rather than aborting them. The storage object exposes the driver's logger.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestStorageCreate` | create persists under computed key |
| `TestStorageUpdate` | update persists new revision |
| `TestStorageDelete` | delete removes version |
| `TestStorageList` | list newest per name |
| `TestStorageDeployed` | deployed lookup or error |
| `TestStorageDeployedWithCorruption` | corrupt entries skipped |
| `TestStorageHistory` | all revisions of a name |
| `TestMaxHistoryErrorHandling` | prune error does not fail write |
| `TestStorageRemoveLeastRecent` | oldest pruned past limit |
| `TestStorageDoNotDeleteDeployed` | deployed never pruned |
| `TestStorageLast` | newest revision returned |
| `TestUpgradeInitiallyFailedReleaseWithHistoryLimit` | failed release prunes correctly |
| `TestStorageGetsLoggerFromDriver` | logger plumbed from driver |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/storage/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestStorageContractTableProperty`, `TestStorageHistoryProperty`, `TestStorageMaxHistoryProperty`, `TestStorageDeployedFiltersProperty`, `TestStorageAdversarialProperty`, `TestStorageUnseenRandomProperty`, `TestStoragePruneFailureProperty`: TestStorageContractTableProperty

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
