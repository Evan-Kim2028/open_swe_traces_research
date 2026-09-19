# Why hard — xrayseg

predicted_flip: L4

Mutex-guarded tree with Once-flushed in-progress submit, parent cause-id walk, pkg/errors stack extraction, millisecond timestamps, and a single UDP Write of header+JSON. Only two in-tree tests, so the contract's untested edges (exception cause, annotation maps, Capture) are easy to skip; stubs help more than prose.
