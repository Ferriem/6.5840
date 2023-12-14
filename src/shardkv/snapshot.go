package shardkv

import (
	"bytes"

	"6.5840/labgob"
	"6.5840/shardctrler"
)

func (kv *ShardKV) DecodeSnapShot(snapshot []byte) {
	if snapshot == nil || len(snapshot) < 1 {
		return
	}
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)

	var db KV
	var maxAppliedOpIdofClerk map[int64]int
	var preconfig shardctrler.Config
	var lastconfig shardctrler.Config
	var ShardState map[int]string
	var ShardToClient map[int][]int64
	var ShardNum map[int]int
	var OutedData map[int]map[int]map[string]string

	if d.Decode(&db) != nil || d.Decode(&maxAppliedOpIdofClerk) != nil || d.Decode(&preconfig) != nil || d.Decode(&lastconfig) != nil || d.Decode(&ShardState) != nil || d.Decode(&ShardToClient) != nil || d.Decode(&ShardNum) != nil || d.Decode(&OutedData) != nil {
	} else {
		kv.StateMachine = &db
		kv.maxAppliedOpIdofClerk = maxAppliedOpIdofClerk
		kv.PreConfig = preconfig
		kv.LastConfig = lastconfig
		kv.ShardState = ShardState
		kv.ShardToClient = ShardToClient
		kv.ShardNum = ShardNum
		kv.OutedData = OutedData
	}
}

func (kv *ShardKV) PersistSnapShot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(kv.StateMachine)
	e.Encode(kv.maxAppliedOpIdofClerk)
	e.Encode(kv.PreConfig)
	e.Encode(kv.LastConfig)
	e.Encode(kv.ShardState)
	e.Encode(kv.ShardToClient)
	e.Encode(kv.ShardNum)
	e.Encode(kv.OutedData)
	snapshot := w.Bytes()
	return snapshot

}
