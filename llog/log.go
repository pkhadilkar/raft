// Package llog implements local log abstraction. This package
// handles storing data on disk and retrieving appropriate
// log entries and provides a simple interface to Raft layer
// Note that index in the internal representation of LogStore
// and logical index of LogEntry (one stored in LogEntry
// struct) may not match as space is reclaimed. LogStore
// functions interface deals exclusively with logical index
// of the entry and hides internal details of the LogStore
// implementation.
// Note that log indices start at 1 not 0.
package llog

import (
	"github.com/pkhadilkar/raft"
	"sync"
)

type LogStore struct {
	log          []*raft.LogEntry // in memory array to store Log entries
	nextIndex    int64            // index of the last log entry
	sync.RWMutex                  // lock to access log immutably
}

func Create() *LogStore {
	l := &LogStore{}
	l.Init()
	return l
}

// init initializes LogStore
func (l *LogStore) Init() {
	l.log = make([]*raft.LogEntry, 1)
	l.log[0] = &raft.LogEntry{Index: -1, Term: -1} // additional entry to simplify index access
	l.nextIndex = 1
}

// append appends a log entry and returns error if any.
// append also sets internal index of log entry as
// server itself need not be aware of index of the log
// while appending a new entry
func (l *LogStore) Append(entry *raft.LogEntry) error {
	l.Lock()
	entry.Index = l.nextIndex
	l.nextIndex += 1
	l.log = append(l.log, entry)
	l.Unlock()
	return nil
}

// get returns raft.LogEntry at given index. The index
// is logical index of the log entry and does not
// correspond to index in internal implementation
func (l *LogStore) Get(index int64) *raft.LogEntry {
	// no need to add error checking here
	// index accesses are checked at
	// runtime
	l.RLock()
	defer l.RUnlock()
	return l.log[index]
}

// tail returns the latest entry in the log
func (l *LogStore) Tail() *raft.LogEntry {
	l.RLock()
	defer l.RUnlock()
	return l.log[l.TailIndex()]
}

// returns index of the latest entry in the log
// and 0 if the log does not have any entries
func (l *LogStore) TailIndex() int64 {
	l.RLock()
	defer l.RUnlock()
	return l.nextIndex - 1
}

// Exists checks if an entry exists in log
// index is the index of the log entry
func (l *LogStore) Exists(index int64) bool {
	return index < l.nextIndex
}

// DiscardFrom discards all entries from
// log index onwards (inclusive)
func (l *LogStore) DiscardFrom(index int64) {
	l.log = l.log[:index]
	l.nextIndex = index
}
