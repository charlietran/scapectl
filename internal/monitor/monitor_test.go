package monitor

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"
)

func TestPowerLossConfirmation(t *testing.T) {
	type reading struct {
		at      time.Duration
		online  bool
		changed bool
		state   bool
	}
	for _, tc := range []struct {
		name    string
		initial bool
		steps   []reading
	}{
		{
			name: "brief loss leaves active headset selected",
			steps: []reading{
				{0, false, false, true},
				{time.Second, false, false, true},
				{2999 * time.Millisecond, true, false, true},
				{3 * time.Second, false, false, true},
				{5 * time.Second, false, false, true},
				{6 * time.Second, false, true, false},
				{7 * time.Second, false, false, false},
				{8 * time.Second, true, true, true},
			},
		},
		{
			name: "recovery at grace boundary cancels power off",
			steps: []reading{
				{0, false, false, true},
				{time.Second, false, false, true},
				{3 * time.Second, true, false, true},
			},
		},
		{
			name:    "startup offline also waits three seconds",
			initial: true,
			steps: []reading{
				{0, false, false, false},
				{2 * time.Second, false, false, false},
				{3 * time.Second, false, true, false},
				{4 * time.Second, false, false, false},
			},
		},
		{
			name:    "startup recovery still settles online readings",
			initial: true,
			steps: []reading{
				{0, false, false, false},
				{time.Second, true, false, false},
				{2 * time.Second, true, false, false},
				{3 * time.Second, true, true, true},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &Monitor{headsetChecked: !tc.initial, headsetOnline: !tc.initial}
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			for _, step := range tc.steps {
				changed := m.powerChanged(step.online, start.Add(step.at))
				if changed != step.changed || m.headsetOnline != step.state {
					t.Fatalf("at %s online=%t: changed=%t state=%t, want changed=%t state=%t", step.at, step.online, changed, m.headsetOnline, step.changed, step.state)
				}
			}
		})
	}
}

func TestPowerLossInterruptedReadRestartsGrace(t *testing.T) {
	m := &Monitor{headsetChecked: true, headsetOnline: true}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m.powerChanged(false, start)
	m.finishPowerLoss(start.Add(2*time.Second), "io-error")
	if m.powerChanged(false, start.Add(4*time.Second)) || !m.headsetOnline {
		t.Fatal("an interrupted observation must not count toward the next loss")
	}
	if !m.powerChanged(false, start.Add(7*time.Second)) || m.headsetOnline {
		t.Fatal("sustained loss after an interruption should still emit power off")
	}
}

func TestPowerLossRecoveryTelemetry(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	m := &Monitor{headsetChecked: true, headsetOnline: true}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m.powerChanged(false, start)
	m.powerChanged(false, start.Add(time.Second))
	m.powerChanged(true, start.Add(2*time.Second))
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want one pending and one recovery record, got %s", output.String())
	}
	for _, line := range lines {
		if !strings.Contains(line, "incident="+start.Format(time.RFC3339Nano)) {
			t.Fatalf("missing shared incident ID: %s", line)
		}
	}
	if !strings.Contains(lines[0], "outcome=pending") || !strings.Contains(lines[1], "outcome=recovered") || !strings.Contains(lines[1], "duration=2s samples=2") {
		t.Fatalf("missing suppressed-dropout evidence: %s", output.String())
	}
}

func TestStopFinishesPendingPowerLoss(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	m := New(time.Second)
	m.running, m.headsetChecked, m.headsetOnline = true, true, true
	m.powerChanged(false, time.Now())
	m.Stop()
	if !m.powerLossSince.IsZero() || m.powerLossSamples != 0 {
		t.Fatal("shutdown left an unfinished power-loss incident")
	}
	if !strings.Contains(output.String(), "outcome=stopped") || !strings.Contains(output.String(), "samples=1") {
		t.Fatalf("shutdown did not record the interrupted incident: %s", output.String())
	}
}
