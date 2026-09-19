package tray

import "testing"

func TestIconFrames(t *testing.T) {
	if n := len(iconFrames("ghost", "connected", "muted")); n != fadeFrames {
		t.Fatalf("mute frames = %d", n)
	}
	if n := len(iconFrames("outline", "nodongle", "disconnected")); n != fadeFrames {
		t.Fatalf("fade frames = %d", n)
	}
	if n := len(iconFrames("bogus", "", "nodongle")); n != 1 {
		t.Fatalf("initial frames = %d", n)
	}
}
