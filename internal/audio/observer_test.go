package audio

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &output
}

func TestSnapshotReadFailuresPreserveLastKnownDevice(t *testing.T) {
	output := captureLogs(t)
	headset := device{id: 7, name: "Headset", valid: true, nameKnown: true}
	next, readErr := headset, error(nil)
	o := newObserver(func(int) (device, error) { return next, readErr })

	o.snapshot(0)

	next, readErr = device{}, errors.New("ID unavailable")
	o.snapshot(0)
	if o.last[0] != headset {
		t.Fatalf("ID read failure erased last known device: %+v", o.last[0])
	}
	if !strings.Contains(output.String(), "read_error default=input source=unknown id_known=false") {
		t.Fatalf("missing ID read error: %s", output.String())
	}

	// An unavailable name must not hide a successfully observed ID change.
	next, readErr = device{id: 9, valid: true}, errors.New("name unavailable")
	o.snapshot(0)
	if o.last[0] != next {
		t.Fatalf("name read failure hid device change: %+v", o.last[0])
	}
	if !strings.Contains(output.String(), "before_id=7 before_name=\"Headset\" before_name_known=true after_id=9 after_name=\"\" after_name_known=false") {
		t.Fatalf("missing partial device transition: %s", output.String())
	}

	next, readErr = device{id: 9, name: "Speakers", valid: true, nameKnown: true}, nil
	o.snapshot(0)
	if o.last[0] != next || !strings.Contains(output.String(), "after_id=9 after_name=\"Speakers\" after_name_known=true") {
		t.Fatalf("name recovery was not recorded: %s", output.String())
	}

	next = device{valid: true, nameKnown: true}
	o.snapshot(0)
	if o.last[0] != next || !strings.Contains(output.String(), "after_id=0 after_name=\"\" after_name_known=true") {
		t.Fatalf("no-default transition was not recorded: %s", output.String())
	}
}

func TestNotificationsQueueBeforeBaselineAndIgnoreAfterStop(t *testing.T) {
	output := captureLogs(t)
	reads := 0
	o := newObserver(func(int) (device, error) {
		reads++
		return device{valid: true, nameKnown: true}, nil
	})
	o.notify(1)
	o.notify(1)
	if reads != 0 || !o.pending[1] || len(o.wake) != 1 {
		t.Fatal("notification should queue a snapshot without reading in the native callback")
	}
	if strings.Count(output.String(), "native_notification default=output source=unknown") != 2 {
		t.Fatalf("coalesced snapshots must still log each notification: %s", output.String())
	}
	o.initialize()
	if reads != 3 || !o.pending[1] {
		t.Fatal("initial baseline should read every default and preserve pending notifications")
	}
	o.stopped = true
	before := output.Len()
	o.notify(2)
	if output.Len() != before || o.pending[2] {
		t.Fatal("late notification should not log or queue work after stop")
	}
}
