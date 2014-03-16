// Package raftImpl implements Raft server protocol
// It provides replicated log abstraction.

package raftImpl

import (
	"encoding/gob"
	"fmt"
	"github.com/pkhadilkar/cluster"
	"github.com/pkhadilkar/raft"
	"github.com/pkhadilkar/raft/llog"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

const bufferSize = 100

const NotVoted = -1

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
	currentState int            // current state of the server
	eTimeout     *time.Timer    // timer for election timeout
	hbTimeout    *time.Timer    // timer to send periodic hearbeats
	server       cluster.Server // cluster server that provides message send/ receive functionality
	log          *log.Logger    // logger for server to store log messages
	rng          *rand.Rand
	state        *PersistentState    // server information that should be persisted
	config       *RaftConfig         // config information for raftServer
	sync.RWMutex                     // mutex to protect the state
	inbox        chan interface{}    // inbox for raft
	outbox       chan *raft.LogEntry // outbox for raft, inbox for upper layer
	localLog     *llog.LogStore      // LogStore dependency. Used to  implement shared log abstraction
	commitIndex  int64               // index of the highest entry known to be committed
	lastApplied  int64               // index of the last entry applied to the log
}

// Term returns current term of a raft server
func (s *raftServer) Term() int64 {
	s.Lock()
	currentTerm := s.state.Term
	s.Unlock()
	return currentTerm
}

// IsLeader returns true if r is a leader and false otherwise
func (s *raftServer) isLeader() bool {
	s.Lock()
	defer s.Unlock()
	return s.currentState == LEADER
}

func (s *raftServer) DiscardUpto(index int64) {

}

func (s *raftServer) Inbox() chan<- interface{} {
	return s.inbox
}

func (s *raftServer) Outbox() <-chan *raft.LogEntry {
	return s.outbox
}

// returns pid of server
func (s *raftServer) Leader() int {
	return -1
}

// SetTerm sets the current term of the server
// The changes are persisted on disk
func (s *raftServer) setTerm(term int64) {
	s.state.Term = term
	s.persistState()
}

// voteFor maintains the server side state
// to ensure that the server persists vote
// Parameters:
//  pid  : pid of the server to whom vote is given
//  term : term for which the vote was granted
func (s *raftServer) voteFor(pid int, term int64) {
	s.state.VotedFor = pid
	s.state.Term = term
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
		s.state = &PersistentState{VotedFor: NotVoted, Term: 0}
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
	s.Lock()
	defer s.Unlock()
	return s.currentState
}

// votedFor returns the pid of the server
// voted for by current server
func (s *raftServer) VotedFor() int {
	s.Lock()
	defer s.Unlock()
	return s.state.VotedFor
}

// setState sets the state of the server to newState
// TODO: Add verification for newState
func (s *raftServer) setState(newState int) {
	s.Lock()
	s.currentState = newState
	s.Unlock()
}

func (s *raftServer) Pid() int {
	return s.server.Pid()
}

func (s *raftServer) CommitIndex() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.commitIndex
}

func (s *raftServer) setCommitIndex(index int64) {
	s.Lock()
	s.commitIndex = index
	s.Unlock()
}

func (s *raftServer) LastApplied() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.lastApplied
}

func (s *raftServer) setLastApplied(index int64) {
	s.Lock()
	s.lastApplied = index
	s.Unlock()
}

// incrTerm  increments server's term
func (s *raftServer) incrTerm() {
	s.Lock()
	s.state.Term++
	s.Unlock()
	// TODO: Is persistState necessary here ?
	s.persistState()
}

//TODO: Load current term from persistent storage
func New(clusterServer cluster.Server, l *llog.LogStore, configFile string) (raft.Raft, error) {
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
func NewWithConfig(clusterServer cluster.Server, l *llog.LogStore, raftConfig *RaftConfig) (raft.Raft, error) {
	s := raftServer{currentState: FOLLOWER, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
	s.server = clusterServer
	s.config = raftConfig
	s.localLog = l
	// read persistent state from the disk if server was being restarted as a
	// part of recovery then it would find the persistent state on the disk
	s.readPersistentState()
	s.eTimeout = time.NewTimer(time.Duration(s.config.TimeoutInMillis+s.rng.Int63n(150)) * time.Millisecond) // start timer
	s.hbTimeout = time.NewTimer(time.Duration(s.config.TimeoutInMillis) * time.Millisecond)
	s.hbTimeout.Stop()
	s.state.VotedFor = NotVoted

	err := getLog(&s, raftConfig.LogDirectoryPath)
	if err != nil {
		return nil, err
	}

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
}
