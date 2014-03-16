package raftImpl

import (
	"github.com/pkhadilkar/raft/utils"
	"time"
	"github.com/pkhadilkar/raft"
//	"fmt"
)

// this file contains leader's implementation for raft

// lead function is called when server is a leader
// assume that initial heartbeat has been sent
func (s *raftServer) lead() {
	s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
	// launch a goroutine to handle followersFormatInt(
	follower := s.followers()
	nextIndex, matchIndex := s.initLeader(follower)
	go s.handleFollowers(follower, nextIndex, matchIndex)
	for s.State() == LEADER {
		select {
		case <-s.hbTimeout.C:
			s.writeToLog("Sending hearbeats")
			s.sendHeartBeat()
			s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
		case msg := <-s.inbox:
			// received message from state machine
			s.localLog.Append(&raft.LogEntry{Term: s.Term(), Data: msg})
		case e := <- s.server.Inbox():
			raftMsg := e.Msg
			if ae, ok := raftMsg.(AppendEntry); ok { // AppendEntry
				s.handleAppendEntry(e.Pid, &ae)
			} else if rv, ok := raftMsg.(RequestVote); ok { // RequestVote
				s.handleRequestVote(e.Pid, &rv)
			}
		}
	}
}

// handleFollowers ensures that followers are informed
// about new messages and lagging followers catch up
func (s *raftServer) handleFollowers(followers []int, nextIndex *utils.SyncIntIntMap, matchIndex *utils.SyncIntIntMap) {
	/*for f, _ := range followers {
		lastIndex = s.localLog.TailIndex()
		n, ok := nextIndex.Get(f)
		if !ok {
			panic("nextIndex not found for follower " + strconv.Itoa(f))
		}
		if lastIndex >= n {
			// send a new AppendEntry
			prevIndex = n - 1
			prevTerm = s.localLog.Get(prevIndex).Term
			ae := &AppendEntry{Term: s.Term(), LeaderId: s.server.Pid(), prevLogIndex: prevIndex, prevLogTerm: prevTerm}
			ae.leaderCommit = 
			e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: ae}
		}
	}*/
}

// followers returns a slice of follower's pids
// TODO: Error handling
func (s *raftServer) followers() []int {
	peers := s.server.Peers()
	follower := make([]int, len(peers))
	i := 0;
	for _, srvr := range peers {
		if srvr != s.server.Pid() {
			follower[i] = srvr
			i++
		}
	}
	return follower
}

// initLeader initializes important leader data structures
func (s *raftServer) initLeader(followers []int) (*utils.SyncIntIntMap, *utils.SyncIntIntMap) {
	nextIndex := utils.CreateSyncIntMap()
	matchIndex := utils.CreateSyncIntMap()
	nextLogEntry := s.localLog.TailIndex()
	for _, f := range followers {
		nextIndex.Set(f, nextLogEntry)
		matchIndex.Set(f, 0)
	}
	return nextIndex, matchIndex
}
