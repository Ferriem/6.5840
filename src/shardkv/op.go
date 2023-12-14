package shardkv

import "6.5840/shardctrler"

type Op struct {
	ShardValid bool
	Cmd        interface{}
}

const (
	Get    = "Get"
	Put    = "Put"
	Append = "Append"
)

const (
	Not          = "Not"
	Pull         = "Pull"
	PUT          = 1
	APPEND       = 2
	GET          = 3
	PULL         = 4
	MigrateShard = 5
	RemoveShard  = 6
	Clearshard   = 7
	UpdateConfig = 8
)

type KV struct {
	DB map[int]map[string]string // shardid -> key -> value
}

type KVOp struct {
	Key      string
	Value    string
	Command  int
	OpId     int
	ClientId int64
}

type ShardOp struct {
	Command               int
	DB                    map[string]string
	MaxAppliedOpIdofClerk map[int64]int
	Servers               []string
	ShardId               int
	Num                   int
	Config                shardctrler.Config
}

func (kv *KV) Get(key string) (string, Err) {
	value, ok := kv.DB[key2shard(key)][key]
	if ok {
		return value, OK
	}
	return "", ErrNoKey
}

func (kv *KV) Put(key string, value string) Err {
	if len(kv.DB[key2shard(key)]) == 0 {
		kv.DB[key2shard(key)] = make(map[string]string)
	}
	kv.DB[key2shard(key)][key] = value
	return OK
}

func (kv *KV) Append(key string, value string) Err {
	kv.DB[key2shard(key)][key] += value
	DPrintf("After Append %v", kv.DB[key2shard(key)][key])
	return OK
}

func (kv *KV) Migrate(ShardId int, shard map[string]string) Err {
	delete(kv.DB, ShardId)
	kv.DB[ShardId] = make(map[string]string)
	for k, v := range shard {
		kv.DB[ShardId][k] = v
	}
	return OK
}

func (kv *KV) Copy(ShardId int) map[string]string {
	res := make(map[string]string)
	for k, v := range kv.DB[ShardId] {
		res[k] = v
	}
	return res
}

func (kv *KV) Remove(ShardId int) Err {
	delete(kv.DB, ShardId)
	return OK
}
