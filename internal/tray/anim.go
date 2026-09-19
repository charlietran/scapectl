package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"time"
)

// Icon transitions on macOS: the menubar has no built-in animation, so we
// cross-fade by swapping template icons on a timer. Each setIconState bumps
// animSeq so a newer state cancels an animation in flight.

const (
	fadeFrames = 6
	fadeStep   = 30 * time.Millisecond
)

type frame struct {
	png  []byte
	hold time.Duration
}

// iconFrames returns the frame sequence for a transition, ending on the
// final state's icon.
func iconFrames(set, from, to string) []frame {
	toPNG := iconBytes(set, to, "black.png")
	if from == "" {
		return []frame{{toPNG, 0}}
	}
	var frames []frame
	for _, f := range crossfade(iconBytes(set, from, "black.png"), toPNG) {
		frames = append(frames, frame{f, fadeStep})
	}
	return frames
}

// crossfade blends two black+alpha template PNGs by alpha only.
func crossfade(from, to []byte) [][]byte {
	a, errA := png.Decode(bytes.NewReader(from))
	b, errB := png.Decode(bytes.NewReader(to))
	if errA != nil || errB != nil {
		return [][]byte{to}
	}
	r := a.Bounds()
	var frames [][]byte
	for i := 1; i < fadeFrames; i++ {
		t := uint32(i * 0xffff / fadeFrames)
		out := image.NewNRGBA(r)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				_, _, _, aa := a.At(x, y).RGBA()
				_, _, _, ab := b.At(x, y).RGBA()
				alpha := (aa*(0xffff-t) + ab*t) / 0xffff
				out.SetNRGBA(x, y, nrgbaBlack(uint8(alpha>>8)))
			}
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, out); err != nil {
			return [][]byte{to}
		}
		frames = append(frames, buf.Bytes())
	}
	return append(frames, to)
}

func nrgbaBlack(alpha uint8) color.NRGBA { return color.NRGBA{A: alpha} }
