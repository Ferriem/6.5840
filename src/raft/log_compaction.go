package raft

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if index <= rf.log.snapshot.Index {
		return
	}
	snapshotTerm, _ := rf.log.term(index)
	rf.log.compactTo(Snapshot{Term: snapshotTerm, Index: index, Data: snapshot})
	rf.persist()
}

func (rf *Raft) makeInstallSnapShot(to int) *InstallSnapshotArgs {
	args := &InstallSnapshotArgs{From: rf.me, To: to, Term: rf.currentTerm, Snapshot: rf.log.snapshot}
	return args
}

func (rf *Raft) sendInstallSnapshot(args *InstallSnapshotArgs) {
	reply := InstallSnapshotReply{}
	if ok := rf.peers[args.To].Call("Raft.InstallSnapshot", args, &reply); ok {
		rf.handleInstallSnapshotReply(args, &reply)
	}
}

func (rf *Raft) lagBehind(to int) bool {
	return rf.peerTrackers[to].nextIndex <= rf.log.firstIndex()
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.From = rf.me
	reply.To = args.From
	reply.Term = rf.currentTerm
	reply.Catchup = false

	m := Message{Type: SnapShot, From: args.From, Term: args.Term}
	ok, termChanged := rf.checkMessage(m)
	if termChanged {
		reply.Term = rf.currentTerm
		defer rf.persist()
	}
	if !ok {
		return
	}

	if args.Snapshot.Index <= rf.log.committed {
		reply.Catchup = true
		return
	}

	rf.log.compactTo(args.Snapshot)
	reply.Catchup = true
	if !termChanged {
		defer rf.persist()
	}
	rf.log.needPendSnapshot = true
	rf.cond.Signal()
}

func (rf *Raft) handleInstallSnapshotReply(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	m := Message{Type: SnapShotReply, From: reply.From, Term: reply.Term, ArgsTerm: args.Term}
	ok, termChanged := rf.checkMessage(m)
	if termChanged {
		defer rf.persist()
	}
	if !ok {
		return
	}

	if reply.Catchup {
		rf.peerTrackers[reply.From].matchIndex = args.Snapshot.Index
		rf.peerTrackers[reply.From].nextIndex = args.Snapshot.Index + 1
		rf.broadcastAppendEntries(true)
	}

}
