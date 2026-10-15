package tray

import "testing"

func TestIconFrames(t *testing.T) {
	if n := len(iconFrames("connected", "muted")); n != fadeFrames {
		t.Fatalf("mute frames = %d", n)
	}
	if n := len(iconFrames("nodongle", "disconnected")); n != fadeFrames {
		t.Fatalf("fade frames = %d", n)
	}
	if n := len(iconFrames("", "nodongle")); n != 1 {
		t.Fatalf("initial frames = %d", n)
	}
}
