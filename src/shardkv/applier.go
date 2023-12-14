package shardkv

import "time"

func (kv *ShardKV) applier() {
	if !kv.killed() {
		for ch := range kv.applyCh {
			if ch.CommandValid {
				kv.mu.Lock()
				tmp := ch.Command.(Op)
				if tmp.ShardValid {
					op := tmp.Cmd.(ShardOp)
					switch op.Command {
					case MigrateShard:
						{
							DPrintf("receive MigrateShard: %v", op)
							if kv.ShardNum[op.ShardId] < op.Num && kv.ShardState[op.ShardId] != OK {
								kv.StateMachine.Migrate(op.ShardId, op.DB)
								delete(kv.ShardToClient, op.ShardId)
								for k, v := range op.MaxAppliedOpIdofClerk {
									if kv.maxAppliedOpIdofClerk[k] < v {
										kv.maxAppliedOpIdofClerk[k] = v
									}

									if len(kv.ShardToClient[op.ShardId]) == 0 {
										kv.ShardToClient[op.ShardId] = append(kv.ShardToClient[op.ShardId], k)
									} else {
										flag := true
										for _, exitClient := range kv.ShardToClient[op.ShardId] {
											if exitClient == k {
												flag = false
												break
											}
										}
										if flag {
											kv.ShardToClient[op.ShardId] = append(kv.ShardToClient[op.ShardId], k)
										}
									}
								}
								kv.ShardNum[op.ShardId] = op.Num
								kv.ShardState[op.ShardId] = OK
								args := PullArgs{Num: op.Num - 1, ShardId: op.ShardId}
								kv.SendClearShard(op.Servers, &args)
							}

						}
					case UpdateConfig:
						{
							DPrintf("receive UpdateConfig: %v", op)
							if kv.PreConfig.Num+1 == op.Config.Num {
								kv.LastConfig = kv.PreConfig
								kv.PreConfig = op.Config
								for shardid, gid := range kv.PreConfig.Shards {
									if gid != kv.gid && kv.LastConfig.Shards[shardid] == kv.gid {
										if kv.ShardState[shardid] == OK {
											Clone := kv.StateMachine.Copy(shardid)
											if len(Clone) == 0 {
												delete(kv.OutedData[op.Config.Num-1], shardid)
											} else {
												if len(kv.OutedData[op.Config.Num-1]) == 0 {
													tmp := make(map[int]map[string]string)
													tmp[shardid] = Clone
													kv.OutedData[op.Config.Num-1] = tmp
												} else {
													kv.OutedData[op.Config.Num-1][shardid] = Clone
												}
											}
											kv.StateMachine.Remove(shardid)
											kv.ShardState[shardid] = Not
											kv.ShardNum[shardid] = op.Config.Num
										}
									} else if gid == kv.gid && kv.LastConfig.Shards[shardid] == kv.gid {
										kv.ShardNum[shardid] = op.Config.Num
									}
								}
							}
						}
					case Clearshard:
						{
							DPrintf("receive Clearshard: %v", op)
							delete(kv.OutedData[op.Num], op.ShardId)
							delete(kv.ShardToClient, op.ShardId)
							if len(kv.OutedData[op.Num]) == 0 {
								delete(kv.OutedData, op.Num)
							}
						}
					case PULL:
						{
							DPrintf("receive PULL: %v", op)
							var reply PullReply
							tmp, ok := kv.ShardState[op.ShardId]
							if !ok || kv.ShardNum[op.ShardId] < op.Num {
								reply.Err = Not
							}
							if tmp == OK && kv.ShardNum[op.ShardId] == op.Num {
								reply.DB = kv.StateMachine.Copy(op.ShardId)
								Clone := kv.StateMachine.Copy(op.ShardId)
								if len(Clone) == 0 {
									delete(kv.OutedData[op.Num], op.ShardId)
								} else {
									if len(kv.OutedData[op.Num]) == 0 {
										tmp := make(map[int]map[string]string)
										tmp[op.ShardId] = Clone
										kv.OutedData[op.Num] = tmp
									} else {
										kv.OutedData[op.Num][op.ShardId] = Clone
									}
								}
								kv.StateMachine.Remove(op.ShardId)
								kv.ShardState[op.ShardId] = Not
								kv.ShardNum[op.ShardId] = op.Num
							} else {
								reply.DB = make(map[string]string)
								for k, v := range kv.OutedData[op.Num][op.ShardId] {
									reply.DB[k] = v
								}
							}
							reply.Err = OK
							reply.MaxAppliedOpIdofClerk = make(map[int64]int)
							for k, v := range kv.maxAppliedOpIdofClerk {
								for _, value := range kv.ShardToClient[op.ShardId] {
									if value == k {
										reply.MaxAppliedOpIdofClerk[k] = v
										break
									}
								}
							}
							ch2, exist := kv.Pullchan[op.Num*100+op.ShardId]
							if !exist {
								ch2 = make(chan PullReply, 1)
								kv.Pullchan[op.Num*100+op.ShardId] = ch2
							}
							go func() {
								select {
								case ch2 <- reply:
									return
								case <-time.After(time.Millisecond * 1000):
									return
								}
							}()
						}

					}
					if kv.LastApplied < ch.CommandIndex {
						kv.LastApplied = ch.CommandIndex
					}
					if kv.maxraftstate != -1 && kv.rf.GetRaftStateSize() > kv.maxraftstate {
						//DPrintf("snapshot: %v", ch.CommandIndex)
						kv.rf.Snapshot(kv.LastApplied, kv.PersistSnapShot())
					}
					kv.mu.Unlock()
					continue
				}
				op := tmp.Cmd.(KVOp)
				DPrintf("appiler receive op: %v, lastAppliedOpId :%v", op, kv.maxAppliedOpIdofClerk[op.ClientId])
				if !kv.CheckGroup(op.Key) || kv.ShardState[key2shard(op.Key)] != OK {
					kv.mu.Unlock()
					continue
				}
				if ch.CommandIndex <= kv.LastApplied {
					kv.mu.Unlock()
					continue
				}
				kv.LastApplied = ch.CommandIndex
				opchan := kv.GetChan(ch.CommandIndex)

				if kv.maxAppliedOpIdofClerk[op.ClientId] < op.OpId {
					kv.applyStateMachine(&op)
					kv.maxAppliedOpIdofClerk[op.ClientId] = op.OpId
					if len(kv.ShardToClient[key2shard(op.Key)]) == 0 {
						kv.ShardToClient[key2shard(op.Key)] = make([]int64, 0)
					}
					flag := 0
					for _, v := range kv.ShardToClient[key2shard(op.Key)] {
						if v == op.ClientId {
							flag = 1
							break
						}
					}
					if flag == 0 {
						kv.ShardToClient[key2shard(op.Key)] = append(kv.ShardToClient[key2shard(op.Key)], op.ClientId)
					}
				}

				if kv.maxraftstate != -1 && kv.rf.GetRaftStateSize() > kv.maxraftstate {
					//DPrintf("snapshot: %v", ch.CommandIndex)
					kv.rf.Snapshot(ch.CommandIndex, kv.PersistSnapShot())
				}

				if op.Command == GET {
					op.Value, _ = kv.StateMachine.Get(op.Key)
				}

				kv.mu.Unlock()

				opchan <- op
			}
			if ch.SnapshotValid {
				//DPrintf("receive snapshot: %v", ch.SnapshotIndex)
				kv.mu.Lock()
				if ch.SnapshotIndex > kv.LastApplied {
					kv.DecodeSnapShot(ch.Snapshot)
					kv.LastApplied = ch.SnapshotIndex
				}
				kv.mu.Unlock()
			}
		}
	}
}
