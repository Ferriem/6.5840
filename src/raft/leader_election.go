package raft

import (
	"math/rand"
	"time"
)

func RandomizedElectionTimeout() time.Duration {
	return time.Duration(150+rand.Int63()%150) * time.Millisecond
}

func (rf *Raft) checkElectionTimeout() bool {
	return time.Since(rf.lastElection) > rf.electionTimeout
}

func (rf *Raft) resetElectionTimeout() {
	rf.electionTimeout = RandomizedElectionTimeout()
	rf.lastElection = time.Now()
}

func (rf *Raft) becomeFollower(term int) bool {
	rf.state = Follower
	if term > rf.currentTerm {
		rf.currentTerm = term
		rf.votedFor = -1
		return true
	}
	return false
}

func (rf *Raft) becomeCandidate() {
	defer rf.persist()
	rf.state = Candidate
	rf.currentTerm++
	rf.voteMe = make([]bool, len(rf.peers))
	rf.votedFor = rf.me
	rf.resetElectionTimeout()
}

func (rf *Raft) becomeLeader() {
	rf.state = Leader
	rf.ResetTrackerIndex()
}

func (rf *Raft) makeRequestVoteArgs(to int) *RequestVoteArgs {
	lastLogIndex := rf.log.lastIndex()
	lastLogTerm, _ := rf.log.term(lastLogIndex)
	args := &RequestVoteArgs{From: rf.me, To: to, Term: rf.currentTerm, LastLogIndex: lastLogIndex, LastLogTerm: lastLogTerm}
	return args
}

func (rf *Raft) sendRequestVote(args *RequestVoteArgs) {
	reply := RequestVoteReply{}
	if ok := rf.peers[args.To].Call("Raft.RequestVote", args, &reply); ok {
		rf.handleRequestVoteReply(args, &reply)
	}
}

func (rf *Raft) broadcastRequestVote() {
	for i := range rf.peers {
		if i != rf.me {
			args := rf.makeRequestVoteArgs(i)
			go rf.sendRequestVote(args)
		}
	}
}

func (rf *Raft) isUpToDate(candidateLastLogTerm int, candidateLastLogIndex int) bool {
	lastLogIndex := rf.log.lastIndex()
	lastLogTerm, _ := rf.log.term(lastLogIndex)
	return candidateLastLogTerm > lastLogTerm || (candidateLastLogTerm == lastLogTerm && candidateLastLogIndex >= lastLogIndex)

}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("term %d server %d receive vote request from %d", rf.currentTerm, rf.me, args.From)

	reply.From = rf.me
	reply.To = args.From
	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	m := Message{Type: Vote, From: args.From, Term: args.Term}
	ok, termChanged := rf.checkMessage(m)

	if termChanged {
		reply.Term = rf.currentTerm
		defer rf.persist()
	}
	if !ok {
		return
	}
	if (rf.votedFor == -1 || rf.votedFor == args.From) && rf.isUpToDate(args.LastLogTerm, args.LastLogIndex) {
		rf.votedFor = args.From
		DPrintf("term %d server %d vote for %d", rf.currentTerm, rf.me, args.From)
		reply.VoteGranted = true
		rf.resetElectionTimeout()
	}
}

func (rf *Raft) checkVoted() bool {
	count := 1
	for i, voteMe := range rf.voteMe {
		if i != rf.me && voteMe {
			count++
		}
	}
	return 2*count > len(rf.peers)
}

func (rf *Raft) handleRequestVoteReply(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	m := Message{Type: VoteReply, From: reply.From, Term: reply.Term, ArgsTerm: args.Term}
	ok, termChanged := rf.checkMessage(m)
	if termChanged {
		defer rf.persist()
	}
	if !ok {
		return
	}

	if reply.VoteGranted {
		rf.voteMe[reply.From] = true
		if rf.checkVoted() {
			DPrintf("term %d server %d become leader", rf.currentTerm, rf.me)
			rf.becomeLeader()
		}
	}
}
