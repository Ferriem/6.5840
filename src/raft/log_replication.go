package raft

import (
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rf *Raft) checkHeartbeatTimeout() bool {
	return time.Since(rf.lastHeartBeat) > rf.heartbeatTimeout
}

func (rf *Raft) resetHeartbeatTimeout() {
	rf.lastHeartBeat = time.Now()
}

func (rf *Raft) makeAppendEntriesArgs(to int) *AppendEntriesArgs {
	nextIndex := rf.peerTrackers[to].nextIndex
	entries := rf.log.slice(nextIndex, rf.log.lastIndex()+1)
	prevLogIndex := nextIndex - 1
	prevLogTerm, _ := rf.log.term(prevLogIndex)
	args := &AppendEntriesArgs{From: rf.me, To: to, Term: rf.currentTerm, PrevLogIndex: prevLogIndex, PrevLogTerm: prevLogTerm, Entries: entries, CommittedIndex: rf.log.committed}
	return args
}

func (rf *Raft) sendAppendEntries(args *AppendEntriesArgs) {
	reply := AppendEntriesReply{}
	if ok := rf.peers[args.To].Call("Raft.AppendEntries", args, &reply); ok {
		rf.handleAppendEntriesReply(args, &reply)
	}
}

func (rf *Raft) hasNewEntries(to int) bool {
	return rf.peerTrackers[to].nextIndex <= rf.log.lastIndex()
}

func (rf *Raft) broadcastAppendEntries(heartbeat bool) {
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		if rf.lagBehind(i) {
			args := rf.makeInstallSnapShot(i)
			go rf.sendInstallSnapshot(args)
		} else if heartbeat || rf.hasNewEntries(i) {
			args := rf.makeAppendEntriesArgs(i)
			DPrintf("send append entries from %d to %d", rf.me, i)
			go rf.sendAppendEntries(args)
		}
	}
}

func (rf *Raft) checkLogPrefixMatched(leaderPrevLogIndex, leaderPrevLogTerm int) Err {
	prevLogTerm, err := rf.log.term(leaderPrevLogIndex)
	if err != nil {
		return IndexNotMatched
	}
	if prevLogTerm != leaderPrevLogTerm {
		return TermNotMatched
	}
	return Matched
}

func (rf *Raft) findFirstConflict(index int) (int, int) {
	conflictTerm, _ := rf.log.term(index)
	firstConflictIndex := index
	for i := index - 1; i >= rf.log.firstIndex(); i-- {
		if term, _ := rf.log.term(i); term != conflictTerm {
			break
		}
		firstConflictIndex = i
	}
	return conflictTerm, firstConflictIndex
}

func (rf *Raft) maybeCommitted(index int) {
	if index > rf.log.committed {
		rf.log.committedTo(index)
		rf.cond.Signal()
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("term %d server %d receive append entries from %d", rf.currentTerm, rf.me, args.From)
	reply.From = rf.me
	reply.To = args.From
	reply.Term = rf.currentTerm
	reply.Err = Reject

	m := Message{Type: Append, From: args.From, Term: args.Term}
	ok, termChanged := rf.checkMessage(m)
	if termChanged {
		reply.Term = rf.currentTerm
		defer rf.persist()
	}
	if !ok {
		return
	}
	reply.Err = rf.checkLogPrefixMatched(args.PrevLogIndex, args.PrevLogTerm)
	if reply.Err != Matched {
		if reply.Err == IndexNotMatched {
			reply.LastLogIndex = rf.log.lastIndex()
		} else {
			reply.ConflictTerm, reply.FirstConflictIndex = rf.findFirstConflict(args.PrevLogIndex)
		}
		return
	}

	for i, entry := range args.Entries {
		if term, err := rf.log.term(entry.Index); err != nil || term != entry.Term {
			rf.log.truncateSuffix(entry.Index)
			rf.log.append(args.Entries[i:])
			// for j := range rf.log.entries {
			// 	DPrintf("term %d server %d log %d log term %d log command %v: %v", rf.currentTerm, rf.me, j, rf.log.entries[j].Term, rf.log.entries[j].Command, rf.log.entries[j].Index)
			// }
			if !termChanged {
				rf.persist()
			}
			break
		}
	}
	lastNewLogIndex := min(args.CommittedIndex, args.PrevLogIndex+len(args.Entries))
	rf.maybeCommitted(lastNewLogIndex)
}

func (rf *Raft) checkMatched(index int) bool {
	count := 1
	for _, tracker := range rf.peerTrackers {
		if tracker.matchIndex >= index {
			count++
		}
	}
	return 2*count > len(rf.peers)
}

func (rf *Raft) maybeCommitMatched(index int) bool {
	for i := index; i > rf.log.committed; i-- {
		if term, _ := rf.log.term(i); term == rf.currentTerm && rf.checkMatched(i) {
			rf.log.committedTo(i)
			rf.cond.Signal()
			return true
		}
	}
	return false
}

func (rf *Raft) handleAppendEntriesReply(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	m := Message{Type: AppendReply, From: reply.From, Term: reply.Term, ArgsTerm: args.Term, PrevLogIndex: args.PrevLogIndex}
	ok, termChanged := rf.checkMessage(m)
	if termChanged {
		defer rf.persist()
	}
	if !ok {
		return
	}

	switch reply.Err {
	case Reject:
	case Matched:
		rf.peerTrackers[reply.From].matchIndex = args.PrevLogIndex + len(args.Entries)
		rf.peerTrackers[reply.From].nextIndex = rf.peerTrackers[reply.From].matchIndex + 1
		if rf.maybeCommitMatched(rf.peerTrackers[reply.From].matchIndex) {
			rf.broadcastAppendEntries(true)
		}
	case IndexNotMatched:
		if reply.LastLogIndex < rf.log.lastIndex() {
			rf.peerTrackers[reply.From].nextIndex = reply.LastLogIndex + 1
		} else {
			rf.peerTrackers[reply.From].nextIndex = rf.log.lastIndex() + 1
		}
		rf.broadcastAppendEntries(true)
	case TermNotMatched:
		newNextIndex := reply.FirstConflictIndex
		for i := rf.log.lastIndex(); i > rf.log.firstIndex(); i-- {
			if term, _ := rf.log.term(i); term == reply.ConflictTerm {
				newNextIndex = i
				break
			}
		}
		rf.peerTrackers[reply.From].nextIndex = newNextIndex
		rf.broadcastAppendEntries(true)
	}
}
