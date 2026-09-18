package eq

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"testing"

	"github.com/charlietran/scapectl/internal/hid"
)

// encode constructs padded band records for parser boundary tests.
func encode(bands []hid.EqPoint) string {
	buf := []byte{1, byte(len(bands))}
	for _, b := range bands {
		band := make([]byte, 15)
		band[0] = byte(b.Type)
		binary.LittleEndian.PutUint32(band[1:], math.Float32bits(float32(b.GainDB)))
		binary.LittleEndian.PutUint32(band[5:], math.Float32bits(float32(b.Q)))
		binary.LittleEndian.PutUint16(band[9:], uint16(b.FreqHz))
		buf = append(buf, band...)
	}
	var checksum byte
	for _, c := range buf {
		checksum ^= c
	}
	return base64.StdEncoding.EncodeToString(append(buf, checksum))
}

var sample = []hid.EqPoint{
	{Type: hid.EqLowShelf, FreqHz: 100, GainDB: 6, Q: 0.7},
	{Type: hid.EqPeak, FreqHz: 3000, GainDB: -4, Q: 1.5},
	{Type: hid.EqHighShelf, FreqHz: 8000, GainDB: 3, Q: 0.7},
}

func TestParseCode(t *testing.T) {
	bands, err := ParseCode(encode(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(bands) != len(sample) {
		t.Fatalf("got %d bands, want %d", len(bands), len(sample))
	}
	for i, b := range bands {
		want := sample[i]
		if b.Type != want.Type || b.FreqHz != want.FreqHz ||
			math.Abs(b.GainDB-want.GainDB) > 1e-5 || math.Abs(b.Q-want.Q) > 1e-5 {
			t.Errorf("band %d: got %+v, want %+v", i, b, want)
		}
	}
}

func TestParseCodeRejectsBadChecksum(t *testing.T) {
	code := encode(sample)
	raw, _ := base64.StdEncoding.DecodeString(code)
	raw[len(raw)-1] ^= 0xff
	if _, err := ParseCode(base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestParseCodeRejectsTruncatedRecord(t *testing.T) {
	raw, _ := base64.StdEncoding.DecodeString(encode(sample[:1]))
	// Remove padding but retain all parameter bytes and a valid checksum.
	raw = raw[:13]
	var checksum byte
	for _, b := range raw {
		checksum ^= b
	}
	raw = append(raw, checksum)
	if _, err := ParseCode(base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("accepted an incomplete 15-byte band record")
	}
}
