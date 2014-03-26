package raftImpl

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/utils"
	"strconv"
	"time"
)

// leader's implementation for raft

// lead function is called when server is a leader
// assume that initial heartbeat has been sent
func (s *raftServer) lead() {
	s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
	// launch a goroutine to handle followersFormatInt(
	follower := s.followers()
	nextIndex, matchIndex := s.initLeader(follower)
	s.leaderId.Set(s.server.Pid())
	fmt.Println("Leader is " + strconv.Itoa(s.Pid()))

	go s.handleFollowers(follower, nextIndex, matchIndex)
	go s.updateLeaderCommitIndex(follower, matchIndex)
	for s.State() == LEADER {
		select {
		case <-s.hbTimeout.C:
			//s.writeToLog("Sending hearbeats")
			s.sendHeartBeat()
			s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
		case msg := <-s.outbox:
			// received message from state machine
			s.writeToLog("Received message from state machine layer")
			s.localLog.Append(&raft.LogEntry{Term: s.Term(), Data: msg})
		case e := <-s.server.Inbox():
			raftMsg := e.Msg
			if ae, ok := raftMsg.(AppendEntry); ok { // AppendEntry
				s.handleAppendEntry(e.Pid, &ae)
			} else if rv, ok := raftMsg.(RequestVote); ok { // RequestVote
				s.handleRequestVote(e.Pid, &rv)
			} else if entryReply, ok := raftMsg.(EntryReply); ok {
				n, found := nextIndex.Get(e.Pid)
				var m int64
				if !found {
					panic("Next index not found for follower " + strconv.Itoa(e.Pid))
				} else {
					m, found = matchIndex.Get(e.Pid)
					if !found {
						panic("Match index not found for follower " + strconv.Itoa(e.Pid))
					}
				}

				if entryReply.Success {
					// update nextIndex for follower
					if entryReply.LogIndex != HEARTBEAT {
						nextIndex.Set(e.Pid, max(n+1, entryReply.LogIndex+1))
						matchIndex.Set(e.Pid, max(m, entryReply.LogIndex))
						//s.writeToLog("Received confirmation from " + strconv.Itoa(e.Pid))
					}
				} else if s.Term() >= entryReply.Term {
					fmt.Println("Decrementing term. " + strconv.FormatInt(entryReply.Term, 10))
					nextIndex.Set(e.Pid, n-1)
				} else {
					s.setState(FOLLOWER)
					// There are no other goroutines active
					// at this point which modify term
					if s.Term() >= entryReply.Term {
						panic("Follower replied false even when Leader's term is not smaller")
					}
					s.setTerm(entryReply.Term)
					break
				}
			}
		}
	}

	s.hbTimeout.Stop()
}

// handleFollowers ensures that followers are informed
// about new messages and lagging followers catch up
func (s *raftServer) handleFollowers(followers []int, nextIndex *utils.SyncIntIntMap, matchIndex *utils.SyncIntIntMap) {
	for s.State() == LEADER {
		for _, f := range followers {
			lastIndex := s.localLog.TailIndex()
			n, ok := nextIndex.Get(f)
			if !ok {
				panic("nextIndex not found for follower " + strconv.Itoa(f))
			}
			if lastIndex != 0 && lastIndex >= n {
				// send a new AppendEntry
				prevIndex := n - 1
				//fmt.Println("Follower: " + strconv.Itoa(f) + "LastIndex: " + strconv.FormatInt(lastIndex, 10))
				//fmt.Println("prevIndex: " + strconv.FormatInt(prevIndex, 10))
				var prevTerm int64 = -1
				// n = 0 when we add first entry to the log
				if prevIndex != -1 {
					prevTerm = s.localLog.Get(prevIndex).Term
				}
				ae := &AppendEntry{Term: s.Term(), LeaderId: s.server.Pid(), PrevLogIndex: prevIndex, PrevLogTerm: prevTerm}
				ae.LeaderCommit = s.commitIndex.Get()
				ae.Entry = *s.localLog.Get(n)
				s.writeToLog("Replicating entry " + strconv.FormatInt(n, 10))
				s.server.Outbox() <- &cluster.Envelope{Pid: f, Msg: ae}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// followers returns a slice of follower's pids
// TODO: Error handling
func (s *raftServer) followers() []int {
	peers := s.server.Peers()
	follower := make([]int, len(peers)-1)
	i := 0
	for _, srvr := range peers {
		if srvr != s.server.Pid() {
			follower[i] = srvr
			i++
		}
	}

	return follower
}

// respondToClient replies to the client when
// an entry is replicated on majority of servers
func (s *raftServer) updateLeaderCommitIndex(followers []int, matchIndex *utils.SyncIntIntMap) {

	for s.State() == LEADER {
		N := s.commitIndex.Get() + 1
		upto := N + 1

		for N <= upto {

			if !s.localLog.Exists(N) {
				break
			}

			i := 1
			for _, f := range followers {
				if j, _ := matchIndex.Get(f); j >= N {
					i++
					upto = max(upto, j)
				}
			}
			// followers do not include Leader
			if entry := s.localLog.Get(N); i > (len(followers)+1)/2 && entry.Term == s.Term() {
				s.writeToLog("Updating commitIndex to " + strconv.FormatInt(N, 10))
				s.commitIndex.Set(N)
			}
			N++
		}
		time.Sleep(NICE * time.Millisecond)
	}
}

// sendHeartBeat sends heartbeat messages to followers
func (s *raftServer) sendHeartBeat() {
	ae := &AppendEntry{Term: s.Term(), LeaderId: s.server.Pid()}
	ae.LeaderCommit = s.commitIndex.Get()
	e := &cluster.Envelope{Pid: cluster.BROADCAST, Msg: ae}
	s.server.Outbox() <- e
}

// initLeader initializes important leader data structures
func (s *raftServer) initLeader(followers []int) (*utils.SyncIntIntMap, *utils.SyncIntIntMap) {
	nextIndex := utils.CreateSyncIntMap()
	matchIndex := utils.CreateSyncIntMap()
	nextLogEntry := s.localLog.TailIndex() + 1
	for _, f := range followers {
		nextIndex.Set(f, nextLogEntry)
		matchIndex.Set(f, 0)
	}
	return nextIndex, matchIndex
}
