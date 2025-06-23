package gradient

import (
	"io"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"

	"vrc-moments/pkg/pool"
)

var PastelColors = []string{"#00E2FD", "#6D90FA", "#FF22EE", "#FF8D7A", "#FFC851"}
var BlueGreenYellow = []string{"#2A7B9B", "#57C785", "#EDDD53"}
var PastelRainbow = []string{"#C5F9D7", "#F7D486", "#F27A7D"}
var PastelGreenBlue = []string{"#C2E59C", "#64B3F4"}
var GreenPinkBlue = []string{"#CAEFD7", "#F5BFD7", "#ABC9E9"}
var PinkOrange = []string{"#FFC05F", "#C4657D"}

func Static(s string, hexColors ...string) string {
	switch len(hexColors) {
	case 0:
		return s
	case 1:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hexColors[0])).Render(s)
	}

	runes := []rune(s)
	total := len(runes)
	if total == 0 {
		return s
	}

	colors := make([]colorful.Color, len(hexColors))
	for i, hex := range hexColors {
		c, err := colorful.Hex(hex)
		if err != nil {
			return s
		}
		colors[i] = c
	}

	segments := len(colors) - 1
	var result strings.Builder

	for i, r := range runes {
		var ratio float64
		if total > 1 {
			ratio = float64(i) / float64(total-1)
		} else {
			ratio = 0
		}

		segmentIndex := int(ratio * float64(segments))
		if segmentIndex >= segments {
			segmentIndex = segments - 1
		}
		localRatio := (ratio * float64(segments)) - float64(segmentIndex)
		c := colors[segmentIndex].BlendLab(colors[segmentIndex+1], localRatio)

		hex := c.Clamped().Hex()
		styled := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(string(r))
		result.WriteString(styled)
	}

	return result.String()
}

// Global gradient renderer (truecolor)
var Global = NewGradientRenderer(termenv.TrueColor)

type Renderer struct {
	profile termenv.Profile
	cache   map[string]*FrameData
	mu      sync.RWMutex

	pool *pool.Pool[*strings.Builder]
}

type FrameData struct {
	message string
	frames  [][]termenv.RGBColor
	index   int
	steps   int

	pool *pool.Pool[*strings.Builder]
}

func NewGradientRenderer(profile termenv.Profile) *Renderer {
	return &Renderer{
		profile: profile,
		cache:   make(map[string]*FrameData),

		pool: pool.New(func() *strings.Builder { return new(strings.Builder) }),
	}
}

// StepsFromDuration computes the frame count
func StepsFromDuration(width int, duration time.Duration, fps float64) int {
	frames := int(float64(duration.Nanoseconds()) * fps / float64(time.Second))
	steps := frames

	if width > 0 {
		steps = max(steps, width)
	}

	return min(max(steps, 5), 120)
}

// Steps returns a clamped number of steps based on string width
func Steps(s string) int {
	width := lipgloss.Width(s)
	const (
		minSteps = 5
		maxSteps = 120
		minWidth = 4
		maxWidth = 80
	)
	clampedWidth := min(max(width, minWidth), maxWidth)
	invScale := float64(clampedWidth-minWidth) / float64(maxWidth-minWidth)
	return int(float64(minSteps) + invScale*float64(maxSteps-minSteps))
}

// New initializes and caches frame data for a given string
func (gr *Renderer) New(s string, steps int, hexColors ...string) *FrameData {
	if s == "" {
		return nil
	}
	if w := lipgloss.Width(s); steps < w {
		steps = w
	}

	gr.mu.RLock()
	data, exists := gr.cache[s]
	gr.mu.RUnlock()
	if exists {
		return data
	}

	runes := []rune(s)
	total := len(runes)
	var frames [][]termenv.RGBColor

	switch len(hexColors) {
	case 0:
		return nil
	case 1:
		// Single-color: repeat the same RGBColor
		frame := make([]termenv.RGBColor, total)
		rgb := termenv.RGBColor(hexColors[0])
		for i := range frame {
			frame[i] = rgb
		}
		frames = [][]termenv.RGBColor{frame}
	default:
		// Multi-color gradient
		colors := make([]colorful.Color, len(hexColors))
		for i, hex := range hexColors {
			c, err := colorful.Hex(hex)
			if err != nil {
				return nil
			}
			colors[i] = c
		}
		// ensure wrap-around
		if colors[0] != colors[len(colors)-1] {
			colors = append(colors, colors[0])
		}
		frames = gr.precomputeFrames(runes, steps, colors)
	}

	data = &FrameData{message: s, frames: frames, steps: steps, pool: gr.pool}
	gr.mu.Lock()
	gr.cache[s] = data
	gr.mu.Unlock()
	return data
}

