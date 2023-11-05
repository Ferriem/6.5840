package shardctrler

//
// Shardctrler clerk.
//

import (
	"crypto/rand"
	"math/big"
	"time"

	"6.5840/labrpc"
)

type Clerk struct {
	servers  []*labrpc.ClientEnd
	ClerkId  int64
	NextOpId int
	// Your data here.
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.ClerkId = nrand()
	ck.NextOpId = 0
	// Your code here.
	return ck
}

func (ck *Clerk) allocateOpId() int {
	opId := ck.NextOpId
	ck.NextOpId++
	return opId
}

func (ck *Clerk) Query(num int) Config {
	args := &QueryArgs{
		Num:     num,
		ClerkId: ck.ClerkId,
		OpId:    ck.allocateOpId(),
	}
	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply QueryReply
			ok := srv.Call("ShardCtrler.Query", args, &reply)
			if ok && reply.Err != WrongLeader {
				return reply.Config
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Join(servers map[int][]string) {
	args := &JoinArgs{
		Servers: servers,
		ClerkId: ck.ClerkId,
		OpId:    ck.allocateOpId(),
	}
	// Your code here.

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply JoinReply
			ok := srv.Call("ShardCtrler.Join", args, &reply)
			if ok && reply.Err != WrongLeader {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Leave(gids []int) {
	args := &LeaveArgs{
		GIDs:    gids,
		ClerkId: ck.ClerkId,
		OpId:    ck.allocateOpId(),
	}
	// Your code here.

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply LeaveReply
			ok := srv.Call("ShardCtrler.Leave", args, &reply)
			if ok && reply.Err != WrongLeader {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Move(shard int, gid int) {
	args := &MoveArgs{
		Shard:   shard,
		GID:     gid,
		ClerkId: ck.ClerkId,
		OpId:    ck.allocateOpId(),
	}
	// Your code here.

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply MoveReply
			ok := srv.Call("ShardCtrler.Move", args, &reply)
			if ok && reply.Err != WrongLeader {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
