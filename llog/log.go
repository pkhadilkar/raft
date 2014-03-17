// Package llog implements local log abstraction. This package
// handles storing data on disk and retrieving appropriate
// log entries and provides a simple interface to Raft layer
// Note that index in the internal representation of LogStore
// and logical index of LogEntry (one stored in LogEntry
// struct) may not match as space is reclaimed. LogStore
// functions interface deals exclusively with logical index
// of the entry and hides internal details of the LogStore
// implementation
package llog

import (
	"github.com/pkhadilkar/raft"
	"sync"
)

type LogStore struct {
	log          []*raft.LogEntry // in memory array to store Log entries
	size         int64            // number of entries in log
	sync.RWMutex                  // lock to access log immutably
}

// init initializes LogStore
func (l *LogStore) Init() {
	l.log = make([]*raft.LogEntry, 100)
	l.size = 0
}

// append appends a log entry and returns error if any.
// append also sets internal index of log entry as
// server itself need not be aware of index of the log
// while appending a new entry
func (l *LogStore) Append(entry *raft.LogEntry) error {
	l.Lock()
	entry.Index = l.size
	l.size += 1
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
	return l.log[l.size-1]
}

// returns index of the latest entry in the log
func (l *LogStore) TailIndex() int64 {
	l.RLock()
	defer l.RUnlock()
	return l.size - 1
}

// Exists checks if an entry exists in log
func (l *LogStore) Exists(index int64) bool {
	return index < l.size
}
