# Closure — createfork

Package: github (root). File: github/repos_forks.go.
Removed bodies: RepositoriesService.CreateFork — stubbed to a variant
that returns the `*AcceptedError` as-is on a 202 response, dropping the
unmarshal of `AcceptedError.Raw` into the `fork` result so the deferred
fork's metadata never reaches the caller. The request build, opts
application, and Do call keep working; only the accepted-error payload
decode is excised.
Kept: Repository fields, ListForks, AcceptedError type.
Tests removed: 2 funcs in github/repos_forks_test.go
(TestRepositoriesService_CreateFork_deferred,
TestRepositoriesService_CreateFork_deferred_badBody).
