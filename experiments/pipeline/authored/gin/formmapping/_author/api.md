# Exported API — formmapping

MapFormWithTag; BindUnmarshaler interface; ErrConvertMapStringSlice/ErrConvertToMapString; reached through Form/FormPost/FormMultipart/Query/Header/Uri binders.

## Pre-existing callers

formBinding.Bind, formPostBinding.Bind, formMultipartBinding.Bind (via multipartRequest.TrySet->setByForm), queryBinding.Bind, headerBinding.Bind (mapHeader->mappingByPtr), uriBinding.BindUri (mapURI); also engine ShouldBind* paths.
