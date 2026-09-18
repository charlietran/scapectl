package hid

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestParseEqSettings(t *testing.T) {
	resp := make([]byte, ReportSize)
	resp[0], resp[1] = CmdEqGetSettings[0], CmdEqGetSettings[1]
	resp[2] = 2<<4 | 7 // profile 2, all three sample rates
	band := func(i int, typ EqFilterType, freq int, gain, q float32) {
		off := 3 + 12*i
		resp[off] = 1
		resp[off+1] = byte(typ)
		binary.LittleEndian.PutUint16(resp[off+2:], uint16(freq))
		binary.LittleEndian.PutUint32(resp[off+4:], math.Float32bits(gain))
		binary.LittleEndian.PutUint32(resp[off+8:], math.Float32bits(q))
	}
	band(2, EqLowShelf, 400, 8, 2) // disabled first band must not hide later bands
	band(4, EqPeak, 1100, 4, 0.7)  // include the last record

	s, err := ParseEqSettings(resp)
	if err != nil {
		t.Fatal(err)
	}
	if s.ProfileID != 2 || s.SamplingRate != 7 {
		t.Errorf("header: got profile %d rates %d", s.ProfileID, s.SamplingRate)
	}
	if len(s.Points) != 2 {
		t.Fatalf("got %d bands, want 2", len(s.Points))
	}
	if p := s.Points[1]; p.Type != EqPeak || p.FreqHz != 1100 || p.GainDB != 4 || math.Abs(p.Q-0.7) > 1e-6 {
		t.Errorf("band: got %+v", p)
	}
}

func TestParseEqSettingsRejectsTruncatedResponse(t *testing.T) {
	resp := make([]byte, 62) // final Q value is incomplete
	resp[0], resp[1] = 0xa7, 0x04
	if _, err := ParseEqSettings(resp); err == nil {
		t.Fatal("accepted a truncated EQ response")
	}
}
