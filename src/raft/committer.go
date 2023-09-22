package raft

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// continuely apply entries
func (rf *Raft) committer() {
	rf.mu.Lock()
	for !rf.killed() {
		if rf.log.needPendSnapshot {
			snapshot := rf.log.cloneSnapshot()
			rf.mu.Unlock()
			rf.applyCh <- ApplyMsg{SnapshotValid: true, Snapshot: snapshot.Data, SnapshotTerm: snapshot.Term, SnapshotIndex: snapshot.Index}
			rf.mu.Lock()
			rf.log.needPendSnapshot = false
		} else if newCommittedEntries := rf.log.newCommittedEntries(); len(newCommittedEntries) > 0 {
			rf.mu.Unlock()
			for _, entry := range newCommittedEntries {
				rf.applyCh <- ApplyMsg{CommandValid: true, Command: entry.Command, CommandIndex: entry.Index}
			}
			rf.mu.Lock()
			rf.log.appliedTo(newCommittedEntries[len(newCommittedEntries)-1].Index)
			DPrintf("term %d server %d applied to %d", rf.currentTerm, rf.me, rf.log.applied)
		} else {
			rf.cond.Wait()
		}
	}
	rf.mu.Unlock()
}
