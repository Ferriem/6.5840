package shardctrler

type Op struct {
	ClerkId int64
	OpId    int
	OpType  string
	Servers map[int][]string //Join
	GIDs    []int            //Leave
	Shard   int              //Move
	GID     int              //Move
	Num     int              //Query
}
