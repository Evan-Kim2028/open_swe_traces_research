# Details — mvccread

1. `mvccLock.check` passes through when `l.startTS > ts` (lock is newer
   than the read) OR `op` is `Op_Lock`/`Op_PessimisticLock` (pessimistic
   locks don't block reads). Inferable: partially — the exempt op set is a
   choice.
2. Short-circuit: `ts == math.MaxUint64` AND `l.primary` equals the raw key
   returns `l.startTS - 1` (read just below the lock's own version — the
   lock's writer reading its own key). Inferable: no.
3. A lock whose `startTS` appears in `resolvedLocks` is skipped;
   otherwise `check` returns `(0, ErrLocked)` built by `lockErr` with the
   key mvcc-encoded at `lockVer`. Inferable: partially.
4. `mvccEntry.Get` runs the lock check only at `IsolationLevel_SI`; then
   returns the first version with `commitTS <= ts` skipping `typeRollback`
   and `typeLock` values; no match -> `(nil, nil)`. Inferable: partially —
   rollback/lock skipping is the subtle part.
5. `regionContains` is `[startKey, endKey)` with `startKey <= key` and
   (`key < endKey` OR `endKey` empty — unbounded end). Inferable: doc.
6. `mvccEntry.Less` is `bytes.Compare(e.key, than.key) < 0` on the
   ENCODED keys. Inferable: yes.
