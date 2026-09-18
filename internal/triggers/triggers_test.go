package triggers

import (
	"testing"
	"time"

	"github.com/charlietran/scapectl/internal/config"
	"github.com/charlietran/scapectl/internal/hid"
	"github.com/charlietran/scapectl/internal/monitor"
)

func batteryEvent(percent int) monitor.Event {
	return monitor.Event{
		Type:   monitor.EventBatteryLevel,
		Status: &hid.DeviceStatus{BatteryPercent: percent},
	}
}

func checkFire(t *testing.T, r *Runner, rule config.TriggerRule, evt monitor.Event, now time.Time, want bool) {
	t.Helper()
	r.mu.Lock()
	got := r.shouldFire(rule, evt, now)
	r.mu.Unlock()
	if got != want {
		t.Fatalf("firing decision = %t, want %t", got, want)
	}
}

func TestBatteryEpisode(t *testing.T) {
	r := New(config.DefaultConfig())
	rule := config.TriggerRule{Event: "BatteryLevel", Script: "notify", Enabled: true}
	unknown := monitor.Event{Type: monitor.EventBatteryLevel}
	steps := []struct {
		name string
		evt  monitor.Event
		want bool
	}{
		{"missing before low", unknown, false},
		{"unknown before low", batteryEvent(-1), false},
		{"first at default threshold", batteryEvent(20), true},
		{"missing after low", unknown, false},
		{"unknown after low", batteryEvent(-1), false},
		{"still low a day later", batteryEvent(0), false},
		{"recovery above threshold", batteryEvent(21), false},
		{"missing after recovery", unknown, false},
		{"re-enter low", batteryEvent(19), true},
		{"threshold is still low", batteryEvent(20), false},
	}
	for i, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			checkFire(t, r, rule, step.evt, time.Unix(0, 0).Add(time.Duration(i)*24*time.Hour), step.want)
		})
	}
}

func TestBatteryReloadPreservesEpisode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Settings.TriggersEnabled = true
	cfg.Triggers = []config.TriggerRule{{Event: "BatteryLevel", Script: "notify", Enabled: true}}
	r := New(cfg)
	now := time.Unix(0, 0)
	checkFire(t, r, cfg.Triggers[0], batteryEvent(20), now, true)

	reloaded := config.DefaultConfig()
	reloaded.Settings.TriggersEnabled = true
	reloaded.Settings.Verbose = true
	reloaded.Triggers = []config.TriggerRule{{Event: "BatteryLevel", Script: "notify", Enabled: true, Battery: 20}}
	r.Reload(reloaded)
	checkFire(t, r, reloaded.Triggers[0], batteryEvent(19), now.Add(time.Hour), false)
	checkFire(t, r, reloaded.Triggers[0], batteryEvent(21), now.Add(2*time.Hour), false)
	checkFire(t, r, reloaded.Triggers[0], batteryEvent(20), now.Add(3*time.Hour), true)

	// A new process starts with no previously notified episodes.
	checkFire(t, New(reloaded), reloaded.Triggers[0], batteryEvent(20), now.Add(4*time.Hour), true)
}

func TestBatteryRulesAreIndependent(t *testing.T) {
	r := New(config.DefaultConfig())
	warning := config.TriggerRule{Event: "BatteryLevel", Script: "notify", Enabled: true, Battery: 20}
	critical := warning
	critical.Battery = 10
	otherScript := warning
	otherScript.Script = "other-notify"
	now := time.Unix(0, 0)

	checkFire(t, r, warning, batteryEvent(15), now, true)
	checkFire(t, r, critical, batteryEvent(15), now, false)
	checkFire(t, r, otherScript, batteryEvent(15), now, true)
	checkFire(t, r, critical, batteryEvent(10), now, true)
	checkFire(t, r, warning, batteryEvent(10), now, false)
	checkFire(t, r, critical, batteryEvent(11), now, false)
	checkFire(t, r, warning, batteryEvent(11), now, false)
	checkFire(t, r, critical, batteryEvent(10), now, true)
	checkFire(t, r, otherScript, batteryEvent(10), now, false)
}

func TestBatteryCooldownDoesNotConsumeEpisode(t *testing.T) {
	r := New(config.DefaultConfig())
	rule := config.TriggerRule{Event: "BatteryLevel", Script: "notify", Enabled: true, Battery: 20, Cooldown: 300}
	now := time.Unix(0, 0)
	checkFire(t, r, rule, batteryEvent(20), now, true)
	checkFire(t, r, rule, batteryEvent(21), now.Add(time.Second), false)
	checkFire(t, r, rule, batteryEvent(19), now.Add(2*time.Second), false)
	checkFire(t, r, rule, batteryEvent(18), now.Add(300*time.Second), true)

	// Distinct thresholds still share the existing event+script cooldown.
	critical := rule
	critical.Battery = 10
	checkFire(t, r, critical, batteryEvent(10), now.Add(301*time.Second), false)
	checkFire(t, r, rule, batteryEvent(17), now.Add(600*time.Second), false)
	checkFire(t, r, critical, batteryEvent(10), now.Add(600*time.Second), true)
}

func TestCooldownIsolatedByEventAndScript(t *testing.T) {
	r := New(config.DefaultConfig())
	rule := config.TriggerRule{Event: "HeadsetPowerOn", Script: "notify", Enabled: true, Cooldown: 300}
	on := monitor.Event{Type: monitor.EventHeadsetPowerOn}
	off := monitor.Event{Type: monitor.EventHeadsetPowerOff}
	now := time.Unix(0, 0)
	checkFire(t, r, rule, on, now, true)
	checkFire(t, r, rule, on, now.Add(time.Second), false)

	otherEvent := rule
	otherEvent.Event = "HeadsetPowerOff"
	checkFire(t, r, otherEvent, off, now.Add(time.Second), true)
	otherScript := rule
	otherScript.Script = "other-notify"
	checkFire(t, r, otherScript, on, now.Add(time.Second), true)
	checkFire(t, r, rule, on, now.Add(300*time.Second), true)

	rule.Cooldown = 0
	checkFire(t, r, rule, on, now.Add(301*time.Second), true)
	checkFire(t, r, rule, on, now.Add(301*time.Second), true)
}
