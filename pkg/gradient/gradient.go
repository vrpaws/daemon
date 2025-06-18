package gradient

import (
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

var PastelColors = []string{"#00E2FD", "#6D90FA", "#FF22EE", "#FF8D7A", "#FFC851"}
var BlueGreenYellow = []string{"#2A7B9B", "#57C785", "#EDDD53"}
var PastelRainbow = []string{"#C5F9D7", "#F7D486", "#F27A7D"}
var PastelGreenBlue = []string{"#C2E59C", "#64B3F4"}
var GreenPinkBlue = []string{"#CAEFD7", "#F5BFD7", "#ABC9E9"}

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

var Global = NewGradientRenderer()

type frameData struct {
	frames [][]string
	index  int
	steps  int
}

type Renderer struct {
	cache map[string]*frameData
	mu    sync.RWMutex
}

func NewGradientRenderer() *Renderer {
	return &Renderer{
		cache: make(map[string]*frameData),
	}
}

func StepsFromDuration(width int, duration time.Duration, fps float64) int {
	frames := int(float64(duration.Nanoseconds()) * fps / float64(time.Second))
	steps := frames

	if width > 0 {
		steps = max(steps, width)
	}

	return min(max(steps, 5), 120)
}

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
	steps := int(float64(minSteps) + invScale*float64(maxSteps-minSteps))
	return steps
}

func (gr *Renderer) New(s string, steps int, hexColors ...string) {
	if s == "" {
		return
	}
	if w := lipgloss.Width(s); steps < w {
		steps = w
	}

	gr.mu.RLock()
	_, exists := gr.cache[s]
	gr.mu.RUnlock()
	if exists {
		return
	}

	runes := []rune(s)
	total := len(runes)
	var frames [][]string

	switch len(hexColors) {
	case 0:
		return
	case 1:
		frame := make([]string, total)
		for i := 0; i < total; i++ {
			frame[i] = hexColors[0]
		}
		frames = [][]string{frame}
	default:
		colors := make([]colorful.Color, len(hexColors))
		for i, hex := range hexColors {
			c, err := colorful.Hex(hex)
			if err != nil {
				return
			}
			colors[i] = c
		}
		if colors[0] != colors[len(colors)-1] {
			colors = append(colors, colors[0]) // ensure the first and last colors are the same
		}
		frames = precomputeFrames(runes, steps, colors)
	}

	gr.mu.Lock()
	gr.cache[s] = &frameData{frames: frames, steps: steps}
	gr.mu.Unlock()
}

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

	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(data.frames[data.index][i]))
		result.WriteString(style.Render(string(r)))
	}
	return result.String()
}

func (gr *Renderer) Advance(s string) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	if data, ok := gr.cache[s]; ok {
		data.index = (data.index + 1) % len(data.frames)
	}
}

func (gr *Renderer) Reset(s string) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	if data, ok := gr.cache[s]; ok {
		data.index = 0
	}
}

func precomputeFrames(runes []rune, steps int, colors []colorful.Color) [][]string {
	total := len(runes)
	segments := len(colors) - 1
	var frames [][]string

	for step := 0; step < steps; step++ {
		progress := float64(step) / float64(steps-1)
		frame := make([]string, total)

		for i := 0; i < total; i++ {
			var ratio float64
			if total > 1 {
				ratio = float64(i)/float64(total-1) + progress/float64(steps)
			}
			ratio -= float64(int(ratio)) // wrap into [0,1)

			segment := int(ratio * float64(segments))
			if segment >= segments {
				segment = segments - 1
			}
			local := (ratio * float64(segments)) - float64(segment)
			blend := colors[segment].BlendLab(colors[segment+1], local).Clamped()
			frame[i] = blend.Hex()
		}
		frames = append(frames, frame)
	}
	return frames
}
