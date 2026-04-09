package tui

import (
	"image"
	"image/color"
	"testing"
)

func TestDigitamaPlaySequence(t *testing.T) {
	seq := digitamaPlaySequence(3)
	if len(seq) != 21 {
		t.Fatalf("want 21 steps, got %d", len(seq))
	}
	if seq[0] != 0 || seq[1] != 1 || seq[19] != 1 || seq[20] != 2 {
		t.Fatalf("unexpected pattern: first pair %d,%d last triple %d,%d,%d", seq[0], seq[1], seq[18], seq[19], seq[20])
	}
	if seq2 := digitamaPlaySequence(2); seq2[20] != 1 {
		t.Fatalf("2 frames: last index should clamp to 1, got %d", seq2[20])
	}
}

func TestSplitDigitamaSheet(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 48, 16))
	for x := range 48 {
		for y := range 16 {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 0, A: 255})
		}
	}
	frames := splitDigitamaSheet(img)
	if len(frames) != 3 {
		t.Fatalf("want 3 frames, got %d", len(frames))
	}
	r, g, _, _ := frames[1].At(0, 0).RGBA()
	if byte(r>>8) != 16 || byte(g>>8) != 0 {
		t.Fatalf("frame 1 top-left: want R=16 G=0, got R=%d G=%d", r>>8, g>>8)
	}
}

func TestLoadRandomDigitama(t *testing.T) {
	a := loadRandomDigitama()
	if a == nil {
		t.Fatal("expected embedded digitama")
	}
	if len(a.frames) != 3 {
		t.Fatalf("want 3 frames from 48x16 sheet, got %d", len(a.frames))
	}
	if len(a.playSeq) != len(digitamaPlayPattern) {
		t.Fatalf("play seq len: want %d got %d", len(digitamaPlayPattern), len(a.playSeq))
	}
	if a.tickEvery != digitamaStepDuration {
		t.Fatalf("tick every: want %v got %v", digitamaStepDuration, a.tickEvery)
	}
	s := a.render()
	if s == "" {
		t.Fatal("expected non-empty render")
	}
}
