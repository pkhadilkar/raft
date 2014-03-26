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
	"encoding/gob"
)

type LogStore interface {
	// append appends a log entry and returns error if any.
	// append also sets internal index of log entry as
	// server itself need not be aware of index of the log
	// while appending a new entry
	Append(entry *raft.LogEntry) error

	// get returns raft.LogEntry at given index. The index
	// is logical index of the log entry and does not
	// correspond to index in internal implementation
	Get(index int64) *raft.LogEntry

	// tail returns the latest entry in the log
	Tail() *raft.LogEntry

	// returns index of the latest entry in the log
	// and 0 if the log does not have any entries
	TailIndex() int64

	// Exists checks if an entry exists in log
	// index is the index of the log entry
	Exists(index int64) bool

	// DiscardFrom discards all entries from
	// log index onwards (inclusive)
	DiscardFrom(index int64)
}


func init() {
	gob.Register(raft.LogEntry{})
}
