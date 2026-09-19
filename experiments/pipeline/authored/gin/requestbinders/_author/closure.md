# Closure — requestbinders

Package: binding.

Files: binding/form.go + binding/query.go + binding/header.go + binding/uri.go (177 lines, ~14 funcs).

Removed functions (bodies stubbed): formBinding.Name/Bind, formPostBinding.Name/Bind, formMultipartBinding.Name/Bind, queryBinding.Name/Bind, headerBinding.Name/Bind, mapHeader, headerSource.TrySet, uriBinding.Name/BindUri.

Exported entry point(s): Form, FormPost, FormMultipart, Query, Header, Uri binder singletons (Binding/BindingUri interfaces).
