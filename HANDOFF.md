# scape-ctl: Project Handoff Document

## Goal

Build a native desktop app (Go + systray) that fully replaces the Fractal Design Adjust Pro web app (https://adjust.fractal-design.com) for controlling the **Fractal Design Scape** wireless gaming headset. Additionally, support **script triggers** that fire when the headset connects or disconnects — something the web app can't do.

## What Exists Today

A scaffolded Go project at `scape-ctl/` with the full architecture wired up but **protocol bytes are placeholders**. The app compiles and runs but can't actually talk to the headset yet — that requires sniffing the real HID protocol from the web app.

### Project Structure

```
scape-ctl/
├── CLAUDE.md                    # Claude Code project context
├── README.md                    # User-facing docs
├── Makefile                     # build/install/udev shortcuts
├── go.mod                       # Go module (run `go mod tidy` first)
├── .gitignore
├── 50-fractal.rules             # Linux udev rule for non-root HID access
├── config.example.toml          # Example trigger config
├── cmd/scape-ctl/main.go        # Entry point: tray app + CLI subcommands
├── internal/
│   ├── hid/
│   │   ├── protocol.go          # Wire protocol: constants, builders, parsers
│   │   └── device.go            # hidapi wrapper: enumerate, open, send, receive
│   ├── monitor/monitor.go       # USB bus poller → connect/disconnect events
│   ├── triggers/triggers.go     # Runs user scripts on device events
│   ├── config/config.go         # TOML config at ~/.config/scape-ctl/config.toml
│   └── tray/tray.go             # System tray menu and click handlers
└── tools/
    └── webhid_sniffer.js        # Chrome DevTools script for capturing HID traffic
```

### Dependencies

- `github.com/sstallion/go-hid` — Go bindings for C hidapi (CGO required)
- `github.com/getlantern/systray` — cross-platform system tray
- `github.com/pelletier/go-toml/v2` — TOML config parser

System deps: `libhidapi-dev` (Linux), `brew install hidapi` (macOS), bundled on Windows.

## What We Know About the Hardware

### Device Identifiers

| Device | Vendor ID | Product ID | Notes |
|--------|-----------|------------|-------|
| Adjust Pro Hub | `0x36bc` | `0x1001` | Confirmed from community |
| Scape Dongle | `0x36bc` | **unknown** | Need `lsusb -d 36bc:` with dongle plugged in |
| Scape (USB cable) | `0x36bc` | **unknown** | Need `lsusb -d 36bc:` with cable connected |

All Fractal Adjust-compatible devices share vendor ID `0x36bc`.

### Connection Modes

The Scape headset connects to a PC three ways:
1. **2.4 GHz dongle** (plugs into charging dock's USB-A) — lowest latency, used for gaming
2. **USB-C cable** (included USB-C to USB-A cable) — wired mode, also used for firmware updates
3. **Bluetooth 5.3** — for mobile/console, NOT used for Adjust Pro configuration

The dongle and wired headset present as **separate USB HID devices** with different product IDs. The dongle acts as a wireless relay — HID commands sent to the dongle are forwarded to the headset. When wired, you talk directly to the headset.

### What the Web App Controls

From the Adjust Pro walkthrough and reviews:

- **EQ**: 3 customizable preset slots (default: "Balance", "Clarity", "Depth"). Parametric EQ with adjustable frequency/gain/Q per band. Switchable via headset button or app. Saved to headset internal memory.
- **Lighting**: 10 preset themes ("Northern Lights", "Summer Sky", "Radiant Dawn", etc.) + 6 custom patterns. Color wheel, RGB values, speed, brightness. Also supports Windows Dynamic Lighting (open standard, separate from HID).
- **Microphone**: Noise cancellation toggle, sidetone gain adjustment.
- **Status**: Battery level, firmware version (headset + dongle separately).
- **Firmware updates**: Also delivered over the same HID channel. The dongle and headset update separately — dongle first, then headset (must be on cable for headset update).
- **EQ sharing**: The app generates a shareable code string for EQ settings.

All settings persist in the headset's internal memory. The web app is stateless — it reads current values on connect and writes changes immediately.

### Linux Access

Fractal's official Linux instructions: create a udev rule at `/etc/udev/rules.d/50-fractal.rules`:
```
SUBSYSTEMS=="usb*", ATTRS{idVendor}=="36bc", MODE="0666"
```

### Offline App

Fractal added a downloadable offline version (Electron) accessible from the Settings tab of the web app. This is valuable for reverse engineering because you can unpack the `.asar` archive and read the JS source directly:
```bash
npx asar extract resources/app.asar unpacked/
grep -rn "sendReport\|sendFeatureReport\|0x36bc" unpacked/
```

## Protocol Reverse Engineering Plan

This is the critical next step. All protocol bytes in `internal/hid/protocol.go` are `0x00` placeholders marked with `TODO(sniff)`.

### Method 1: WebHID Sniffer (fastest)

The file `tools/webhid_sniffer.js` is a ready-to-use Chrome DevTools script that monkey-patches all WebHID methods to log traffic.

**Steps:**
1. Open https://adjust.fractal-design.com in Chrome
2. Open DevTools → Console
3. Paste entire contents of `tools/webhid_sniffer.js`, press Enter
4. Click "Add Fractal USB Device" in the web app and connect
5. Exercise each feature ONE AT A TIME with annotations:
   ```js
   scapeNote("about to switch EQ to slot 1 (Clarity)")
   // click Clarity in the app
   scapeNote("switched to Clarity")
   ```
6. Export the log: `scapeExport()` downloads JSON

**What to capture:**
- `DEVICE_OPENED` event → contains VID, PID, and `collections` (the HID report descriptor with report IDs, usage pages, and report sizes)
- `TX_OUTPUT` or `TX_FEATURE` → outgoing commands (tells you transport method + command bytes)
- `RX_INPUT` or `RX_FEATURE` → device responses (status, current settings)
- `REQUEST_DEVICE` → the VID/PID filter the web app uses

**What to determine:**
- Transport: does the app use `sendReport` (output reports → `TX_OUTPUT`) or `sendFeatureReport` (feature reports → `TX_FEATURE`)?
- Report ID(s): which report ID(s) are used?
- Command byte layout: first byte is usually a command selector, followed by parameters
- Response format: which byte offset is battery? firmware version? EQ data?

### Method 2: JS Bundle Decompilation

In Chrome DevTools → Sources → look under `adjust.fractal-design.com` for JS chunks. Pretty-print and search for:
- `navigator.hid` → device filter with VID/PID
- `sendReport` / `sendFeatureReport` → transport method
- `new Uint8Array` or hex literals near send calls → command payloads
- `inputreport` event listeners → response parsing logic

### Method 3: Electron App Unpacking

Download the offline version from Adjust Pro settings, then:
```bash
npx asar extract resources/app.asar unpacked/
grep -rn "sendReport\|sendFeatureReport\|0x36bc\|vendorId\|productId" unpacked/
```
This gives you browseable, less-minified source.

## What Needs to Be Done

### Phase 1: Protocol Discovery (must be done with physical hardware)

1. **Get device PIDs**: `lsusb -d 36bc:` with dongle, then with cable
2. **Run the sniffer**: capture a full session exercising all features
3. **Update `protocol.go`**:
   - `PIDScapeDongle` and `PIDScapeWired`
   - `ReportIDCommand` (and any other report IDs)
   - `ReportSize` (confirm 64 or discover actual size from collections)
   - `Transport` (output vs feature)
   - All `Cmd*` constants (command selector bytes)
   - `BuildSetEqCurve` — encoding of frequency/gain/Q into bytes
   - All `Parse*` functions — byte offset mapping
4. **Test**: `./scape-ctl raw`, `./scape-ctl sniff`, `./scape-ctl status`

### Phase 2: Core Functionality

- [ ] Verify `GetStatus()` returns correct battery/firmware/connection info
- [ ] Verify `SetActiveEq()` switches presets correctly
- [ ] Implement full EQ curve read/write (depends on band count and encoding from sniffing)
- [ ] Verify lighting mode set/get
- [ ] Test mic config (NC toggle, sidetone)
- [ ] Handle dongle-vs-wired differences (may need different command paths)

### Phase 3: Tray Polish

- [ ] Embed a proper tray icon (currently text-only — need a small PNG/ICO)
- [ ] Show which EQ preset is currently active (checkmark in submenu)
- [ ] Add all lighting modes to the submenu (not just on/off)
- [ ] Show firmware versions in a submenu or tooltip
- [ ] Add "Reconnect" menu item for when device is lost
- [ ] System notification on low battery

### Phase 4: Trigger System Extensions

- [ ] `HeadsetOnline` / `HeadsetOffline` events (dongle connected but headset powered on/off) — requires periodic status polling to detect headset presence through the dongle
- [ ] File watcher on config.toml for live reload
- [ ] Trigger cooldown (prevent rapid re-firing if device flaps)

### Phase 5: Nice to Haves

- [ ] EQ curve import/export (compatible with Adjust Pro's share codes if format is discovered)
- [ ] Battery history logging
- [ ] DBus interface (Linux) for integration with status bars like Waybar/Polybar
- [ ] Homebrew formula / AUR package / .deb packaging

## Known Gotchas

- **go-hid requires CGO** — cross-compilation needs target platform headers. For CI, use Docker or build natively on each platform.
- **The monitor fans out events** to both the tray (via its own goroutine) and the trigger runner. Currently both consume from the same channel, which means **only one receiver gets each event**. This needs to be fixed — either duplicate the channel or use a broadcast pattern. The tray uses its own `handleMonitorEvents()` goroutine reading from `mon.Events`, but `triggerRunner.Run()` also reads from `mon.Events`. **Fix: create a second channel or use a fan-out dispatcher.**
- **Firmware update commands** exist in the protocol. Identify them during sniffing and add safeguards so they can't be triggered accidentally.
- **The headset stores settings in flash memory.** Every write persists. Don't spam writes during testing.
- **Dynamic Lighting**: Windows has a built-in RGB standard that the Scape supports. It conflicts with HID-based RGB control. The web app tells users to disable it. The CLI should probably check/warn about this on Windows.

## Reference Links

- Adjust Pro web app: https://adjust.fractal-design.com
- Adjust Pro landing page: https://www.fractal-design.com/adjust-pro/
- Linux setup guide: https://support.fractal-design.com/support/solutions/articles/4000218717
- Scape product page: https://www.fractal-design.com/products/headsets/scape/scape/light/
- Scape FAQ: https://support.fractal-design.com/support/solutions/articles/4000218752-scape
- Ubuntu WebHID fix (has VID/PID for Hub): https://www.prefure.com/docs/logs/errorSolutions/fractal_adjust_pro_hub_ubuntu_webhid_fix
- Audio walkthrough: https://support.fractal-design.com/support/solutions/articles/4000218728
