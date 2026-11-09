// Package triggers runs user-configured scripts when device events occur.
//
// Scripts receive event context via environment variables:
//
//	SCAPE_EVENT      = event name
//	SCAPE_DEVICE     = product name
//	SCAPE_VID        = vendor ID (hex)
//	SCAPE_PID        = product ID (hex)
//	SCAPE_PATH       = device path
//	SCAPE_TIMESTAMP  = ISO 8601 timestamp
//	SCAPE_BATTERY    = battery percentage (BatteryLevel events only)
//	SCAPE_EQ_SLOT    = active EQ slot, 1-3
//	SCAPE_EQ_DATA    = JSON bands of the new EQ slot (EqChanged events only)
//	SCAPE_DIR        = directory containing the scapectl executable
package triggers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charlietran/scapectl/internal/config"
	"github.com/charlietran/scapectl/internal/monitor"
)

// Runner listens for monitor events and fires matching trigger scripts.
type Runner struct {
	cfg        *config.Config
	mu         sync.Mutex
	lastFire   map[string]time.Time // event+script → last fire time (for cooldown)
	lowBattery map[batteryRuleKey]bool
}

type batteryRuleKey struct {
	script    string
	threshold int
}

// New creates a trigger runner with the given config.
func New(cfg *config.Config) *Runner {
	return &Runner{
		cfg:        cfg,
		lastFire:   make(map[string]time.Time),
		lowBattery: make(map[batteryRuleKey]bool),
	}
}

// Reload swaps in a new config without resetting cooldowns or low-battery episodes.
func (r *Runner) Reload(cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cfg = cfg
	log.Printf("[triggers] reloaded (%d rules)", len(cfg.Triggers))
}

// Run listens on the event channel and dispatches triggers. Blocking.
func (r *Runner) Run(events <-chan monitor.Event) {
	for evt := range events {
		r.dispatch(evt)
	}
}

// Enabled reports whether trigger script execution is enabled.
func (r *Runner) Enabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg.Settings.TriggersEnabled
}

func (r *Runner) dispatch(evt monitor.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	evtStr := evt.Type.String()
	periodic := evt.Type == monitor.EventHeadsetStatus || evt.Type == monitor.EventBatteryLevel

	// HeadsetStatus and BatteryLevel fire every poll — only log in verbose mode
	if periodic {
		if r.cfg.Settings.Verbose {
			log.Printf("[event] %s: %s", evtStr, evt.Device)
		}
	} else if r.cfg.Settings.Verbose {
		log.Printf("[event] %s: %s", evtStr, evt.Device)
	} else {
		log.Printf("[event] %s: %s", evtStr, evt.Device.ShortString())
	}

	if !r.cfg.Settings.TriggersEnabled {
		if !periodic {
			log.Printf("[triggers] skipped event=%s event_time=%s event_age=%s reason=triggers-disabled", evtStr, evt.Timestamp.Format(time.RFC3339Nano), time.Since(evt.Timestamp))
		}
		return
	}

	now := time.Now()
	matched := false
	for index, rule := range r.cfg.Triggers {
		if rule.Event != evtStr {
			continue
		}
		matched = true
		if !rule.Enabled {
			if !periodic {
				log.Printf("[triggers] skipped event=%s event_time=%s event_age=%s rule=%d reason=rule-disabled", evtStr, evt.Timestamp.Format(time.RFC3339Nano), time.Since(evt.Timestamp), index+1)
			}
			continue
		}

		if !r.shouldFire(rule, evt, now) {
			if !periodic {
				log.Printf("[triggers] skipped event=%s event_time=%s event_age=%s rule=%d reason=cooldown", evtStr, evt.Timestamp.Format(time.RFC3339Nano), time.Since(evt.Timestamp), index+1)
			}
			continue
		}

		log.Printf("[triggers] scheduled event=%s event_time=%s event_age=%s rule=%d", evtStr, evt.Timestamp.Format(time.RFC3339Nano), time.Since(evt.Timestamp), index+1)

		go r.exec(rule, evt, index+1)
	}
	if !matched && !periodic {
		log.Printf("[triggers] skipped event=%s event_time=%s event_age=%s reason=no-matching-rule", evtStr, evt.Timestamp.Format(time.RFC3339Nano), time.Since(evt.Timestamp))
	}
}

