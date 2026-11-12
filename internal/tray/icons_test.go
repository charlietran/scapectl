package tray

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestWindowsIconColors(t *testing.T) {
	for _, state := range []string{"nodongle", "disconnected", "connected", "muted"} {
		for color, want := range map[string]byte{"black": 0, "white": 255} {
			t.Run(fmt.Sprintf("%s/%s", state, color), func(t *testing.T) {
				data := iconBytes(state, color+".ico")
				if len(data) < 6 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
					t.Fatal("invalid ICO header")
				}
				count := int(binary.LittleEndian.Uint16(data[4:6]))
				seen := map[int]bool{}
				for i := range count {
					entry := data[6+i*16 : 6+(i+1)*16]
					size := int(entry[0])
					offset := int(binary.LittleEndian.Uint32(entry[12:16]))
					bitmap := data[offset:]
					if binary.LittleEndian.Uint32(bitmap[:4]) != 40 || binary.LittleEndian.Uint16(bitmap[14:16]) != 32 {
						t.Fatal("expected a 32-bit BMP icon")
					}
					pixels := bitmap[40 : 40+size*size*4]
					visible := false
					for p := 0; p < len(pixels); p += 4 {
						if pixels[p+3] == 0 {
							continue
						}
						visible = true
						if pixels[p] != want || pixels[p+1] != want || pixels[p+2] != want {
							t.Fatalf("%dpx icon has the wrong foreground color", size)
						}
					}
					if !visible {
						t.Fatalf("%dpx icon is invisible", size)
					}
					seen[size] = true
				}
				for _, size := range []int{16, 32, 48, 64} {
					if !seen[size] {
						t.Errorf("missing %dpx icon", size)
					}
				}
			})
		}
	}
}
