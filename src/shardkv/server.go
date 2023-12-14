package shardkv

import (
	"encoding/gob"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
	"6.5840/shardctrler"
)

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	make_end     func(string) *labrpc.ClientEnd
	gid          int
	ctrlers      []*labrpc.ClientEnd
	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	dead                  int32
	maxAppliedOpIdofClerk map[int64]int
	IndexToCommand        map[int]chan KVOp
	LastApplied           int
	StateMachine          KVStateMachine

	sc            *shardctrler.Clerk
	LastConfig    shardctrler.Config
	PreConfig     shardctrler.Config
	ShardState    map[int]string
	ShardToClient map[int][]int64
	ShardNum      map[int]int
	OutedData     map[int]map[int]map[string]string // num -> shard -> key -> value
	Pullchan      map[int]chan PullReply
}

func (kv *ShardKV) GetChan(index int) chan KVOp {
	ch, ok := kv.IndexToCommand[index]
	if !ok {
		ch = make(chan KVOp, 1)
		kv.IndexToCommand[index] = ch
	}
	return ch
}

func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	DPrintf("Get: %v", args)
	// Your code here.
	if kv.killed() {
		reply.Err = ErrWrongLeader
		return
	}
	isLeader := kv.isLeader()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	kv.mu.Lock()
	if !kv.CheckGroup(args.Key) || kv.ShardState[key2shard(args.Key)] != OK {
		reply.Err = ErrWrongGroup
		kv.mu.Unlock()
		return
	}

	op := KVOp{
		Key:      args.Key,
		Command:  GET,
		OpId:     args.OpId,
		ClientId: args.ClientId,
	}

	index, _, isLeader := kv.rf.Start(Op{ShardValid: false, Cmd: op})
	//DPrintf("Start Get, index %v", index)
	if !isLeader {
		reply.Err = ErrWrongLeader
		kv.mu.Unlock()
		return
	}

	ch := kv.GetChan(index)
	kv.mu.Unlock()

	select {
	case result := <-ch:
		if result.ClientId == op.ClientId && result.OpId == op.OpId {
			reply.Value = result.Value
			//DPrintf("Get %v success, value %v", op.Key, reply.Value)
			if reply.Value != "" {
				reply.Err = OK
			} else {
				reply.Err = ErrNoKey
			}
		} else {
			reply.Err = ErrWrongLeader
		}
	case <-time.After(time.Millisecond * 200):
		//DPrintf("Get Timeout")
		reply.Err = ErrWrongLeader
	}
	go func() {
		kv.mu.Lock()
		delete(kv.IndexToCommand, index)
		kv.mu.Unlock()
	}()

}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	DPrintf("PutAppend: %v", args)
	if kv.killed() {
		reply.Err = ErrWrongLeader
		return
	}
	isLeader := kv.isLeader()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	kv.mu.Lock()
	if !kv.CheckGroup(args.Key) || kv.ShardState[key2shard(args.Key)] != OK {
		reply.Err = ErrWrongGroup
		kv.mu.Unlock()
		return
	}
	if kv.maxAppliedOpIdofClerk[args.ClientId] >= args.OpId {
		reply.Err = OK
		kv.mu.Unlock()
		return
	}
	intcmd := 0
	switch args.Op {
	case Put:
		intcmd = PUT
	case Append:
		intcmd = APPEND
	}
	op := KVOp{
		Key:      args.Key,
		Value:    args.Value,
		Command:  intcmd,
		OpId:     args.OpId,
		ClientId: args.ClientId,
	}
	index, _, isLeader := kv.rf.Start(Op{ShardValid: false, Cmd: op})
	if !isLeader {
		reply.Err = ErrWrongLeader
		kv.mu.Unlock()
		return
	}
	ch := kv.GetChan(index)
	kv.mu.Unlock()
	select {
	case result := <-ch:
		if result.ClientId == op.ClientId && result.OpId == op.OpId {
			//DPrintf("PutAppend success")
			reply.Err = OK
		} else {
			reply.Err = ErrWrongLeader
		}
	case <-time.After(time.Millisecond * 200):
		//DPrintf("PutAppend Timeout")
		reply.Err = ErrWrongLeader
	}

	go func() {
		kv.mu.Lock()
		delete(kv.IndexToCommand, index)
		kv.mu.Unlock()
	}()
}

// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
func (kv *ShardKV) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctrler.
//
// pass ctrlers[] to shardctrler.MakeClerk() so you can send
// RPCs to the shardctrler.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, gid int, ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.make_end = make_end
	kv.gid = gid
	kv.ctrlers = ctrlers
	kv.sc = shardctrler.MakeClerk(ctrlers)

	// Your initialization code here.

	// Use something like this to talk to the shardctrler:
	// kv.mck = shardctrler.MakeClerk(kv.ctrlers)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.maxAppliedOpIdofClerk = make(map[int64]int)
	kv.IndexToCommand = make(map[int]chan KVOp)
	kv.LastApplied = 0
	kv.StateMachine = &KV{make(map[int]map[string]string)}
	kv.OutedData = make(map[int]map[int]map[string]string)
	kv.ShardState = make(map[int]string)
	kv.LastConfig = shardctrler.Config{Num: 0, Groups: map[int][]string{}}
	kv.PreConfig = shardctrler.Config{Num: 0, Groups: map[int][]string{}}
	kv.ShardToClient = make(map[int][]int64)
	kv.ShardNum = make(map[int]int)
	kv.Pullchan = make(map[int]chan PullReply)
	snapshot := persister.ReadSnapshot()
	if len(snapshot) > 0 {
		kv.DecodeSnapShot(snapshot)
	}
	go kv.applier()
	go kv.UpdateConfig()
	gob.Register(KVOp{})
	gob.Register(ShardOp{})
	return kv
}

func (kv *ShardKV) CheckGroup(key string) bool {
	return kv.PreConfig.Shards[key2shard(key)] == kv.gid
}
