# Exported API — bindingdispatch

Default(method, contentType) Binding; the MIME* constants; binder singletons; Validator var.

## Pre-existing callers

engine request dispatch (Bind picks binder by method+content-type); every binder's Bind calls validate().
