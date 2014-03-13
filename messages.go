// This file contains details about raft messages.
// The message formats are defined for raft in general.
// Thus, they should really be in a separate raft common
// package. For now, they are included here for convenience

package raft

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
	CandidateId int // pid of candidate
}

// AppendEntries struct is used in Raft for sending
// log messages and hear beats. For this component
// it only contains term and leaderid
type AppendEntry struct {
	Term     int64 // leader's term
	LeaderId int // pid of the leader
}

type GrantVote struct {
	Term        int64 // currentTerm for candidate
	VoteGranted bool
}

// EntryReply is reply message for AppendEntry request
type EntryReply struct {
	Term    int64  // replying server's updated current term
	Success bool // true if AppendEntry was accepted
}
