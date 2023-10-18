package kvraft

type Op struct {
	ClerkId int64
	OpId    int
	OpType  string // "Get", "Put", "Append"
	Key     string
	Value   string
}
