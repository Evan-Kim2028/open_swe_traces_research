# Why hard — evalrun

predicted_flip: L4

Four ordered passes (execute, including expressions appended mid-walk; then prepare; then validate; then finalize), two different 100-iteration caps (generated expressions vs generated roots), stack push/pop around every source function, and error recording that must not drop later-pass failures. A mid-tier model given only the bug report will stub RunDSL; the full contract still leaves the walk-set mutation and root-registration loop easy to get wrong until signatures/stubs are in view.
