package llog

import (
	"github.com/jmhodges/levigo"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/utils"
	"encoding/gob"
	"strconv"
	"bytes"
)

// cache size in MB
const CACHE_SIZE = 10 * 1024 * 1024
const BASE = 10

// levelDbLogStore is LevelDB based implementation
// of LogStore interface
type levelDbLogStore struct {
	localDb *levigo.DB // reference to LevelDB object
	nextIndex *utils.AtomicI64 // index of the next log entry
	firstIndex *utils.AtomicI64 // index of the first log entry
}

// Create creates a new Log at location
func Create(location string) (LogStore, error) {
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(CACHE_SIZE))
	opts.SetCreateIfMissing(true)
	db, err := levigo.Open(location, opts)
	if err != nil {
		return nil, err
	}
	opts.Close()
	logStore := &levelDbLogStore{localDb: db, nextIndex: &utils.AtomicI64{Value: 1}, firstIndex: &utils.AtomicI64{Value: 0}}
	return LogStore(logStore), err
}


// append appends a log entry and returns error if any.
// append also sets internal index of log entry as
// server itself need not be aware of index of the log
// while appending a new entry
func (l *levelDbLogStore) Append(entry *raft.LogEntry) error {
	writeOpts := levigo.NewWriteOptions()
	writeOpts.SetSync(true)
	defer writeOpts.Close()
	entry.Index = l.nextIndex.Get()
	data := logEntryToBytes(entry)
	key := entry.Index
	// insert values in LevelDb
	err := l.localDb.Put(writeOpts, int64ToBytes(key), data)
	l.nextIndex.Incr() // increment entry only after a write
	if l.firstIndex.Get() == 0 {
		l.firstIndex.Set(1)
	}
	return err
}


// get returns raft.LogEntry at given index. The index
// is logical index of the log entry and does not
// correspond to index in internal implementation
// TODO: Add error check here
func (l *levelDbLogStore) Get(index int64) *raft.LogEntry {
	readOpts := levigo.NewReadOptions()
	defer readOpts.Close()
	data, err := l.localDb.Get(readOpts, int64ToBytes(index))
	// for now panic locally. Propogate this error when fixed
	if err != nil {
		panic(err.Error())
	}

	if data == nil {
		return nil
	}
	return bytesToLogEntry(data)
}

// tail returns the latest entry in the log
func (l *levelDbLogStore) Tail() *raft.LogEntry {
	return l.Get(l.TailIndex())
}

// returns index of the latest entry in the log
// and 0 if the log does not have any entries
func (l *levelDbLogStore) TailIndex() int64 {
	return (l.nextIndex.Get() - 1)
}

// returns index of the first entry in the log
// and 0 if the log does not have any entries
func (l *levelDbLogStore) HeadIndex() int64 {
	return l.firstIndex.Get()
}

// Exists checks if an entry exists in log
// index is the index of the log entry
func (l *levelDbLogStore) Exists(index int64) bool {
	data := l.Get(index)
	// nil indicates that entry does not exist in LevelDB
	return data != nil
}

// DiscardFrom discards all entries from
// log index onwards (inclusive)
// TODO: Propogate error to higher layers
func (l *levelDbLogStore) DiscardFrom(index int64) {
	writeOpts := levigo.NewWriteOptions()
	writeOpts.SetSync(true)
	defer writeOpts.Close()
	writeBatch := levigo.NewWriteBatch()
	for i := index; i <= l.TailIndex(); i += 1 {
		writeBatch.Delete(int64ToBytes(i))
	}
	err := l.localDb.Write(writeOpts, writeBatch)
	if err != nil {
		panic("Error in DiscardFrom")
	}
	l.nextIndex.Set(index)
}

//--------------------------------------------------------------
func int64ToBytes(num int64) []byte {
	return []byte(strconv.FormatInt(num, BASE))
}

func logEntryToBytes(e *raft.LogEntry) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(e)
	return buf.Bytes()
}

// BytesToEnvelope decodes gob encoded representation of Envelope
func bytesToLogEntry(gobbed []byte) *raft.LogEntry {
	buf := bytes.NewBuffer(gobbed)
	dec := gob.NewDecoder(buf)
	var ungobbed raft.LogEntry
	dec.Decode(&ungobbed)
	return &ungobbed
}
