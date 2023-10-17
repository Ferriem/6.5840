package kvraft

type Err string

const (
	OK             = "OK"
	ErrNotApplied  = "ErrNotApplied"
	ErrWrongLeader = "ErrWrongLeader"
)

// Put or Append
type PutAppendArgs struct {
	Key     string
	Value   string
	OpType  string
	ClerkId int64
	OpId    int
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key     string
	ClerkId int64
	OpId    int
}

type GetReply struct {
	Err   Err
	Value string
}
