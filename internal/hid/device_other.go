//go:build !darwin

package hid

// setNonExclusive is a no-op on non-macOS platforms.
func setNonExclusive() {}
