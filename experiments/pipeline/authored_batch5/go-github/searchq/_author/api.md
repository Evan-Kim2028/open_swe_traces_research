# Exported API — searchq

Internal `search(ctx, searchType, parameters, opts, result)` underpins
every `SearchService.<Kind>` method. Its Accept header is assembled per
search type (commits/topics/repositories/issues each pull a preview
media type) plus the text-match media type when `opts.TextMatch` is
set; `parameters.RepositoryID` is flattened to a `repository_id` query
param.
