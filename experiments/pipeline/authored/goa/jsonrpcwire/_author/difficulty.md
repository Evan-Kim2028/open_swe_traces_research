# Why hard — jsonrpcwire

predicted_flip: L3

Wire-level identity: empty-string vs missing vs null ids, json.Number vs float64, HasID/HasMethod/HasResult flags, and Validate's narrow null-id exception (only −32700 and −32600). The full contract names every rule the in-tree tests encode; a mid-tier model can implement from L2 but will miss a flag or the null-id table without it.
