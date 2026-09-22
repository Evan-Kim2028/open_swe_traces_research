# difficulty — hdrsurgery

Medium. Small helpers with sharp edges: getHeaderKeyIndex must retry past
prefix-shadowed keys and anchor at a real CRLF (not just '\n');
sliceHeader's three-index return caps capacity; remove* helpers loop for
duplicates and collapse to nil at emptyHdrLine; jsAckDeliverIdx counts 8
dots before honoring '@'; splitSubjectQueue must collapse whitespace runs
rather than split on a single byte.
