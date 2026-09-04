package vz

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestVirtualMachineLeavesNoGoroutines pins that a collected machine
// leaves none of its notification goroutines behind, whether or not
// anyone subscribed, and that a channel it handed out is closed.
func TestVirtualMachineLeavesNoGoroutines(t *testing.T) {
	const bufferGoroutine = "github.com/Code-Hex/go-infinity-channel.(*Channel[...]).infiniteBuffer("
	before := countGoroutines(bufferGoroutine)

	// One machine nobody listens to, dropped on the spot.
	if _, err := NewVirtualMachine(newTestConfig(t)); err != nil {
		t.Fatal(err)
	}
	// One machine with both channels handed out.
	subscribed, err := NewVirtualMachine(newTestConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	states := subscribed.StateChangedNotify()
	disconnects, err := subscribed.NetworkDeviceAttachmentWasDisconnected()
	if err != nil {
		t.Fatal(err)
	}
	subscribed = nil

	deadline := time.Now().Add(10 * time.Second)
	for countGoroutines(bufferGoroutine) > before {
		if time.Now().After(deadline) {
			t.Fatalf("%d notification goroutines still alive 10s after the machines were dropped",
				countGoroutines(bufferGoroutine)-before)
		}
		runtime.GC()
		time.Sleep(20 * time.Millisecond)
	}
	if _, ok := <-states; ok {
		t.Fatal("StateChangedNotify channel still open after the machine was collected")
	}
	if _, ok := <-disconnects; ok {
		t.Fatal("NetworkDeviceAttachmentWasDisconnected channel still open after the machine was collected")
	}
}

// countGoroutines counts the goroutines with frame on their stack.
func countGoroutines(frame string) int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return strings.Count(string(buf[:n]), frame)
		}
		buf = make([]byte, 2*len(buf))
	}
}
