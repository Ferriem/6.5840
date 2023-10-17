package kvraft

import (
	"crypto/rand"
	"math/big"
	"time"

	"6.5840/labrpc"
)

const retryInterval = 100 * time.Millisecond

type Clerk struct {
	servers  []*labrpc.ClientEnd
	ClerkId  int64
	leader   int
	NextOpId int
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
	ck.leader = 0
	ck.NextOpId = 0
	return ck
}

func (ck *Clerk) allocateOpId() int {
	opId := ck.NextOpId
	ck.NextOpId++
	return opId
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	args := &GetArgs{Key: key, ClerkId: ck.ClerkId, OpId: ck.allocateOpId()}
	for {
		for i := 0; i < len(ck.servers); i++ {
			serverId := (ck.leader + i) % len(ck.servers)
			var reply GetReply
			if ok := ck.servers[serverId].Call("KVServer.Get", args, &reply); ok {
				if reply.Err == OK {
					ck.leader = serverId
					return reply.Value
				}
			}
		}
		time.Sleep(retryInterval)
	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	args := &PutAppendArgs{Key: key, Value: value, OpType: op, ClerkId: ck.ClerkId, OpId: ck.allocateOpId()}
	for {
		for i := 0; i < len(ck.servers); i++ {
			serverId := (ck.leader + i) % len(ck.servers)
			var reply PutAppendReply
			if ok := ck.servers[serverId].Call("KVServer.PutAppend", args, &reply); ok {
				if reply.Err == OK {
					ck.leader = serverId
					return
				}
			}
		}
		time.Sleep(retryInterval)
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
