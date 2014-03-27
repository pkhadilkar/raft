// Package utils contains helper data structures
// and functions
package utils

import (
	"sync"
)
// SyncIntIntMap is a synchronized implementation of
// map from int to int64. The data types are chosen
// for convenience of current implementation
type SyncIntIntMap struct {
	m            map[int]int64 // internal map
	sync.RWMutex               // mutex to synchronize access to map
}

func CreateSyncIntMap() *SyncIntIntMap {
	return &SyncIntIntMap{m: make(map[int]int64)}
}

func (s *SyncIntIntMap) Get(key int) (int64, bool) {
	s.RLock()
	defer s.RUnlock()
	val, found := s.m[key]
	return val, found
}

func (s *SyncIntIntMap) Set(key int, val int64) {
	s.Lock()
	defer s.Unlock()
	s.m[key] = val
}

func (s *SyncIntIntMap) Contains(key int) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.m[key]
	return ok
}
