# Closure — bodydecoders

Package: binding.

Files: binding/{json,xml,yaml,toml,protobuf,msgpack,bson,plain}.go (~300 lines, ~30 funcs across 8 files).

Removed functions (bodies stubbed): all Binder Name/Bind/BindBody methods and decodeJSON/decodeXML/decodeYAML/decodeToml/decodeMsgPack/decodePlain.

Exported entry point(s): JSON, XML, YAML, TOML, ProtoBuf, MsgPack, BSON, Plain binder singletons; EnableDecoderUseNumber / EnableDecoderDisallowUnknownFields globals.
