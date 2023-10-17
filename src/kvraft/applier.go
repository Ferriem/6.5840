package kvraft

func (kv *KVServer) executor() {
	for m := range kv.applyCh {
		if kv.killed() {
			break
		}
		kv.mu.Lock()
		if m.SnapshotValid {
			kv.readSnapShot(m.Snapshot)
		} else {
			op := m.Command.(Op)

			kv.maybeApplyClientOp(op)

			if kv.snapshotEnable && kv.approachLimit() {
				kv.checkpoint(m.CommandIndex)
			}
		}
		kv.mu.Unlock()
	}
}

func (kv *KVServer) maybeApplyClientOp(op Op) {
	if !kv.isApplied(op) {
		kv.applyClientOp(op)
		kv.maxAppliedOpIdofClerk[op.ClerkId] = op.OpId
		kv.notify(op)
	}
}

func (kv *KVServer) applyClientOp(op Op) {
	switch op.OpType {
	case "Put":
		kv.db[op.Key] = op.Value
	case "Append":
		kv.db[op.Key] += op.Value
	case "Get":
		// do nothing
	default:
		DPrintf("Wrong OpType: %s", op.OpType)
	}
}

func (kv *KVServer) isApplied(op Op) bool {
	maxAppliedOpId, ok := kv.maxAppliedOpIdofClerk[op.ClerkId]
	return ok && maxAppliedOpId >= op.OpId
}

func (kv *KVServer) Applier(op Op) (Err, string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if !kv.isApplied(op) {
		if !kv.Operation(op) {
			return ErrWrongLeader, ""
		}
		kv.makeNotifier(op)
		kv.wait(op)
	}

	if kv.isApplied(op) {
		value := ""
		if op.OpType == "Get" {
			value = kv.db[op.Key]
		}
		return OK, value
	}
	return ErrNotApplied, ""

}
