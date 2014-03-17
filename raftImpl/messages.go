// This file contains details about raft messages.
// The message formats are defined for raft in general.
// Thus, they should really be in a separate raft common
// package. For now, they are included here for convenience

package raftImpl

// enum to indicate type of the message
const (
	REQ_VOTE = iota
	APP_ENTRY
	GRANT_VOTE
	ENTRY_REPLY
)

// RequestVote struct is used in Raft leader election
type RequestVote struct {
	Term        int64 // candidate's term
	CandidateId int   // pid of candidate
}

// AppendEntries struct is used in Raft for sending
// log messages and hear beats. For this component
// it only contains term and leaderid
type AppendEntry struct {
	Term         int64 // leader's term
	LeaderId     int   // pid of the leader
	PrevLogIndex int64 // index of previous log entry in Leader's log
	PrevLogTerm  int64 // term for previous log entry in Leader's log
	LeaderCommit int64 // last commited index in Leader's log
}

type GrantVote struct {
	Term        int64 // currentTerm for candidate
	VoteGranted bool
}

// EntryReply is reply message for AppendEntry request
// Note that we have to include LogIndex in EntryReply
// as our replication mechanism does not use true RPC
// Adding LogIndex makes EntryReply idempotent
type EntryReply struct {
	Term    int64 // replying server's updated current term
	Success bool  // true if AppendEntry was accepted
	LogIndex int64 // index of the log entry if AppendEntry call is succesfull
}

// To see why idempotent EntryReply is necessary in
// this implementation. Consider following scenario
// 1. Leader sends AppendEntry for LogIndex l to follower
// 2. AppendEntry reaches follower and follower replies
// 3. Leader retries AppendEntry for LogIndex l to
//    follower before EntryReply is delivered to server
// 4. EntryReply delivered to leader. Leader increments
//    matchIndex for that follower
// 5. Follower receives second AppendEntry for LogIndex
//    l and replies affirmative to Leader 
// 6. EntryReply is delivered to Leader and leader again
//    moves matchIndex forward for follower
// Thus, a matchIndex was skipped for a follower
