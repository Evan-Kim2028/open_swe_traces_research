# Why hard — requestbinders

predicted_flip: L3
the contract splits one engine across four input sources with different tag namespaces and value plumbing (header canonicalization, request-as-setter for multipart, PostForm vs Form vs URL query); the shared engine stays intact so the wrong-source bugs are silent.
