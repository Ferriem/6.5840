package shardkv

import "time"

type KVStateMachine interface {
	Get(key string) (string, Err)
	Put(key string, value string) Err
	Append(key string, value string) Err
	Migrate(ShardId int, DB map[string]string) Err
	Copy(ShardId int) map[string]string
	Remove(ShardId int) Err
}

func (kv *ShardKV) applyStateMachine(op *KVOp) {
	DPrintf("applyStateMachine: %v", op)
	switch op.Command {
	case PUT:
		kv.StateMachine.Put(op.Key, op.Value)
	case APPEND:
		kv.StateMachine.Append(op.Key, op.Value)
	}
}

func (kv *ShardKV) UpdateConfig() {
	for !kv.killed() {
		if _, isLeader := kv.rf.GetState(); !isLeader {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		kv.mu.Lock()
		if kv.PreConfig.Num != kv.LastConfig.Num {
			updateConfig := true
			for shardid, gid := range kv.PreConfig.Shards {
				if gid == kv.gid && kv.LastConfig.Shards[shardid] != kv.gid && kv.ShardNum[shardid] != kv.PreConfig.Num {
					args := PullArgs{Num: kv.PreConfig.Num - 1, ShardId: shardid}
					oldgid := kv.LastConfig.Shards[shardid]
					updateConfig = false
					if oldgid == 0 {
						op := ShardOp{
							Command: MigrateShard,
							Num:     kv.PreConfig.Num,
							DB:      make(map[string]string),
							ShardId: shardid,
						}
						kv.rf.Start(Op{Cmd: op, ShardValid: true})
						continue
					}
					Group := make(map[int][]string)
					for k, v := range kv.LastConfig.Groups {
						Group[k] = v
					}
					go kv.SendPullShard(Group, &args, oldgid)
				}
			}
			if updateConfig {
				kv.LastConfig = kv.PreConfig
			}
			kv.mu.Unlock()
			time.Sleep(time.Millisecond * 100)
			continue
		}
		nextNum := kv.PreConfig.Num + 1
		kv.mu.Unlock()
		newConfig := kv.sc.Query(nextNum)
		kv.mu.Lock()
		if newConfig.Num == nextNum {
			kv.rf.Start(Op{Cmd: ShardOp{Command: UpdateConfig, Config: newConfig, Num: nextNum}, ShardValid: true})
			for shardid, gid := range newConfig.Shards {
				if gid == kv.gid && kv.PreConfig.Shards[shardid] != kv.gid {
					oldgid := kv.PreConfig.Shards[shardid]
					if oldgid == 0 { //initial state
						op := ShardOp{
							Command: MigrateShard,
							Num:     kv.PreConfig.Num + 1,
							DB:      make(map[string]string),
							ShardId: shardid,
						}
						kv.rf.Start(Op{Cmd: op, ShardValid: true})
						continue
					}
					args := PullArgs{Num: nextNum - 1, ShardId: shardid}
					Group := make(map[int][]string)
					for k, v := range kv.PreConfig.Groups {
						Group[k] = v
					}
					go kv.SendPullShard(Group, &args, oldgid)
				}
			}
		}
		kv.mu.Unlock()
		time.Sleep(time.Millisecond * 50)
	}
}
