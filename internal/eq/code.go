// Package eq decodes Adjust Pro EQ codes.
package eq

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/charlietran/scapectl/internal/hid"
)

// ParseCode decodes a base64 EQ code exported by Adjust Pro.
// Format: [version=1][numBands][15 bytes per band][XOR checksum]
// Per band: [type:u8][gain:f32 LE][Q:f32 LE][freq:u16 LE][4 bytes padding]
func ParseCode(code string) ([]hid.EqPoint, error) {
	data, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	if len(data) < 3 {
		return nil, fmt.Errorf("EQ code too short")
	}
	if data[0] != 1 {
		return nil, fmt.Errorf("unsupported EQ code version: %d", data[0])
	}

	var checksum byte
	for _, b := range data[:len(data)-1] {
		checksum ^= b
	}
	if checksum != data[len(data)-1] {
		return nil, fmt.Errorf("checksum mismatch: computed 0x%02x, stored 0x%02x", checksum, data[len(data)-1])
	}

	numBands := int(data[1])
	if len(data) != 3+15*numBands {
		return nil, fmt.Errorf("invalid EQ code length: got %d bytes for %d bands", len(data), numBands)
	}
	bands := make([]hid.EqPoint, 0, numBands)
	for i := range numBands {
		off := 2 + 15*i
		point := hid.EqPoint{
			Type:   hid.EqFilterType(data[off]),
			GainDB: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[off+1:]))),
			Q:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[off+5:]))),
			FreqHz: int(binary.LittleEndian.Uint16(data[off+9:])),
		}
		if err := point.Validate(); err != nil {
			return nil, fmt.Errorf("EQ band %d: %w", i+1, err)
		}
		bands = append(bands, point)
	}
	return bands, nil
}
