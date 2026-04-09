package tui

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/draw"
	_ "image/png" // register PNG decoder
	"io/fs"
	"math/rand/v2"
	"path"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

//go:embed digitama/*.png
var digitamaFS embed.FS

const (
	digitamaLoopPeriod = 3 * time.Second
	digitamaFrameW     = 16
	digitamaFrameH     = 16
)

// digitamaPlayPattern is 0-based indices for frames 1,2,1,2,… (ten pairs) then frame 3; repeats after step 21.
var digitamaPlayPattern = []int{
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1,
	0, 1, 0, 1, 0, 1, 0, 1, 0, 1,
	2,
}

type digitamaTickMsg struct{}

type digitamaAnim struct {
	frames    []image.Image
	playSeq   []int // indices into frames; one step per tick
	seqPos    int
	tickEvery time.Duration
}

func loadRandomDigitama() *digitamaAnim {
	entries, err := fs.ReadDir(digitamaFS, "digitama")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(path.Ext(e.Name()), ".png") {
			names = append(names, path.Join("digitama", e.Name()))
		}
	}
	if len(names) == 0 {
		return nil
	}
	name := names[rand.IntN(len(names))]
	raw, err := digitamaFS.ReadFile(name)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	frames := splitDigitamaSheet(img)
	if len(frames) == 0 {
		return nil
	}
	seq := digitamaPlaySequence(len(frames))
	tick := digitamaLoopPeriod
	if len(seq) > 1 {
		tick = digitamaLoopPeriod / time.Duration(len(seq))
	}
	return &digitamaAnim{frames: frames, playSeq: seq, tickEvery: tick}
}

// digitamaPlaySequence maps the fixed 1,2,1,2,…,3 choreography onto frame indices, clamped for smaller sheets.
func digitamaPlaySequence(numFrames int) []int {
	if numFrames < 1 {
		return nil
	}
	out := make([]int, len(digitamaPlayPattern))
	for i, v := range digitamaPlayPattern {
		if v >= numFrames {
			out[i] = numFrames - 1
		} else {
			out[i] = v
		}
	}
	return out
}

func splitDigitamaSheet(src image.Image) []image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < digitamaFrameW || h < digitamaFrameH {
		return []image.Image{src}
	}
	n := w / digitamaFrameW
	if n < 1 {
		n = 1
	}
	out := make([]image.Image, 0, n)
	for i := range n {
		dst := image.NewRGBA(image.Rect(0, 0, digitamaFrameW, digitamaFrameH))
		r := image.Rect(b.Min.X+i*digitamaFrameW, b.Min.Y, b.Min.X+(i+1)*digitamaFrameW, b.Min.Y+digitamaFrameH)
		draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
		out = append(out, dst)
	}
	return out
}

func (a *digitamaAnim) needsAnimation() bool {
	if a == nil || len(a.playSeq) < 2 {
		return false
	}
	first := a.playSeq[0]
	for _, v := range a.playSeq[1:] {
		if v != first {
			return true
		}
	}
	return false
}

func (a *digitamaAnim) advance() {
	if a == nil || len(a.playSeq) == 0 {
		return
	}
	a.seqPos = (a.seqPos + 1) % len(a.playSeq)
}

func (a *digitamaAnim) render() string {
	if a == nil || len(a.frames) == 0 || len(a.playSeq) == 0 {
		return ""
	}
	fi := a.playSeq[a.seqPos]
	if fi < 0 || fi >= len(a.frames) {
		fi = len(a.frames) - 1
	}
	return renderHalfBlock16(a.frames[fi])
}

func renderHalfBlock16(img image.Image) string {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return ""
	}
	var sb strings.Builder
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x++ {
			r1, g1, b1, a1 := rgbaAt(img, b.Min.X+x, b.Min.Y+y)
			r2, g2, b2, a2 := rgbaAt(img, b.Min.X+x, b.Min.Y+y+1)
			fg := lipgloss.Color(hexRGB(r2, g2, b2, a2))
			bg := lipgloss.Color(hexRGB(r1, g1, b1, a1))
			sb.WriteString(lipgloss.NewStyle().Foreground(fg).Background(bg).Render("▄"))
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func rgbaAt(img image.Image, x, y int) (r, g, b, a uint32) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return 0, 0, 0, 0
	}
	return img.At(x, y).RGBA()
}

func hexRGB(r, g, b, a uint32) string {
	if a < 0x8000 {
		return "#000000"
	}
	return fmt.Sprintf("#%02x%02x%02x", byte(r>>8), byte(g>>8), byte(b>>8))
}
