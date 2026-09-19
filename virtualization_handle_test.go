package vz

import (
	"runtime"
	"testing"
	"time"

	"github.com/Code-Hex/vz/v3/internal/objc"
)

// TestStateChangeAfterWrapperCollected pins that a machine which
// outlives its Go wrapper can still change state without killing the
// process. The framework mid-transition, a completion block or an
// autorelease pool nothing drains can all keep the Objective-C machine
// alive past the wrapper's finalizer; the extra retain stands in for
// them. A state change delivered into the deleted handle panics inside
// the cgo callback, which takes the whole test binary down.
func TestStateChangeAfterWrapperCollected(t *testing.T) {
	config := newTestConfig(t)
	m, err := NewVirtualMachine(config)
	if err != nil {
		t.Fatal(err)
	}
	objc.Retain(m)
	// alias drives the machine once the wrapper is gone. It carries no
	// finalizer; the machine retains its own queue.
	alias := &VirtualMachine{
		pointer:       objc.NewPointer(objc.Ptr(m)),
		dispatchQueue: m.dispatchQueue,
		config:        config,
	}
	states := m.StateChangedNotify()
	m = nil

	// The finalizer closes the channel last, so a closed channel means
	// it has run to the end.
	deadline := time.After(10 * time.Second)
collect:
	for {
		select {
		case _, ok := <-states:
			if !ok {
				break collect
			}
		case <-deadline:
			t.Fatal("wrapper not collected within 10s")
		default:
			runtime.GC()
			time.Sleep(20 * time.Millisecond)
		}
	}

	// The empty kernel cannot boot; the attempt alone moves the machine
	// through Starting, which is the state change under test.
	dropped := droppedStateChanges.Load()
	_ = alias.Start()
	if n := droppedStateChanges.Load() - dropped; n != 0 {
		t.Fatalf("%d state changes reached the collected machine's handle; the observer must be gone before the handle is", n)
	}

	// Dropping the last reference deallocates the machine, which must
	// not remove the observer a second time.
	objc.Release(alias)
}
