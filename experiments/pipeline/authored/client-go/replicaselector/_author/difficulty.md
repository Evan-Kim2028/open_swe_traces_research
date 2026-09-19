# Why hard — replicaselector

pure state machine with ~6 states (leader/follower attempts/proxy/exhausted/invalid), liveness gating via slow scores, ordering by load, and multi-input transitions driven by error kinds. Correctness is entirely in the transition table; there is no algorithmic crux to copy.
