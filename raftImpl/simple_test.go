package raftImpl

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/llog"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestElect tests normal behavior that leader should
// be elected under normal condition and everyone
// should agree upon current leader
func TestElect(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 1500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable", RaftLogDirectoryPath: "../LocalLog"}

	// delete stored state to avoid unnecessary effect on following test cases
	initState(raftConf.StableStoreDirectoryPath, raftConf.LogDirectoryPath)

	// launch cluster proxy servers
	cluster.NewProxyWithConfig(RaftToClusterConf(raftConf))

	fmt.Println("Started Proxy")

	time.Sleep(100 * time.Millisecond)

	serverCount := 5
	raftServers := make([]raft.Raft, serverCount+1)

	for i := 1; i <= serverCount; i += 1 {
		// create cluster.Server
		clusterServer, err := cluster.NewWithConfig(i, "127.0.0.1", 5000+i, RaftToClusterConf(raftConf))
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
	data := "Woah, it works !!"
	leader := raftServers[leaderId]

	leader.Outbox() <- data

	var follower raft.Raft = nil

	for i, server := range raftServers {
		if i > 0 && server.Pid() != leaderId {
			follower = server
			break
		}
	}

	select {
	case msg := <-follower.Inbox():
		fmt.Println("Received message: ")
		fmt.Println(msg)
	case <-time.After(15 * time.Second):
		fmt.Println("Message replication took more than 15 seconds !")

	}

}


func initState(stableStoreBaseDir string, logBaseDir string) {
	err := os.RemoveAll(stableStoreBaseDir)
	if err != nil {
		panic("Cannot remove " + stableStoreBaseDir)
	}

	err = os.RemoveAll(logBaseDir)

	if err != nil {
		panic("Cannot remove " + logBaseDir)
	}

	err = os.Mkdir(stableStoreBaseDir, os.ModeDir | 0764)
	if err != nil {
		panic("Cannot create " + stableStoreBaseDir + "." + err.Error())
	}

	err = os.Mkdir(logBaseDir, os.ModeDir | 0764)
	if err != nil {
		panic("Cannot create " + logBaseDir + "." + err.Error())
	}
}
