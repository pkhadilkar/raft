package utils

import (
	"sync"
)

// AtomicI64 is an int64 data element
// which is access protected by mutex.
// Atomic access is guranteed only
// through Get and Set methods
type AtomicI64 struct {
	Value int64
	sync.RWMutex
}

// Get returns value of AtomicI64
func (a *AtomicI64) Get() int64 {
	a.RLock()
	defer a.RUnlock()
	return a.Value
}

// Set sets a new value for AtomicI64 object
func (a *AtomicI64) Set(num int64) {
	a.Lock()
	a.Value = num
	a.Unlock()
}

// AtomicInt is an int protected by a mutex
type AtomicInt struct {
	Value int
	sync.RWMutex
}

// Get returns Value of AtomicI64
func (a *AtomicInt) Get() int {
	a.RLock()
	defer a.RUnlock()
	return a.Value
}

// Set sets a new Value for AtomicI64 object
func (a *AtomicInt) Set(num int) {
	a.Lock()
	a.Value = num
	a.Unlock()
}
