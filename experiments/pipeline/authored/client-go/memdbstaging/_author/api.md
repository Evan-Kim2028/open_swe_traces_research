# Exported API — memdbstaging

MemDB.Staging()->handle; Release(h); Cleanup(h); RevertToCheckpoint(cp); Checkpoint(); InspectStage(h,f); Get/Set/SetWithFlags/Delete/DeleteWithFlags/UpdateFlags; Len/Size/Dirty

## Pre-existing callers

unionstore.UnionStore mem-buffer, txn staging buffers, pipelined flush staging.
