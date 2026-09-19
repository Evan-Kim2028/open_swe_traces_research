# Closure — streamrenders

Package: render.

Files: render/reader.go + render/data.go + render/redirect.go + render/text.go (~150 lines, ~10 funcs).

Removed functions (bodies stubbed): Reader.Render/WriteContentType/writeHeaders, Data.Render/WriteContentType, Redirect.Render/WriteContentType, String.Render/WriteContentType, WriteString.

Exported entry point(s): Reader{ContentType,ContentLength,Reader,Headers}, Data{ContentType,Data}, Redirect{Code,Request,Location}, String{Format,Data}; WriteString.
