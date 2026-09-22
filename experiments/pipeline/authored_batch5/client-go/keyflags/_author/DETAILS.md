# Details — keyflags

1. `KeyFlags` is a `uint16`; the top bit is reserved for the red-black tree
   and is never set by any flag op. Inferable: doc — stated on the type.
2. The assertion pair is two bits with four states: neither set = unsettable;
   `Exist` only = assert-exists; `NotExist` only = assert-absent; both =
   "unknown" (assertion frozen). `HasAssertExist`/`HasAssertNotExist` are
   true ONLY in the exclusive states; `HasAssertUnknown` needs both;
   `HasAssertionFlags` needs either. Inferable: doc — the truth table is in
   the flag comments.
3. `HasPresumeKeyNotExists` is true when EITHER `flagPresumeKNE` or
   `flagPreviousPresumeKNE` is set — the "previous" bit is read together with
   the live one. Inferable: partially — the coupling is documented on the
   flag, not the predicate.
4. `ApplyFlagsOps` applies ops left to right; `SetPresumeKeyNotExists` also
   sets `flagNeedCheckExists`, and `DelPresumeKeyNotExists` clears both.
   Inferable: partially — the implied flag is documented on the op.
5. `SetKeyLockedValueExists` sets `flagKeyLockedValExist` AND clears
   `flagNeedConstraintCheckInPrewrite`; `SetKeyLockedValueNotExists` clears
   the value bit AND the constraint bit. Inferable: no — the cross-flag clear
   is internal.
6. `SetAssertExist`/`SetAssertNotExist` clear the opposite bit before setting
   theirs; `SetAssertUnknown` sets both; `SetAssertNone` clears both.
   Inferable: partially.
7. `AndPersistent` masks to `flagKeyLocked | flagKeyLockedValExist |
   flagNeedConstraintCheckInPrewrite` — all other bits drop. Inferable:
   partially — the persistent set is a named constant but its membership is a
   choice.
8. Unknown ops leave the value unchanged (the switch has no default action).
   Inferable: no.
9. Predicates observe the raw bit: `HasLocked`/`HasNeedLocked`/
   `HasLockedValueExists`/`HasNeedCheckExists`/`HasPrewriteOnly`/
   `HasIgnoredIn2PC`/`HasReadable`/`HasNeedConstraintCheckInPrewrite`/
   `HasNewlyInserted` are single-bit tests. Inferable: yes.
