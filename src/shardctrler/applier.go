package shardctrler

func (sc *ShardCtrler) applier() {
	for !sc.killed() {
		for ch := range sc.applyCh {
			if ch.CommandValid {
				sc.mu.Lock()
				opchan := sc.getCh(ch.CommandIndex)
				op := ch.Command.(Op)
				res := result{
					ClerkId: op.ClerkId,
					OpId:    op.OpId,
				}
				if !sc.isApplied(&op) {
					switch op.OpType {
					case Join:
						sc.configs = append(sc.configs, *sc.handleJoin(op.Servers))
					case Leave:
						sc.configs = append(sc.configs, *sc.handleLeave(op.GIDs))
					case Move:
						sc.configs = append(sc.configs, *sc.handleMove(op.GID, op.Shard))
					case Query:
						res.config = sc.handleQuery(op.Num)
					}
					sc.maxAppliedOpIdofClerk[op.ClerkId] = op.OpId
					sc.mu.Unlock()
					opchan <- res
					DPrintf("Apply Success: %v, %v", op.OpType, ch.CommandIndex)
				} else {
					sc.mu.Unlock()
				}
			}
		}
	}
}

func (sc *ShardCtrler) isApplied(op *Op) bool {
	maxAppiedOpId, ok := sc.maxAppliedOpIdofClerk[op.ClerkId]
	return ok && maxAppiedOpId >= op.OpId
}
