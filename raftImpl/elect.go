package raftImpl

import (
	"github.com/pkhadilkar/cluster"
	"strconv"
	"time"
)

// RandomTimeoutRange indicates number of milliseconds
// randome timeout should vary from its base value
const RandomTimeoutRange = 150

// replyTo replies to sender of envelope
// msg is the reply content. Pid in the
// is used to identify the sender
func (s *raftServer) replyTo(to int, msg interface{}) {
	reply := &cluster.Envelope{Pid: to, Msg: msg}
	s.server.Outbox() <- reply
}

// serve is the main goroutine for Raft server
// This will serve as a "main" routine for a RaftServer.
// RaftServer's FOLLOWER and LEADER states are handled
// in this server routine.
func (s *raftServer) serve() {
	s.writeToLog("Started serve")
	go s.logApply()
	for {
		select {
		case e := <-s.server.Inbox():
			// received a message on server's inbox
			msg := e.Msg
			if ae, ok := msg.(AppendEntry); ok { // AppendEntry
				acc := s.handleAppendEntry(e.Pid, &ae)
				if acc {
					s.resetElectionTimeout()
				}
			} else if rv, ok := msg.(RequestVote); ok { // RequestVote
				s.handleRequestVote(e.Pid, &rv) // reset election timeout here too ? To avoid concurrent elections ?
			}

		case _ = <-s.outbox:
			// non leaders ignore the message from state machine
			// TODO: Send error entry with leader's Pid to client

		case <-s.eTimeout.C:
			// received timeout on election timer
			s.writeToLog("Starting Election")
			// TODO: Use glog
			s.startElection()
			s.writeToLog("Election completed")
			if s.isLeader() {
				s.eTimeout.Stop() // leader should not time out for election
				s.lead()
			}
		}
	}
}

// startElection handles election component of the
// raft server. Server stays in this function till
// it is in candidate state
func (s *raftServer) startElection() {
	s.setState(CANDIDATE)
	peers := s.server.Peers()
	s.writeToLog("Number of peers: " + strconv.Itoa(len(peers)))
	votes := make(map[int]bool) // map to store received votes
	votes[s.server.Pid()] = true
	//s.voteFor(s.server.Pid(), s.Term())
	for s.State() == CANDIDATE {
		s.incrTerm() // increment term for current
		s.voteFor(s.server.Pid(), s.Term())
		candidateTimeout := time.Duration(s.config.TimeoutInMillis + s.rng.Int63n(RandomTimeoutRange)) // random timeout used by Raft authors
		s.sendRequestVote()
		s.writeToLog("Sent RequestVote message " + strconv.Itoa(int(candidateTimeout)))
		s.eTimeout.Reset(candidateTimeout * time.Millisecond) // start re-election timer
		for {
			acc := false
			select {
			case e, _ := <-s.server.Inbox():
				// received a message on server's inbox
				msg := e.Msg
				if ae, ok := msg.(AppendEntry); ok { // AppendEntry
					acc = s.handleAppendEntry(e.Pid, &ae)
				} else if rv, ok := msg.(RequestVote); ok { // RequestVote
					acc = s.handleRequestVote(e.Pid, &rv)

				} else if grantV, ok := msg.(GrantVote); ok && grantV.VoteGranted {
					votes[e.Pid] = true
					s.writeToLog("Received grantVote message from " + strconv.Itoa(e.Pid) + " with term #" + strconv.FormatInt(grantV.Term, TERM_BASE))
					s.writeToLog("Votes received so far " + strconv.Itoa(len(votes)))
					if len(votes) == len(peers)/2+1 { // received majority votes
						s.setState(LEADER)
						s.sendHeartBeat()
						acc = true
					}
				}
			case <-s.eTimeout.C:
				// received timeout on election timer
				s.writeToLog("Received re-election timeout")
				acc = true
			default:
				time.Sleep(NICE * time.Millisecond)
			}

			if acc {
				break
			}
		}
	}
}
