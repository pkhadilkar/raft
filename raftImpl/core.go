package raftImpl

import (
	"github.com/pkhadilkar/cluster"
	"time"
	"strconv"
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
	s.writeToLog("Received requestVote message from " + strconv.Itoa(from) + " with term #" + strconv.FormatInt(rv.Term, TERM_BASE))

	if (s.VotedFor() == from || s.VotedFor() == NotVoted) && s.isMoreUpToDate(rv.LastLogIndex, rv.LastLogTerm)|| rv.Term > s.Term() {
		s.setTerm(rv.Term)
		s.voteFor(from, s.Term())
		if s.State() != FOLLOWER {
			s.setState(FOLLOWER)
		}
		acc = true
		s.writeToLog("Granting vote to " + strconv.Itoa(from) + ".Changing state to follower")
	}
	s.replyTo(from, &GrantVote{Term: s.Term(), VoteGranted: acc})
	return acc
}

// handleAppendEntry handles AppendEntry messages received
// when server is in CANDIDATE state
func (s *raftServer) handleAppendEntry(from int, ae *AppendEntry) bool {
	acc := false
	s.writeToLog("Received appendEntry message from " + strconv.Itoa(from) + " with term #" + strconv.FormatInt(ae.Term, TERM_BASE))
	if ae.Term >= s.Term() { // AppendEntry with same or larger term
		s.leaderId.Set(ae.LeaderId)
		if isHeartbeat(&ae.Entry) {
			// heartbeat
			acc = true
		} else if s.localLog.Exists(ae.PrevLogIndex) && s.localLog.Get(ae.PrevLogIndex).Term == ae.PrevLogTerm {
			if s.localLog.Exists(ae.Entry.Index) && s.localLog.Get(ae.Entry.Index).Term != ae.Entry.Term {
				// existing entry conflicts with the entry from the leader
				s.localLog.DiscardFrom(ae.Entry.Index)
			}
			// apply entry to local log
			s.localLog.Append(&ae.Entry)
			acc = true
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
			s.Outbox() <- s.localLog.Get(N).Data
			s.lastApplied.Set(N)
		}
		time.Sleep(1 * time.Millisecond) // since goroutines are cooperative
	}
}

func (s *raftServer) isMoreUpToDate(LastLogIndex int64, LastLogTerm int64) bool {
	latest := s.localLog.Tail()
	return LastLogTerm > latest.Term || LastLogTerm == latest.Term && LastLogIndex > latest.Index
}
