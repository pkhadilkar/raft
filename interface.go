package raft

// Raft object interface.
// Term returns current server term
// isLeader returns true on leader and false on followers

type Raft interface {
	Term() int64
	Leader() int

	// Mailbox for state machine layer above to send commands of any
	// kind, and to have them replicated by raft.  If the server is not
	// the leader, the message will be silently dropped.
	Inbox() chan<- interface{}

	//Mailbox for state machine layer above to receive commands. These
	//are guaranteed to have been replicated on a majority
	Outbox() <-chan *LogEntry

	//Remove items from 0 .. index (inclusive), and reclaim disk
	//space. This is a hint, and there's no guarantee of immediacy since
	//there may be some servers that are lagging behind).

	DiscardUpto(index int64)
}

// Identifies an entry in the log
type LogEntry struct {
	// An index into an abstract 2^64 array
	Index int64

	// Term when log entry was created
	Term int64

	// The data that was supplied to raft's inbox
	Data interface{}
}