func (gr *Renderer) RenderAdvance(s string) string {
	gr.Advance(s)
	return gr.RenderCurrent(s)
}

// RenderCurrent returns the current frame as a colored string
func (gr *Renderer) RenderCurrent(s string) string {
	gr.mu.RLock()
	data, ok := gr.cache[s]
	gr.mu.RUnlock()
	if !ok {
		color := [...][]string{
			PastelColors,
			BlueGreenYellow,
			PastelRainbow,
			PastelGreenBlue,
			GreenPinkBlue,
		}

		gr.New(s, Steps(s), color[rand.IntN(len(color))]...)
	}

	result := gr.pool.Get()
	defer gr.pool.Put(result)
	runes := []rune(s)
	for i, r := range runes {
		color := data.frames[data.index][i]
		styled := termenv.String(string(r)).Foreground(color).String()
		result.WriteString(styled)
	}
	return result.String()
}

func (f *FrameData) String() string {
	if f == nil {
		return ""
	}
	result := f.pool.Get()
	defer f.pool.Put(result)
	for i, r := range f.message {
		hex := f.frames[f.index][i]
		styled := termenv.String(string(r)).
			Foreground(hex).
			String()
		result.WriteString(styled)
	}
	return result.String()
}

func (f *FrameData) Advance() {
	f.index = (f.index + 1) % len(f.frames)
}

// Reset resets the animation to the first frame
func (f *FrameData) Reset() {
	f.index = 0
}

// Write writes the current frame directly to an io.Writer
func (gr *Renderer) Write(s string, w io.Writer) error {
	gr.mu.RLock()
	data, ok := gr.cache[s]
	gr.mu.RUnlock()
	if !ok {
		return gr.Write(s, w)
	}

	for i, r := range s {
		color := data.frames[data.index][i]
		styled := termenv.String(string(r)).Foreground(color).String()
		if _, err := w.Write([]byte(styled)); err != nil {
			return err
		}
	}
	return nil
}

// Advance moves to the next frame
func (gr *Renderer) Advance(s string) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	if data, ok := gr.cache[s]; ok {
		data.index = (data.index + 1) % len(data.frames)
	}
}

func (gr *Renderer) AdvanceAll() {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	for _, data := range gr.cache {
		data.index = (data.index + 1) % len(data.frames)
	}
}

// Reset resets the animation to the first frame
func (gr *Renderer) Reset(s string) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	if data, ok := gr.cache[s]; ok {
		data.index = 0
	}
}

// Delete removes the cached data for a string
func (gr *Renderer) Delete(s string) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	delete(gr.cache, s)
}

// precomputeFrames builds all frames as RGBColor arrays
func (gr *Renderer) precomputeFrames(runes []rune, steps int, colors []colorful.Color) [][]termenv.RGBColor {
	total := len(runes)
	segments := len(colors) - 1
	var frames [][]termenv.RGBColor

	for step := range steps {
		progress := float64(step) / float64(steps)

		frame := make([]termenv.RGBColor, total)
		var wg sync.WaitGroup
		wg.Add(total)
		for i := range total {
			go func() {
				base := 0.0
				if total > 1 {
					base = float64(i) / float64(total-1)
				}

				r := math.Mod(base+progress, 1.0)
				seg := int(r * float64(segments))
				if seg >= segments {
					seg = segments - 1
				}
				local := r*float64(segments) - float64(seg)

				c := colors[seg].BlendLab(colors[seg+1], local).Clamped()
				frame[i] = termenv.RGBColor(c.Hex())
				wg.Done()
			}()
		}
		wg.Wait()
		frames = append(frames, frame)
	}

	return frames
}
