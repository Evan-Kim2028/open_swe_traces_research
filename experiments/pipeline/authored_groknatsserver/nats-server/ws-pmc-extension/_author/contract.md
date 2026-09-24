# Contract — ws-pmc-extension

Hidden suite: `tests/hidden/server/ws_pmc_extension_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
The extension/parameter names use the package's own visible constants.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | A value stored under the canonical extension header key is found; a differently-named header is not consulted. |
| TestDetail02 | 2 | yes | Comma-separated extensions inside one value and across multiple header values are each scanned; semicolons separate token from parameters. |
| TestDetail03 | 3 | yes | Tokens and parameters padded with spaces and tabs still match and count. |
| TestDetail04 | 4 | yes | `permessage-deflate`, `PerMessage-Deflate`, and `PERMESSAGE-DEFLATE` all match. |
| TestDetail05 | 5 | yes | With the check-P-M-C-only flag set, a match returns `(true, false)` even when both parameters are present. |
| TestDetail06 | 6 | no | Only parameters after the matching token count: the parameter names appearing as earlier extension tokens do not count, parameters attached to a different extension do not count, and the same names after the token do. (Membership-after-token is the derivable shape; scan internals are not pinned.) |
| TestDetail07 | 7 | yes | Both no-context parameter names match when written in other cases. |
| TestDetail08 | 8 | yes | Both parameters present — in either order — yields `(true, true)`. |
| TestDetail09 | 9 | yes | The extension with zero or one of the parameters, or with unrelated parameters, yields `(true, false)`. |
| TestDetail10 | 10 | yes | No header, an empty value, unrelated extensions, a token that merely contains the extension name as a sub/superstring, and a header holding only the parameter names all yield `(false, false)`. |
| TestDetail11 | 11 | partially | A first matching extension lacking the parameters still decides `(true, false)` when a later matching entry has them — across two header values and within one value; and a first match carrying both params decides `(true, true)` ahead of a bare later match. |

Refusals/softening: line 6 is `Inferable: no` — asserted only through which
parameter positions count (after the matching token, same extension), which
is the derivable behaviour; nothing is asserted about iteration internals.
Line 11 is `partially` — asserted as "the first match's own parameters
decide"; no claim is made about how partial scans interleave.
