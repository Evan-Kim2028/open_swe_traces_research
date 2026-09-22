# Closure — searchq

Package: github (root). File: github/search.go.
Removed bodies: the internal `search` helper — stubbed to send the
request with no Accept-header assembly: drops the `repository_id` query
param, the per-searchType preview-media-type switch (commits, topics/
repositories, issues), and the `opts.TextMatch` text-match media type.
`qs.Values(opts)` encoding and the Do call keep working.
Kept: all SearchService wrappers (Repositories, Code, Issues, etc.),
SearchOptions, media-type constants.
Tests removed: 3 funcs in github/search_test.go
(TestSearchService_RepositoriesTextMatch, _CodeTextMatch, _Labels).
