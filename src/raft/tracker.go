package raft

import (
	"time"
)

const activeTimeout = 300 * time.Millisecond

type PeerTracker struct {
	nextIndex  int
	matchIndex int
	lastAck    time.Time
}

func (rf *Raft) ResetTrackerIndex() {
	for i := range rf.peerTrackers {
		rf.peerTrackers[i].nextIndex = rf.log.lastIndex() + 1
		rf.peerTrackers[i].matchIndex = 0
	}
}

func (rf *Raft) checkActive() bool {
	activePeers := 1
	for i := range rf.peerTrackers {
		if i != rf.me && time.Since(rf.peerTrackers[i].lastAck) <= activeTimeout {
			activePeers++
		}
	}
	return 2*activePeers > len(rf.peers)
}
