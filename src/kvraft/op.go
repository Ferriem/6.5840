package kvraft

import "time"

const NoOpInterval = 250 * time.Millisecond

type Op struct {
	ClerkId int64
	OpId    int
	OpType  string // "Get", "Put", "Append"
	Key     string
	Value   string
}
