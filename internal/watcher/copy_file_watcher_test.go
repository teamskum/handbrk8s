package watcher

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// An atomic counter
type counter struct {
	val int32
}

func (c *counter) increment() {
	atomic.AddInt32(&c.val, 1)
}

func (c *counter) value() int32 {
	return atomic.LoadInt32(&c.val)
}

func TestCopyFileWatcher(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatalf("%#v", err)
	}
	defer os.RemoveAll(tmpDir)
	fmt.Println("Watching ", tmpDir)

	w, err := NewCopyFileWatcher(tmpDir)

	// Track how many times an event is raised
	var gotEvents counter
	done := make(chan bool)
	go func() {
		for e := range w.Events {
			fmt.Println(e)
			gotEvents.increment()
		}

		// Stop the goroutine once the events has been closed
		done <- true
	}()

	// Create a file in the watched directory
	tmpfile := filepath.Join(tmpDir, "foo.txt")
	f, err := os.Create(tmpfile)
	if err != nil {
		t.Fatalf("%#v", err)
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("%#v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Write to it a few times
	for i := 0; i < 10; i++ {
		if gotEvents.value() > 0 {
			t.Fatalf("expected no events to be raised until the StableThreshold has been reached but got %d events", gotEvents)
		}

		f, err = os.OpenFile(tmpfile, os.O_WRONLY, 0666)
		if err != nil {
			t.Fatalf("%#v", err)
		}
		_, err = f.WriteString(fmt.Sprintf("%d", i))
		if err != nil {
			t.Fatalf("%#v", err)
		}

		err = f.Sync()
		if err != nil {
			t.Fatalf("%#v", err)
		}

		err = f.Close()
		if err != nil {
			t.Fatalf("%#v", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Give the file time to be considered stable
	time.Sleep(w.StableThreshold)

	// Stop listening for events
	err = w.Close()
	if err != nil {
		t.Fatalf("%#v", err)
	}

	// Wait for all the events to be processed
	fmt.Println("Wait for all events to be processed")
	<-done

	var wantEvents int32 = 1
	if gotEvents.value() != wantEvents {
		t.Fatalf("expected %d events, got %d", wantEvents, gotEvents)
	}
}