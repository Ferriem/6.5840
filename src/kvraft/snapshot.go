package kvraft

import (
	"bytes"

	"6.5840/labgob"
)

func (kv *KVServer) approachLimit() bool {
	return kv.persister.RaftStateSize() > kv.maxraftstate
}

func (kv *KVServer) makeSnapShot() []byte {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(kv.db)
	e.Encode(kv.maxAppliedOpIdofClerk)
	data := w.Bytes()
	return data
}

func (kv *KVServer) readSnapShot(snapshot []byte) {
	r := bytes.NewBuffer(snapshot)
	d := labgob.NewDecoder(r)
	d.Decode(&kv.db)
	d.Decode(&kv.maxAppliedOpIdofClerk)
}

func (kv *KVServer) checkpoint(index int) {
	snapshot := kv.makeSnapShot()
	kv.rf.Snapshot(index, snapshot)
}
