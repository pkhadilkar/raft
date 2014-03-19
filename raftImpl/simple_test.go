package raftImpl

import (
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/llog"
	"strconv"
	"testing"
	"time"
	"os"
)


// TestElect tests normal behavior that leader should
// be elected under normal condition and everyone
// should agree upon current leader
func TestElect(t *testing.T) {
	raftConf := &RaftConfig{MemberRegSocket: "127.0.0.1:9999", PeerSocket: "127.0.0.1:9009", TimeoutInMillis: 500, HbTimeoutInMillis: 50, LogDirectoryPath: "logs", StableStoreDirectoryPath: "./stable"}

	// delete stored state to avoid unnecessary effect on following test cases
	deleteState(raftConf.StableStoreDirectoryPath)

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
		logStore := llog.Create()

		s, err := NewWithConfig(clusterServer, logStore, raftConf)
		if err != nil {
			t.Errorf("Error in creating Raft servers. " + err.Error())
			return
		}
		raftServers[i] = s
	}

	// there should be a leader after sufficiently long duration
	time.Sleep(15 * time.Second)
	leaderId := raftServers[2].Leader()

	if leaderId == NONE {
		t.Errorf("No leader was chosen")
	}

	fmt.Println("Leader pid: " + strconv.Itoa(leaderId))

	// replicate an entry
	_ = "Woah, it works !!"
	
}

// deleteState deletes persistent state on
// the disk for each server
// parameters:
// baseDir: Path to base directory
// which contains state of all
// the servers on the disk
func deleteState(baseDir string) {
	base, err := os.Open(baseDir)
	if err != nil {
		fmt.Println("Error in opening directory.")
		return
	}
	fis, err := base.Readdir(-1) // read information for all files in the directory
	for _, f := range fis {
		err = os.Remove(baseDir + "/" + f.Name())
		if err != nil {
			fmt.Println("Error in deleting the file")
			fmt.Println(err.Error())
			return
		}
	}
	return
}
