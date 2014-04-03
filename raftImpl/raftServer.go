// Package raftImpl implements Raft server protocol
// It provides replicated log abstraction.

package raftImpl

import (
	"encoding/gob"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/llog"
	"github.com/pkhadilkar/raft/utils"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	bufferSize = 100
	NotVoted   = -1
	NONE       = -1
	HEARTBEAT  = -2 // LogIndex to send when replying to a heartbeat message
)

//TODO: Use separate locks for state and term

// constants for states of the server
const (
	FOLLOWER = iota
	LEADER
	CANDIDATE
)

// base used in converting term to string
const TERM_BASE = 10

// TODO: store pointer to raftConfig rather than storing all fields in raftServer
// raftServer is a concrete implementation of raft Interface
type raftServer struct {
	currentState *utils.AtomicInt // current state of the server
	eTimeout     *time.Timer      // timer for election timeout
	hbTimeout    *time.Timer      // timer to send periodic hearbeats
	server       cluster.Server   // cluster server that provides message send/ receive functionality
	log          *log.Logger      // logger for server to store log messages
	rng          *rand.Rand
	state        *PersistentState    // server information that should be persisted
	config       *RaftConfig         // config information for raftServer
	sync.RWMutex                     // mutex to protect the state
	inbox        chan *raft.LogEntry // inbox for raft
	outbox       chan interface{}    // outbox for raft, inbox for upper layer
	localLog     llog.LogStore       // LogStore dependency. Used to  implement shared log abstraction
	commitIndex  *utils.AtomicI64    // index of the highest entry known to be committed
	lastApplied  *utils.AtomicI64    // index of the last entry applied to the log
	leaderId     *utils.AtomicInt    // leader's PID
}

// Term returns current term of a raft server
func (s *raftServer) Term() int64 {
	return s.state.Term.Get()
}

// IsLeader returns true if r is a leader and false otherwise
func (s *raftServer) isLeader() bool {
	s.Lock()
	defer s.Unlock()
	return s.currentState.Get() == LEADER
}

func (s *raftServer) DiscardUpto(index int64) {

}

func (s *raftServer) Inbox() <-chan *raft.LogEntry {
	return s.inbox
}

func (s *raftServer) Outbox() chan<- interface{} {
	return s.outbox
}

// returns pid of server
func (s *raftServer) Leader() int {
	return s.leaderId.Get()
}

// SetTerm sets the current term of the server
// The changes are persisted on disk
func (s *raftServer) setTerm(term int64) {
	// when term changes, reset votedFor
	s.state.VotedFor.Set(NotVoted)
	s.state.Term.Set(term)
	s.persistState()
}

// voteFor maintains the server side state
// to ensure that the server persists vote
// Parameters:
//  pid  : pid of the server to whom vote is given
//  term : term for which the vote was granted
func (s *raftServer) voteFor(pid int, term int64) {
	s.state.VotedFor.Set(pid)
	s.state.Term.Set(term)
	//TODO: Force this on stable storage
	s.persistState()
}

// persistState persists the state of the server on
// stable storage. This function panicks if it cannot
// write the state.
func (s *raftServer) persistState() {
	s.Lock()
	defer s.Unlock()
	pStateBytes, err := PersistentStateToBytes(s.state)
	if err != nil {
		//TODO: Add the state that was encoded. This might help in case of an error
		panic("Cannot encode PersistentState")
	}
	err = ioutil.WriteFile(s.config.StableStoreDirectoryPath+"/"+ServerFileName(s.server.Pid()), pStateBytes, UserReadWriteMode)
	if err != nil {
		panic("Could not persist state to storage on file " + s.config.StableStoreDirectoryPath)
	}
}

// readPersistentState tries to read the persistent
// state from the stable storage. It does not panic if
// no record of state is found in stable storage as
// that represents a valid state when server is  started
// for the first time
func (s *raftServer) readPersistentState() {
	pStateRead, err := ReadPersistentState(s.config.StableStoreDirectoryPath + "/" + ServerFileName(s.server.Pid()))
	if err != nil {
		vf := &utils.AtomicInt{Value: NotVoted}
		t := &utils.AtomicI64{Value: 0}
		s.state = &PersistentState{VotedFor: vf, Term: t}
		return
	}
	s.Lock()
	s.state = pStateRead
	s.Unlock()
}

// state returns the current state of the server
// value returned is one of the FOLLOWER,
// CANDIDATE and LEADER
func (s *raftServer) State() int {
	return s.currentState.Get()
}

// votedFor returns the pid of the server
// voted for by current server
func (s *raftServer) VotedFor() int {
	return s.state.VotedFor.Get()
}

// setState sets the state of the server to newState
// TODO: Add verification for newState
func (s *raftServer) setState(newState int) {
	s.currentState.Set(newState)
}

func (s *raftServer) Pid() int {
	return s.server.Pid()
}

// incrTerm  increments server's term
func (s *raftServer) incrTerm() {
	s.state.VotedFor.Set(NotVoted)
	s.state.Term.Set(s.state.Term.Get() + 1)
	s.persistState()
}

func New(clusterServer cluster.Server, l llog.LogStore, configFile string) (raft.Raft, error) {
	raftConfig, err := ReadConfig(configFile)
	if err != nil {
		fmt.Println("Error in reading config file.")
		return nil, err
	}
	// NewWithConfig(pid int, ip string, port int, raftConfig *RaftConfig) (Raft, error) {
	return NewWithConfig(clusterServer, l, raftConfig)
}

// function getLog creates a log for a raftServer
func getLog(s *raftServer, logDirPath string) error {
	f, err := os.Create(logDirPath + "/" + strconv.Itoa(s.server.Pid()) + ".log")
	if err != nil {
		fmt.Println("Error: Cannot create log files")
		return err
	}
	s.log = log.New(f, "", log.LstdFlags)
	return err
}

// NewWithConfig creates a new raftServer.
// Parameters:
//  clusterServer : Server object of cluster API. cluster.Server provides message send
//                  receive along with other facilities such as finding peers
//  raftConfig    : Raft configuration object
func NewWithConfig(clusterServer cluster.Server, l llog.LogStore, raftConfig *RaftConfig) (raft.Raft, error) {
	s := raftServer{currentState: &utils.AtomicInt{Value: FOLLOWER}, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	s.server = clusterServer
	s.config = raftConfig
	s.localLog = l
	s.commitIndex = &utils.AtomicI64{}
	s.lastApplied = &utils.AtomicI64{}
	s.leaderId = &utils.AtomicInt{Value: NONE}
	s.inbox = make(chan *raft.LogEntry, bufferSize)
	s.outbox = make(chan interface{}, bufferSize)
	// read persistent state from the disk if server was being restarted as a
	// part of recovery then it would find the persistent state on the disk
	s.readPersistentState()
	s.eTimeout = time.NewTimer(time.Duration(s.config.TimeoutInMillis+s.rng.Int63n(150)) * time.Millisecond) // start timer
	s.hbTimeout = time.NewTimer(time.Duration(s.config.TimeoutInMillis) * time.Millisecond)
	s.hbTimeout.Stop()
	s.state.VotedFor.Set(NotVoted)

	err := getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}
	
	go s.redeliverLogEntries()
	go s.serve()
	return raft.Raft(&s), err
}

// registerMessageTypes registers Message types used by
// server to gob. This is required because messages are
// defined as interfaces in cluster.Envelope
func init() {
	gob.Register(AppendEntry{})
	gob.Register(EntryReply{})
	gob.Register(GrantVote{})
	gob.Register(RequestVote{})
	// LogEntry is used in Log
	gob.Register(raft.LogEntry{})
}
