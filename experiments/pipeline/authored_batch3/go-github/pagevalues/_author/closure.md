# Closure — pagevalues

Package: github (root). File: github/github.go.
Removed bodies: Response.populatePageValues — stubbed to a page/before/after
parser (drops the cursor= → Response.Cursor branch and the since→page
fallback). Keeps page-, token-, before-, after-, and first/prev/last-based
pagination working so agent_tasks/examples/projects cursor fixtures and all
page-based iterators stay green; cursor= link iterators and since-link tests
are excised.
Kept: newResponse, parseRate, parseTokenExpiration, all Response fields.
Tests removed: 7 funcs in github/github_test.go (TestResponse_populate*,
TestResponse_SinceWithPage, TestResponse_cursorPagination,
TestResponse_beforeAfterPagination) plus 3 funcs in
github/github-iterators_test.go that paginate on `?cursor=yo` links
(TestAppsService/ListHookDeliveriesIter,
TestOrganizationsService_ListHookDeliveriesIter,
TestRepositoriesService_ListHookDeliveriesIter).
