package raft

import (
	"time"
)

type RequestVoteArgs struct {
	From         int
	To           int
	Term         int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	From        int
	To          int
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	From           int
	To             int
	Term           int
	PrevLogIndex   int
	PrevLogTerm    int
	Entries        []LogEntry
	CommittedIndex int
}

type InstallSnapshotArgs struct {
	From     int
	To       int
	Term     int
	Snapshot Snapshot
}

type InstallSnapshotReply struct {
	From    int
	To      int
	Term    int
	Catchup bool
}

type Err int

const (
	Reject Err = iota
	Matched
	IndexNotMatched
	TermNotMatched
)

type AppendEntriesReply struct {
	From               int
	To                 int
	Term               int
	Err                Err
	LastLogIndex       int
	ConflictTerm       int
	FirstConflictIndex int
}

type MessageType string

const (
	Vote          MessageType = "RequestVote"
	VoteReply     MessageType = "RequestVoteReply"
	Append        MessageType = "AppendEntires"
	AppendReply   MessageType = "AppendEntriesReply"
	SnapShot      MessageType = "InstallSnapshot"
	SnapShotReply MessageType = "InstallSnapshotReply"
)

type Message struct {
	Type         MessageType
	From         int
	Term         int
	ArgsTerm     int
	PrevLogIndex int
}

func (rf *Raft) checkTerm(m Message) (bool, bool) {
	if m.Term < rf.currentTerm {
		return false, false
	}
	if m.Term > rf.currentTerm || m.Type == Append || m.Type == SnapShot {
		termChanged := rf.becomeFollower(m.Term)
		return true, termChanged
	}
	return true, false
}

func (rf *Raft) checkState(m Message) bool {
	res := false
	switch m.Type {
	case Vote:
		fallthrough
	case Append:
		res = rf.state == Follower
	case VoteReply:
		res = rf.state == Candidate && rf.currentTerm == m.ArgsTerm
	case AppendReply:
		res = rf.state == Leader && rf.currentTerm == m.ArgsTerm && rf.peerTrackers[m.From].nextIndex == m.PrevLogIndex+1
	case SnapShot:
		res = rf.state == Follower && !rf.log.needPendSnapshot
	case SnapShotReply:
		res = rf.state == Leader && rf.currentTerm == m.ArgsTerm && rf.lagBehind(m.From)
	default:
		DPrintf("term %d server %d receive unknown message type %s", rf.currentTerm, rf.me, m.Type)
	}

	if rf.state == Follower && m.Type == Append || m.Type == SnapShot {
		rf.resetElectionTimeout()
	}
	return res
}

func (rf *Raft) checkMessage(m Message) (bool, bool) {
	if m.Type == VoteReply || m.Type == AppendReply || m.Type == SnapShotReply {
		rf.peerTrackers[m.From].lastAck = time.Now()
	}
	ok, termChanged := rf.checkTerm(m)
	if !ok || !rf.checkState(m) {
		return false, termChanged
	}
	return true, termChanged

}
