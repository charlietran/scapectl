// Package audio observes system audio defaults without changing them.
package audio

import (
	"log"
	"sync"
)

var defaultKinds = [...]string{"input", "output", "system_output"}

type device struct {
	id        uint32
	name      string
	valid     bool
	nameKnown bool
}

type observer struct {
	mu      sync.Mutex
	stopped bool
	pending [3]bool
	wake    chan struct{}
	done    chan struct{}
	read    func(int) (device, error)
	last    [3]device
}

func newObserver(read func(int) (device, error)) *observer {
	return &observer{read: read, wake: make(chan struct{}, 1), done: make(chan struct{})}
}

// Only the observation goroutine reads snapshots after initialization.
// Failed reads do not erase the last known ID.
func (o *observer) snapshot(index int) {
	next, err := o.read(index)
	if err != nil {
		log.Printf("[audio] read_error default=%s source=unknown id_known=%t id=%d error=%q", defaultKinds[index], next.valid, next.id, err)
	}
	if !next.valid {
		return
	}
	previous := o.last[index]
	if !previous.valid {
		log.Printf("[audio] initial default=%s source=unknown id=%d name=%q name_known=%t", defaultKinds[index], next.id, next.name, next.nameKnown)
	} else if previous != next {
		log.Printf("[audio] changed default=%s source=unknown before_id=%d before_name=%q before_name_known=%t after_id=%d after_name=%q after_name_known=%t", defaultKinds[index], previous.id, previous.name, previous.nameKnown, next.id, next.name, next.nameKnown)
	}
	o.last[index] = next
}

func (o *observer) initialize() {
	for index := range defaultKinds {
		o.snapshot(index)
	}
}

func (o *observer) notify(index int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return
	}
	// A notification need not mean the value changed, and does not identify its caller.
	log.Printf("[audio] native_notification default=%s source=unknown", defaultKinds[index])
	o.pending[index] = true
	o.signal()
}

func (o *observer) signal() {
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

func (o *observer) run() {
	defer close(o.done)
	for {
		<-o.wake
		o.mu.Lock()
		stopped, pending := o.stopped, o.pending
		o.pending = [3]bool{}
		o.mu.Unlock()
		if stopped {
			return
		}
		// Read outside the callback and mutex; CoreAudio may deliver more notifications.
		// Bursts can share a snapshot, but each native notification is logged.
		for index, changed := range pending {
			if changed {
				o.snapshot(index)
			}
		}
	}
}
