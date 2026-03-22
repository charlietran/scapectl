package hid

import usbhid "github.com/sstallion/go-hid"

// setNonExclusive disables exclusive mode on macOS so the OS can still
// receive Consumer Control HID reports (volume, media keys) from the device.
func setNonExclusive() {
	usbhid.SetOpenExclusive(false)
}
