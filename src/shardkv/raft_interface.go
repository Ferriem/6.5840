package shardkv

func (kv *ShardKV) isLeader() bool {
	_, isleader := kv.rf.GetState()
	return isleader
}
