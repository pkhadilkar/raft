package llog

import (
	"fmt"
	"github.com/pkhadilkar/raft"
	"testing"
)

func TestDefault(t *testing.T) {
	l, err := Create("/tmp/localLog")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	to := int64(100)
	for i := int64(1); i <= to; i += 1 {
		l.Append(&raft.LogEntry{Term: i / 10, Index: i, Data: "Add"})
	}

	// get
	if entry := l.Get(9); entry.Term != 0 || entry.Index != 9 {
		fmt.Println(entry)
		t.Errorf("Could not retrieve entry appended to log previously")
	}

	if l.TailIndex() != to {
		t.Errorf("Incorrect tail index")
	}

	l.DiscardFrom(to / 2)

	if l.TailIndex() != (to/2)-1 {
		t.Errorf("Error in DiscardFrom")
	}
}
