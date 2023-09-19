package raft

import (
	"errors"
)

var ErrIndex = errors.New("index out of range")

type LogEntry struct {
	Index   int
	Term    int
	Command interface{}
}

type Log struct {
	entries   []LogEntry
	applied   int
	committed int
}

func makeLog() Log {
	log := Log{
		entries:   []LogEntry{{Index: 0, Term: 0}},
		applied:   0,
		committed: 0,
	}
	return log
}

func (log *Log) toArrayIndex(index int) int {
	return index - log.firstIndex()
}

func (log *Log) firstIndex() int {
	return log.entries[0].Index
}

func (log *Log) lastIndex() int {
	return log.entries[len(log.entries)-1].Index
}

func (log *Log) term(index int) (int, error) {
	if index < log.firstIndex() || index > log.lastIndex() {
		return 0, ErrIndex
	}
	index = log.toArrayIndex(index)
	return log.entries[index].Term, nil
}

func (log *Log) clone(entries []LogEntry) []LogEntry {
	cloned := make([]LogEntry, len(entries))
	copy(cloned, entries)
	return cloned
}

func (log *Log) slice(start, end int) []LogEntry {
	if start == end {
		return nil
	}
	start = log.toArrayIndex(start)
	end = log.toArrayIndex(end)
	return log.clone(log.entries[start:end])
}

func (log *Log) truncateSuffix(index int) {
	if index <= log.firstIndex() || index > log.lastIndex() {
		return
	}
	index = log.toArrayIndex(index)
	if len(log.entries[index:]) > 0 {
		DPrintf("truncate")
		log.entries = log.entries[:index]
	}
}

func (log *Log) append(entries []LogEntry) {
	log.entries = append(log.entries, entries...)
}

func (log *Log) committedTo(index int) {
	if index > log.committed {
		log.committed = index
	}
}

func (log *Log) newCommittedEntries() []LogEntry {
	start := log.toArrayIndex(log.applied + 1)
	end := log.toArrayIndex(log.committed + 1)
	if start >= end {
		return nil
	}
	return log.clone(log.entries[start:end])
}

func (log *Log) appliedTo(index int) {
	if index > log.applied {
		log.applied = index
	}
}
