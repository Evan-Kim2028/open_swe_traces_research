# Why hard — bodydecoders

predicted_flip: L3
a multi-file decode->validate sequence contract plus two mutable global decoder flags (state that changes behavior across binds), the protobuf no-validate exception, and the plain binder's pointer-chasing string/[]byte special case; a cheat that 'decodes' by assignment or skips validation passes casual checks but fails the suite.
