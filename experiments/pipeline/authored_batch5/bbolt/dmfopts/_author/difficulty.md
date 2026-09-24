# Difficulty — dmfopts

predicted_flip: L1
details: 9

Missed edges: option independence (each FeatOpt mutates exactly one
featCfg field); validateFSType accepts the declared set only, including
rejecting FSType(""); DevicePath/Filesystem are pure accessors on an
unexported struct; createEmptyFSImage validates before any filesystem
touch and refuses to overwrite.

Hardness driver: six small pure functions whose behavior is fully
deterministic but whose receivers (`featCfg`, `flakey`) are unexported —
the shortcut failure mode is assuming the options write shared state or
that validation happens after image creation, not before.
