package raftImpl

import (
	"github.com/pkhadilkar/cluster"
	"strconv"
	"time"
)

// this file contains functions common to all Raft servers

const NICE = 1 // amount of time in milliseconds goroutines
// should sleep to release the CPU in
// non-event driven functions

// handleRequestVote  handles RequestVote messages
// when server is in candidate state
func (s *raftServer) handleRequestVote(from int, rv *RequestVote) bool {
	acc := false

	// in currentTerm candidate votes for itself
	s.writeToLog("Received requestVote message from " + strconv.Itoa(from) + " with term #" + strconv.FormatInt(rv.Term, TERM_BASE) + "VotedFor = " + strconv.Itoa(s.VotedFor()))

	if (s.VotedFor() == from || s.VotedFor() == NotVoted) && s.isMoreUpToDate(rv.LastLogIndex, rv.LastLogTerm) {
		s.setTerm(rv.Term)
		s.voteFor(from, s.Term())
		if s.State() != FOLLOWER {
			s.setState(FOLLOWER)
		}
		acc = true
		s.writeToLog("Granting vote to " + strconv.Itoa(from) + ".Changing state to follower")
		s.resetElectionTimeout()
	}
	s.replyTo(from, &GrantVote{Term: s.Term(), VoteGranted: acc})
	return acc
}

// handleAppendEntry handles AppendEntry messages
func (s *raftServer) handleAppendEntry(from int, ae *AppendEntry) bool {
	acc := false
	//s.writeToLog("Received appendEntry message from " + strconv.Itoa(from) + " with term #" + strconv.FormatInt(ae.Term, TERM_BASE))
	if ae.Term >= s.Term() { // AppendEntry with same or larger term
		s.leaderId.Set(ae.LeaderId)
		if isHeartbeat(&ae.Entry) {
			// heartbeat
			ae.Entry.Index = HEARTBEAT
			acc = true
		} else if s.localLog.Exists(ae.PrevLogIndex) && s.localLog.Get(ae.PrevLogIndex).Term == ae.PrevLogTerm || (ae.PrevLogIndex == -1 && ae.PrevLogTerm == -1) {
			if s.localLog.Exists(ae.Entry.Index) && s.localLog.Get(ae.Entry.Index).Term != ae.Entry.Term {
				// existing entry conflicts with the entry from the leader
				s.localLog.DiscardFrom(ae.Entry.Index)
			}
			// apply entry to local log
			s.localLog.Append(&ae.Entry)
			acc = true
		}

		if acc {
			s.updateCommitIndex(ae)
		}

		s.setTerm(ae.Term)
		s.setState(FOLLOWER)
	}
	s.replyTo(from, &EntryReply{Term: s.Term(), Success: acc, LogIndex: ae.Entry.Index})
	return acc
}

func (s *raftServer) sendRequestVote() {
	rv := &RequestVote{Term: s.Term(), CandidateId: s.server.Pid()}
	tailEntry := s.localLog.Tail()
	rv.LastLogTerm = tailEntry.Term
	rv.LastLogIndex = tailEntry.Index
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: rv}
	s.writeToLog("Sending message (Pid: " + strconv.Itoa(e.Pid) + ", CandidateId: " + strconv.Itoa(s.server.Pid()))
	s.server.Outbox() <- e
}

// logApply checks for updates to commitIndex
// and delivers entries to the state machine
func (s *raftServer) logApply() {
	for {
		if s.commitIndex.Get() > s.lastApplied.Get() {
			N := s.lastApplied.Get() + 1 // lastApplied is accessed only here
			s.inbox <- s.localLog.Get(N)
			s.writeToLog("Applied log entry at index " + strconv.FormatInt(N, 10))
			s.lastApplied.Set(N)
		}
		time.Sleep(NICE * time.Millisecond) // since goroutines are cooperative
	}
}

func (s *raftServer) isMoreUpToDate(LastLogIndex int64, LastLogTerm int64) bool {
	s.writeToLog("LastLogIndex: " + strconv.FormatInt(LastLogIndex, 10) + "\tLastLogTerm: " + strconv.FormatInt(LastLogTerm, 10))
	latest := s.localLog.Tail()
	if latest.Term == -1 && latest.Index == -1 { // log has no entries yet
		return true
	}
	return LastLogTerm > latest.Term || LastLogTerm == latest.Term && LastLogIndex > latest.Index
}

func (s *raftServer) updateCommitIndex(ae *AppendEntry) {
	if ae.LeaderCommit > s.commitIndex.Get() {
		//s.writeToLog("Updating commitIndex. LeaderCommit = " + strconv.FormatInt(ae.LeaderCommit, 10))
		s.commitIndex.Set(min(ae.LeaderCommit, s.localLog.TailIndex()))
	}
}

func (s *raftServer) resetElectionTimeout() {
	candidateTimeout := time.Duration(s.config.TimeoutInMillis + s.rng.Int63n(RandomTimeoutRange))
	s.eTimeout.Reset(candidateTimeout * time.Millisecond)
}
