# systray fork

Copy of fyne.io/systray v1.11.0 (BSD-3-Clause, see LICENSE) with one change:
`setIcon` in `systray_darwin.m` sizes the status item image at 18pt instead
of 16pt so the tray icon matches its neighbors. Wired in via a `replace`
directive in the root `go.mod`.
