# Why hard — evalctx

predicted_flip: L3

Topological sort with an explicit reverse (dependencies last), pairwise cycle detection after flattening each root's DependsOn, and duplicate-name rejection. In-tree tests only go through a full evaluation run, so the cycle path is untested; L2 states it, L3+ is needed if the solver ignores the unused rule. The DFS `seen` marking is easy to implement as a standard topo and get the order backwards.
