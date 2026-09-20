# Exported API — pagevalues

`Response` is returned by every API call and carries pagination state read off
the HTTP `Link` header. Iterators and hand-written pagination loops depend on
these fields.

- `Response.NextPage`, `PrevPage`, `FirstPage`, `LastPage` — integer page
  numbers parsed from `page=` link params.
- `Response.NextPageToken` — non-numeric page token when `page=` is not an int.
- `Response.Cursor` — cursor value from `cursor=` links (cursor pagination).
- `Response.Before`, `Response.After` — opaque cursors from `before=`/`after=`
  link params.
- `populatePageValues` (excised body) is invoked by `newResponse` for every
  response.
