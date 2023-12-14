package shardkv

import (
	"time"
)

type PullArgs struct {
	Num     int
	ShardId int
}

type PullReply struct {
	Err                   Err
	DB                    map[string]string
	MaxAppliedOpIdofClerk map[int64]int
}

func (kv *ShardKV) ClearShard(args *PullArgs, reply *PullReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.OutedData[args.Num], args.ShardId)
	delete(kv.ShardToClient, args.ShardId)
	if len(kv.OutedData[args.Num]) == 0 {
		delete(kv.OutedData, args.Num)
	}
	_, _, isLeader := kv.rf.Start(Op{Cmd: ShardOp{Command: Clearshard, ShardId: args.ShardId, Num: args.Num}, ShardValid: true})

	if isLeader {
		reply.Err = OK
	}
}

func (kv *ShardKV) PullShard(args *PullArgs, reply *PullReply) {
	kv.mu.Lock()
	if _, isLeader := kv.rf.GetState(); !isLeader {
		reply.Err = ErrWrongLeader
		kv.mu.Unlock()
		return
	}

	if args.Num > kv.PreConfig.Num {
		reply.Err = Not
		kv.mu.Unlock()
		return
	}

	tmp, ok := kv.ShardState[args.ShardId]
	if !ok || kv.ShardNum[args.ShardId] < args.Num {
		reply.Err = Not
		kv.mu.Unlock()
		return
	}
	if tmp == OK && kv.ShardNum[args.ShardId] == args.Num {
		_, _, isleader := kv.rf.Start(Op{Cmd: ShardOp{Command: PULL, ShardId: args.ShardId, Num: args.Num}, ShardValid: true})
		if !isleader {
			reply.Err = Not
			kv.mu.Unlock()
			return
		}
		ch, exist := kv.Pullchan[args.Num*100+args.ShardId]
		if !exist {
			ch = make(chan PullReply, 1)
			kv.Pullchan[args.Num*100+args.ShardId] = ch
		}
		kv.mu.Unlock()
		select {
		case app := <-ch:
			reply.DB = app.DB
			reply.MaxAppliedOpIdofClerk = app.MaxAppliedOpIdofClerk
			reply.Err = OK
			DPrintf("pull success, reply %v", reply)
		case <-time.After(time.Millisecond * 1000):
			//DPrintf("pull timeout")
			reply.Err = Not
		}
		go func() {
			kv.mu.Lock()
			delete(kv.Pullchan, args.Num*100+args.ShardId)
			kv.mu.Unlock()
		}()
		return
	} else {
		reply.DB = make(map[string]string)
		for k, v := range kv.OutedData[args.Num][args.ShardId] {
			reply.DB[k] = v
		}
	}
	reply.Err = OK
	reply.MaxAppliedOpIdofClerk = make(map[int64]int)
	for k, v := range kv.maxAppliedOpIdofClerk {
		for _, value := range kv.ShardToClient[args.ShardId] {
			if value == k {
				reply.MaxAppliedOpIdofClerk[k] = v
				break
			}
		}
	}
	DPrintf("client to opid %v", kv.maxAppliedOpIdofClerk)
	kv.mu.Unlock()
}

func (kv *ShardKV) SendClearShard(server []string, args *PullArgs) {
	for s := 0; s < len(server); s++ {
		srv := kv.make_end(server[s])
		go func() {
			for !kv.killed() {
				var reply PullReply
				ok := srv.Call("ShardKV.ClearShard", args, &reply)
				if ok {
					return
				}
			}
		}()
	}
}

func (kv *ShardKV) SendPullShard(Group map[int][]string, args *PullArgs, oldgid int) {
	for !kv.killed() {
		if server, ok := Group[oldgid]; ok {
			var res PullReply
			flag := false
			for s := 0; s < len(server); s++ {
				srv := kv.make_end(server[s])
				for !kv.killed() {
					var reply PullReply
					ok := srv.Call("ShardKV.PullShard", args, &reply)
					DPrintf("PullShard reply %v", reply)
					if ok {
						if reply.Err == ErrWrongLeader {
							break
						}
						if reply.Err == OK {
							if len(reply.DB) >= len(res.DB) {
								res.DB = reply.DB
								res.MaxAppliedOpIdofClerk = reply.MaxAppliedOpIdofClerk
								flag = true
							}
							break
						}
						if reply.Err == Not {
							break
						}
					} else {
						break
					}
					time.Sleep(time.Millisecond * 100)
				}
			}
			if flag {
				kv.mu.Lock()
				op := ShardOp{
					Command:               MigrateShard,
					DB:                    res.DB,
					ShardId:               args.ShardId,
					Num:                   args.Num + 1,
					MaxAppliedOpIdofClerk: res.MaxAppliedOpIdofClerk,
					Servers:               server,
				}
				kv.rf.Start(Op{Cmd: op, ShardValid: true})
				kv.mu.Unlock()
				return
			} else {
				return
			}
		}
	}
}
