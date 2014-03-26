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
	mutex sync.RWMutex
}

// Get returns value of AtomicI64
func (a *AtomicI64) Get() int64 {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.Value
}

// Set sets a new value for AtomicI64 object
func (a *AtomicI64) Set(num int64) {
	a.mutex.Lock()
	a.Value = num
	a.mutex.Unlock()
}

// Incr increments value by 1
func (a *AtomicI64) Incr() {
	a.mutex.Lock()
	a.Value += 1
	a.mutex.Unlock()
}

// Decr decrements value by 1
func (a *AtomicI64) Decr() {
	a.mutex.Lock()
	a.Value -= 1
	a.mutex.Unlock()
}

// AtomicInt is an int protected by a mutex
type AtomicInt struct {
	Value int
	mutex sync.RWMutex
}

// Get returns Value of AtomicInt
func (a *AtomicInt) Get() int {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.Value
}

// Set sets a new Value for AtomicInt object
func (a *AtomicInt) Set(num int) {
	a.mutex.Lock()
	a.Value = num
	a.mutex.Unlock()
}
