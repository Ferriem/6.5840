package shardctrler

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
)

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.

	configs []Config // indexed by config num

	dead int32 // set by Kill()

	maxAppliedOpIdofClerk map[int64]int
	IndexToCommand        map[int]chan result
}

type result struct {
	ClerkId int64
	OpId    int
	config  Config
}

type GidAndShards struct {
	gid    int
	shards []int
}

type Movement struct {
	from   int   // from group.
	to     int   // to group.
	shards []int // moved shards.
}

func (sc *ShardCtrler) getCh(index int) chan result {
	ch, ok := sc.IndexToCommand[index]
	if !ok {
		ch = make(chan result, 1)
		sc.IndexToCommand[index] = ch
	}
	return ch
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	DPrintf("Join: %v", args)
	if sc.killed() {
		reply.Err = WrongLeader
		return
	}

	isLeader := sc.isLeader()
	if !isLeader {
		reply.Err = WrongLeader
		return
	}

	sc.mu.Lock()

	op := Op{
		ClerkId: args.ClerkId,
		OpId:    args.OpId,
		OpType:  Join,
		Servers: args.Servers,
	}

	if sc.isApplied(&op) {
		reply.Err = OK
		sc.mu.Unlock()
		return
	}

	index, _, isLeader := sc.rf.Start(op)
	DPrintf("Start Join, index: %v", index)
	if !isLeader {
		reply.Err = WrongLeader
		sc.mu.Unlock()
		return
	}

	ch := sc.getCh(index)
	sc.mu.Unlock()

	select {
	case result := <-ch:
		if result.ClerkId == op.ClerkId && result.OpId == op.OpId {
			//DPrintf("Join Success")
			reply.Err = OK
		} else {
			reply.Err = WrongLeader
		}
	case <-time.After(50 * time.Millisecond):
		//DPrintf("Join Timeout")
		reply.Err = WrongLeader
	}
	go func() {
		sc.mu.Lock()
		delete(sc.IndexToCommand, index)
		sc.mu.Unlock()
	}()

}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	if sc.killed() {
		reply.Err = WrongLeader
		return
	}

	isLeader := sc.isLeader()
	if !isLeader {
		reply.Err = WrongLeader
		return
	}

	sc.mu.Lock()

	op := Op{
		ClerkId: args.ClerkId,
		OpId:    args.OpId,
		OpType:  Leave,
		GIDs:    args.GIDs,
	}

	if sc.isApplied(&op) {
		reply.Err = OK
		sc.mu.Unlock()
		return
	}

	index, _, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.Err = WrongLeader
		sc.mu.Unlock()
		return
	}

	ch := sc.getCh(index)
	sc.mu.Unlock()

	select {
	case result := <-ch:
		if result.ClerkId == args.ClerkId && result.OpId == args.OpId {
			reply.Err = OK
		} else {
			reply.Err = WrongLeader
		}
	case <-time.After(50 * time.Millisecond):
		reply.Err = WrongLeader
	}
	go func() {
		sc.mu.Lock()
		delete(sc.IndexToCommand, index)
		sc.mu.Unlock()
	}()
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	if sc.killed() {
		reply.Err = WrongLeader
		return
	}

	isLeader := sc.isLeader()
	if !isLeader {
		reply.Err = WrongLeader
		return
	}

	sc.mu.Lock()

	op := Op{
		ClerkId: args.ClerkId,
		OpId:    args.OpId,
		OpType:  Move,
		Shard:   args.Shard,
		GID:     args.GID,
	}

	if sc.isApplied(&op) {
		reply.Err = OK
		sc.mu.Unlock()
		return
	}

	index, _, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.Err = WrongLeader
		sc.mu.Unlock()
		return
	}

	ch := sc.getCh(index)
	sc.mu.Unlock()

	select {
	case result := <-ch:
		if result.ClerkId == args.ClerkId && result.OpId == args.OpId {
			reply.Err = OK
		} else {
			reply.Err = WrongLeader
		}
	case <-time.After(50 * time.Millisecond):
		reply.Err = WrongLeader
	}
	go func() {
		sc.mu.Lock()
		delete(sc.IndexToCommand, index)
		sc.mu.Unlock()
	}()
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	if sc.killed() {
		reply.Err = WrongLeader
		return
	}

	isLeader := sc.isLeader()
	if !isLeader {
		reply.Err = WrongLeader
		return
	}

	sc.mu.Lock()
	op := Op{
		ClerkId: args.ClerkId,
		OpId:    args.OpId,
		OpType:  Query,
		Num:     args.Num,
	}
	if sc.isApplied(&op) {
		reply.Err = OK
		reply.Config = sc.handleQuery(args.Num)
		sc.mu.Unlock()
		return
	}

	index, _, isLeader := sc.rf.Start(op)
	DPrintf("Start Query, index: %v", index)

	if !isLeader {
		reply.Err = WrongLeader
		sc.mu.Unlock()
		return
	}

	ch := sc.getCh(index)
	sc.mu.Unlock()

	select {
	case result := <-ch:
		if result.ClerkId == args.ClerkId && result.OpId == args.OpId {
			reply.Err = OK
			reply.Config = result.config
		} else {
			reply.Err = WrongLeader
		}
	case <-time.After(50 * time.Millisecond):
		reply.Err = WrongLeader
	}
	go func() {
		sc.mu.Lock()
		delete(sc.IndexToCommand, index)
		sc.mu.Unlock()
	}()
}

// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
func (sc *ShardCtrler) Kill() {
	atomic.StoreInt32(&sc.dead, 1)
	sc.rf.Kill()
	// Your code here, if desired.
}

func (sc *ShardCtrler) killed() bool {
	z := atomic.LoadInt32(&sc.dead)
	return z == 1
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.maxAppliedOpIdofClerk = make(map[int64]int)
	sc.IndexToCommand = make(map[int]chan result)

	go sc.applier()

	return sc
}

func (sc *ShardCtrler) GetLastConfig() Config {
	if len(sc.configs) == 0 {
		return Config{}
	}
	return sc.configs[len(sc.configs)-1]
}

func (sc *ShardCtrler) handleJoin(Servers map[int][]string) *Config {
	lastConfig := sc.GetLastConfig()
	lastGroup := make(map[int][]string)
	for k, v := range lastConfig.Groups {
		lastGroup[k] = v
	}
	NewConfig := Config{
		Num:    len(sc.configs),
		Shards: lastConfig.Shards,
		Groups: lastGroup,
	}

	sc.PrintGidToShards(&NewConfig)

	flag := 0

	for _, v := range NewConfig.Shards {
		if v == 0 {
			flag = 1
		}
	}

	id := 0

	for gid, servers := range Servers {
		if flag == 1 && id == 0 && gid != 0 {
			id = gid
		}
		NewConfig.Groups[gid] = servers
	}

	DPrintf("Before rebalance")
	sc.PrintGidToShards(&NewConfig)

	if flag == 1 {
		for i := 0; i < NShards; i++ {
			NewConfig.Shards[i] = id
		}
		if len(NewConfig.Groups) > 1 {
			NewConfig.Shards = sc.rebalance(NewConfig.Shards, NewConfig.Groups, true)
		}
	} else {
		NewConfig.Shards = sc.rebalance(lastConfig.Shards, NewConfig.Groups, true)
	}

	DPrintf("After rebalance")
	NewConfig.Shards = sc.sortShards(NewConfig.Shards)
	sc.PrintGidToShards(&NewConfig)
	return &NewConfig
}

func (sc *ShardCtrler) handleLeave(GIDs []int) *Config {
	lastConfig := sc.GetLastConfig()
	lastGroup := make(map[int][]string)
	for k, v := range lastConfig.Groups {
		lastGroup[k] = v
	}
	NewConfig := Config{
		Num:    len(sc.configs),
		Shards: lastConfig.Shards,
		Groups: lastGroup,
	}
	DPrintf("leave gids: %v", GIDs)
	sc.PrintGidToShards(&NewConfig)
	for _, gid := range GIDs {
		delete(NewConfig.Groups, gid)
	}
	DPrintf("Before rebalance")
	sc.PrintGidToShards(&NewConfig)
	NewConfig.Shards = sc.rebalance(lastConfig.Shards, NewConfig.Groups, false)
	DPrintf("After rebalance")
	sc.PrintGidToShards(&NewConfig)
	NewConfig.Shards = sc.sortShards(NewConfig.Shards)

	return &NewConfig
}

func (sc *ShardCtrler) handleMove(gid, shard int) *Config {
	DPrintf("handleMove gid: %v, shard: %v", gid, shard)
	lastConfig := sc.GetLastConfig()
	lastGroup := make(map[int][]string)
	for k, v := range lastConfig.Groups {
		lastGroup[k] = v
	}
	NewConfig := Config{
		Num:    len(sc.configs),
		Shards: lastConfig.Shards,
		Groups: lastGroup,
	}
	NewConfig.Shards[shard] = gid
	NewConfig.Shards = sc.sortShards(NewConfig.Shards)
	return &NewConfig
}

func (sc *ShardCtrler) handleQuery(num int) Config {
	if num == -1 || num >= len(sc.configs) {
		return sc.configs[len(sc.configs)-1]
	}
	return sc.configs[num]
}

func (sc *ShardCtrler) sortShards(shards [NShards]int) [NShards]int {
	intSlice := shards[:]
	sort.Ints(intSlice)
	copy(shards[:], intSlice)
	return shards
}
