# CLAUDE.md

## Project Overview

scapectl is a Go system tray app + CLI for controlling the Fractal Design Scape wireless gaming headset over USB HID. It replaces the browser-only Adjust Pro web app (https://adjust.fractal-design.com) with a native desktop app that also supports script triggers on device connect/disconnect.

**Status: working.** The core protocol is mapped in `internal/hid/protocol.go`. Use `tools/webhid_sniffer.js` to discover new commands.

## Architecture

```
cmd/scapectl/main.go           CLI entry point + tray launcher
internal/
  hid/protocol.go                Wire protocol constants, report builders/parsers
  hid/device.go                  Device wrapper over internal/usbhid (open/send/receive)
  monitor/monitor.go             USB bus scan + headset status poller, emits events
  triggers/triggers.go           Runs user shell scripts on device events
  usbhid/                        Inlined pure Go HID library (from rafaelmartins/usbhid)
  config/config.go               TOML config (~/.config/scapectl/config.toml)
  tray/tray.go                   System tray menu and click handlers
tools/webhid_sniffer.js          Chrome DevTools script for capturing HID traffic
```

## Key Technical Details

- **Vendor ID**: `0x36bc` (Fractal Design). The Scape dongle is PID `0x0001`; the Adjust Pro Hub is `0x1001`. The wired headset PID is still unknown.
- **Transport**: HID output reports (`sendReport`), report ID 2, 63-byte payload. `Transport` in `protocol.go` selects the path.
- **Monitor**: Scans the USB bus only while no device is open, then polls `f1 21` (default 1.5s). No hotplug listener.
- **Triggers**: Config-driven. Scripts get event context via env vars (`SCAPE_EVENT`, `SCAPE_DEVICE`, `SCAPE_JSON`, etc). Executed via `sh -c` on Unix, `cmd /S /C` on Windows.
- **Tray**: Uses `fyne.io/systray`. The tray and trigger runner both consume events from the monitor's channel.

## Build & Run

```bash
# Prerequisites (Linux)
sudo cp 50-fractal.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules && sudo udevadm trigger

make build        # produces ./scapectl
make run          # build + launch tray
./scapectl devices   # list Fractal HID devices
./scapectl sniff     # print incoming HID data
./scapectl raw 01 02 # send arbitrary bytes
```

On macOS: Xcode Command Line Tools (the tray uses cgo). On Linux and Windows: nothing extra.

## Conventions

- `TODO` comments in `protocol.go` mark protocol bytes still unknown. Search for these when updating the protocol.
- The `internal/hid/protocol.go` file is the single source of truth for the wire format. Report builders return `(reportID, payload)` tuples. Parsers take `[]byte` and return typed structs.
- Keep the CLI subcommands in `main.go` thin — they should call into `internal/` packages.
- Config file format is TOML. The canonical example is `config.example.toml`.
- The `raw` and `sniff` CLI commands exist specifically for protocol discovery. Don't remove them.
- Always run `make build` as its own separate step before committing. Never combine build and git commit in the same command.
- Do not commit or push automatically. Only commit when the user explicitly asks.

## Dependencies

- `internal/usbhid` — Inlined pure Go HID library (from github.com/rafaelmartins/usbhid, BSD-3-Clause). Uses IOKit directly on macOS via purego for async input report callbacks.
- `github.com/ebitengine/purego` — Pure Go syscall bridge (used by usbhid for IOKit on macOS)
- `fyne.io/systray` — Cross-platform system tray (fork of getlantern/systray, no GTK dependency)
- `github.com/pelletier/go-toml/v2` — TOML parser

Linux and Windows builds need no cgo and cross-compile anywhere. macOS builds use cgo for the tray, so build them on a Mac.

## Key Architecture Notes

- Monitor keeps a persistent HID connection and polls f1 21 every 1.5s
- Tray commands use monitor.RunCommand() which locks devMu to prevent interleaving with polls
- Sidetone requires interleaved status polls between f1 34 commands (dongle firmware requirement)
- The dongle sends unsolicited 11 21 reports — SendAndReceive filters by echo bytes to find the correct response
- 11 21 byte 3 (headset present) is unreliable — flaps in a 3-tick cycle. Use f1 21 byte 18 for presence.
- f1 21 byte 18 can be stale right after USB re-enumeration (KVM switch), so the first power event after connect waits for it to hold 2s (`powerSettle`).
- Device is opened non-exclusive so the OS still receives volume/media key reports

## Reverse Engineering Workflow

1. Run `lsusb -d 36bc:` to get product IDs
2. Paste `tools/webhid_sniffer.js` into Chrome DevTools on adjust.fractal-design.com
3. Connect device, exercise features, call `scapeExport()` to download the HID log
4. Update constants in `internal/hid/protocol.go`
5. Test with `./scapectl raw` and `./scapectl sniff`

Alternatively, download the offline Electron app from Adjust Pro's Settings tab and unpack it:
```bash
npx asar extract resources/app.asar unpacked/
grep -rn "sendReport\|sendFeatureReport\|0x36bc" unpacked/
```
