# Closure — getcontents

Package: github (root). File: github/repos_contents.go.
Removed bodies: GetContents — stubbed to decode the response body
directly into `*RepositoryContent` and always return it as
`fileContent`, dropping the raw-message two-pass decode (file first,
then directory) and the combined unmarshal error; also drops the path
normalization (`strings.TrimSuffix(path, "/")` + `url.URL{Path}`
escaping) so paths with spaces, `+`, or `..` are sent raw. The request
build and Do call keep working.
Kept: CreateFile/UpdateFile/DeleteFile, RepositoryContent,
RepositoryContentGetOptions.
Tests removed: 10 funcs in github/repos_contents_test.go covering
directory decode, path escaping (spaces, plus chars, parent refs,
trailing slash), and the DownloadContents NotFile/Submodule cases that
flow through GetContents.
