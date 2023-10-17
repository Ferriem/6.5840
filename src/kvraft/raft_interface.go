package kvraft

func (kv *KVServer) Operation(op Op) bool {
	_, _, isleader := kv.rf.Start(op)
	return isleader
}

func (kv *KVServer) isLeader() bool {
	_, isleader := kv.rf.GetState()
	return isleader
}
