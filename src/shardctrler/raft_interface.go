package shardctrler

func (sc *ShardCtrler) isLeader() bool {
	_, isleader := sc.rf.GetState()
	return isleader
}
