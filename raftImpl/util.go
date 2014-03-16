package raftImpl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"io/ioutil"
	"strconv"
	"github.com/pkhadilkar/raft/utils"
)

// Config struct represents all config information
// required to start a server. It represents
// information in config file in structure
type RaftConfig struct {
	MemberRegSocket          string // socket to connect to , to register a cluster server
	PeerSocket               string // socket to connect to , to get a list of cluster peers
	TimeoutInMillis          int64  // timeout duration to start a new Raft election
	HbTimeoutInMillis        int64  // timeout to sent periodic heartbeats
	LogDirectoryPath         string // path to log directory
	StableStoreDirectoryPath string // path to directory that can be used to store persistent information
}

type PersistentState struct {
	Term     *utils.AtomicI64 // last term seen by the server
	VotedFor *utils.AtomicInt   // pid of the server voted for
}

// ReadConfig reads configuration file information into Config object
// parameters:
// path : Path to config file
func ReadConfig(path string) (*RaftConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf RaftConfig
	err = json.Unmarshal(data, &conf)
	if err != nil {
		fmt.Println("Error", err.Error())
		return nil, errors.New("Incorrect format in config file.\n" + err.Error())
	}
	return &conf, err
}

func RaftToClusterConf(r *RaftConfig) *cluster.Config {
	return &cluster.Config{MemberRegSocket: r.MemberRegSocket, PeerSocket: r.PeerSocket}
}

// writeToLog writes a formatted message to log
// It specifically adds server details to log
func (s *raftServer) writeToLog(msg string) {
	s.log.Println(strconv.Itoa(s.server.Pid()) + ": #" + strconv.FormatInt(s.Term(), TERM_BASE) + ":" + msg)
}

// returns name of the file that is used
// to store persistent state of the
// server on the stable storage
func ServerFileName(pid int) string {
	return strconv.Itoa(pid)
}
