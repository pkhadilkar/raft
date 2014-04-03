package raftImpl

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/llog"
	"strconv"
	"testing"
	"time"
)

// TestReplicateScale tests that 1000 messages
// are replicated in reasonable amount of time
// Reasonable is considering 5 servers doing
// disk IO for each message
func TestReplicateScale(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:8145", PeerSocket: "127.0.0.1:9100", TimeoutInMillis: 1500, HbTimeoutInMillis: 100, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable", RaftLogDirectoryPath: "../LocalLog"}

	// delete stored state to avoid unnecessary effect on following test cases
	initState(raftConf.StableStoreDirectoryPath, raftConf.LogDirectoryPath, raftConf.RaftLogDirectoryPath)

	/*if raftConf.MemberRegSocket == "127.0.0.1:8145" {
		fmt.Println("Returning")
		return
	}*/

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]raft.Raft, serverCount+1)

	for i := 1; i <= serverCount; i += 1 {
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 7000+i, RaftToClusterConf(raftConf))
		if err != nil {
			t.Errorf("Error in creating cluster server. " + err.Error())
			return
		}

		logStore, err := llog.Create(raftConf.RaftLogDirectoryPath + "/" + strconv.Itoa(i))
		if err != nil {
			t.Errorf("Error in creating log. " + err.Error())
			return
		}

		s, err := NewWithConfig(clusterServer, logStore, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
			return
		}
		raftServers[i] = s
	}

	// there should be a leader after sufficiently long duration
	time.Sleep(5 * time.Second)
	leaderId := raftServers[2].Leader()

	if leaderId == NONE {
		t.Errorf("No leader was chosen")
		return
	}

	fmt.Println("Leader pid: " + strconv.Itoa(leaderId))

	// replicate an entry
	// data := "Woah, it works !!"
	leader := raftServers[leaderId]

	var follower raft.Raft = nil

	for i, server := range raftServers {
		if i > 0 && server.Pid() != leaderId {
			follower = server
			fmt.Println("Follower is " + strconv.Itoa(follower.Pid()))
			break
		}
	}

	count := 100

	// launch a goroutine to consume messages for other followers and leader
	for i, server := range raftServers {
		if i > 0 && server.Pid() != follower.Pid() {
			go readMessages(server)
		}
	}

	done := make(chan bool, 1)

	go replicateVerification(follower, count, done)

	for count != 0 {
		leader.Outbox() <- strconv.Itoa(count)
		count -= 1
	}

	select {
	case <-done:
	case <-time.After(5 * time.Minute):
		t.Errorf("Message replication took more than 5 minutes !")

	}
}

func readMessages(s raft.Raft) {
	for {
		<-s.Inbox()
	}
}

func replicateVerification(s raft.Raft, count int, done chan bool) {
	for count > 0 {
		<-s.Inbox()
		count--
		if count%100 == 0 {
			fmt.Println(count)
		}
	}
	done <- true
}
