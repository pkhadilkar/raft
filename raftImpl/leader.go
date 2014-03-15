package raftImpl

// this file contains leader's implementation for raft

// lead function is called when server is a leader
// assume that initial heartbeat has been sent
func (s *raftServer) lead() {
/*	s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
	select {
		case <-s.hbTimeout.C:
			s.writeToLog("Sending hearbeats")
			s.sendHeartBeat()
			s.hbTimeout.Reset(time.Duration(s.config.HbTimeoutInMillis) * time.Millisecond)
	case msg := <- s.inbox:
		// received message from state machine
		s.localLog.Append(&LogEntry{Term: s.Term(), Data: msg})
	}
*/
}