// shouldFire records an allowed dispatch. The caller must hold r.mu.
func (r *Runner) shouldFire(rule config.TriggerRule, evt monitor.Event, now time.Time) bool {
	var batteryKey batteryRuleKey
	if evt.Type == monitor.EventBatteryLevel {
		if evt.Status == nil || evt.Status.BatteryPercent < 0 {
			return false
		}
		threshold := rule.Battery
		if threshold <= 0 {
			threshold = 20
		}
		batteryKey = batteryRuleKey{script: rule.Script, threshold: threshold}
		if evt.Status.BatteryPercent > threshold {
			delete(r.lowBattery, batteryKey)
			return false
		}
		if r.lowBattery[batteryKey] {
			return false
		}
	}

	// Keep cooldowns isolated by event+script, including battery rules.
	if rule.Cooldown > 0 {
		key := rule.Event + "\x00" + rule.Script
		if last, exists := r.lastFire[key]; exists && now.Sub(last) < time.Duration(rule.Cooldown)*time.Second {
			return false
		}
		r.lastFire[key] = now
	}

	// A cooldown-blocked reading must not consume the next low-battery episode.
	if evt.Type == monitor.EventBatteryLevel {
		r.lowBattery[batteryKey] = true
	}
	return true
}

func (r *Runner) exec(rule config.TriggerRule, evt monitor.Event, ruleIndex int) {
	started := time.Now()
	eventTime := evt.Timestamp.Format(time.RFC3339Nano)
	log.Printf("[triggers] started event=%s event_time=%s event_age=%s rule=%d", evt.Type, eventTime, started.Sub(evt.Timestamp), ruleIndex)
	evtJSON, _ := json.Marshal(map[string]string{
		"event":     evt.Type.String(),
		"device":    evt.Device.ProductName,
		"vid":       fmt.Sprintf("%04x", evt.Device.VendorID),
		"pid":       fmt.Sprintf("%04x", evt.Device.ProductID),
		"path":      evt.Device.Path,
		"timestamp": evt.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	})

	cmd := shellCmd(rule.Script)

	env := append(os.Environ(),
		"SCAPE_EVENT="+evt.Type.String(),
		"SCAPE_DEVICE="+evt.Device.ProductName,
		"SCAPE_VID="+fmt.Sprintf("%04x", evt.Device.VendorID),
		"SCAPE_PID="+fmt.Sprintf("%04x", evt.Device.ProductID),
		"SCAPE_PATH="+evt.Device.Path,
		"SCAPE_TIMESTAMP="+evt.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		"SCAPE_JSON="+string(evtJSON),
	)

	// Add battery percentage for BatteryLevel events
	if evt.Status != nil && evt.Status.BatteryPercent >= 0 {
		env = append(env, fmt.Sprintf("SCAPE_BATTERY=%d", evt.Status.BatteryPercent))
	}
	if evt.Status != nil && evt.Status.EqSlot > 0 {
		env = append(env, fmt.Sprintf("SCAPE_EQ_SLOT=%d", evt.Status.EqSlot))
	}
	if evt.Eq != nil {
		if data, err := json.Marshal(evt.Eq.Points); err == nil {
			env = append(env, "SCAPE_EQ_DATA="+string(data))
		} else {
			log.Printf("[triggers] encode EQ bands: %v", err)
		}
	}

	// Add app directory so trigger scripts can reference bundled helpers (e.g. notify.ps1)
	if execPath, err := os.Executable(); err == nil {
		env = append(env, "SCAPE_DIR="+filepath.Dir(execPath))
	}

	cmd.Env = env

	output, err := cmd.CombinedOutput()
	log.Printf("[triggers] completed event=%s event_time=%s event_age=%s rule=%d duration=%s error=%v", evt.Type, eventTime, time.Since(evt.Timestamp), ruleIndex, time.Since(started), err)
	if len(output) > 0 {
		log.Printf("[triggers] output event=%s event_time=%s rule=%d output=%q", evt.Type, eventTime, ruleIndex, output)
	}
}
