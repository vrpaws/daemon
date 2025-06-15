package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

var ConfigDirectory string
var ConfigPath string

func init() {
	dir, err := os.UserConfigDir()
	if err == nil {
		if HasUpperCase(dir) {
			ConfigDirectory = filepath.Join(dir, "VRPaws")
		} else {
			ConfigDirectory = filepath.Join(dir, "vrpaws")
		}
	}

	ConfigPath = filepath.Join(ConfigDirectory, "vrpaws-config.json")
}

func HasUpperCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func Map[T any](s []T, f func(T) T) {
	for i := range s {
		s[i] = f(s[i])
	}
}

func Scan[From any, To any](s []From, f func(From) To) []To {
	if s == nil {
		return nil
	}
	out := make([]To, len(s))
	for i := range s {
		out[i] = f(s[i])
	}
	return out
}

func DigitCount(i int) int {
	if i == 0 {
		return 1
	}
	count := 0
	for i != 0 {
		i /= 10
		count++
	}
	return count
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func Encode(v any) (io.Reader, error) {
	var buf bytes.Buffer
	return &buf, json.NewEncoder(&buf).Encode(v)
}

func EncodeToFile(path string, v any) error {
	if base := filepath.Dir(path); base != "." {
		err := os.MkdirAll(base, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create directory %q: %w", base, err)
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func Decode[T any](r io.Reader) (T, error) {
	var t T
	return t, json.NewDecoder(r).Decode(&t)
}

func DecodeFromFile[T any](path string) (T, error) {
	var t T
	f, err := os.Open(path)
	if err != nil {
		return t, err
	}
	defer f.Close()
	return t, json.NewDecoder(f).Decode(&t)
}

func RemoveExtension(filename string) string {
	return strings.TrimSuffix(filename, path.Ext(filename))
}

// ExpandPatterns walks or globs each pattern and returns all matching files,
// applying exclusions inline.  Supported exclusions:
//   - "!path"       – exclude exactly that directory (but not its contents)
//   - "!path/*"     – exclude files directly under that directory
//   - "!path/***"   – exclude that directory and everything under it
func ExpandPatterns(patterns ...string) ([]string, error) {
	var files []string

	// resolve ~/
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = os.Getenv("USERPROFILE")
	}
	if homedir == "" {
		return nil, fmt.Errorf("cannot get home dir: %w", err)
	}

	// classify patterns
	var includes []string
	var exclDirOnly []string
	var exclFilesLevel []string
	var exclRecursive []string

	for _, pat := range patterns {
		// detect exclusion before cleaning
		isExclude := strings.HasPrefix(pat, "!")
		p := pat
		if isExclude {
			p = strings.TrimPrefix(pat, "!")
		}
		// expand ~/
		var sep string
		if sep = "~" + string(filepath.Separator); strings.HasPrefix(p, sep) {
			p = filepath.Join(homedir, p[len(sep):])
		} else if strings.HasPrefix(p, `~/`) || strings.HasPrefix(p, `~\`) {
			p = filepath.Join(homedir, p[2:])
		}
		// now clean the actual filesystem path
		p = filepath.Clean(p)

		if isExclude {
			e := p
			sep := string(os.PathSeparator)
			switch {
			case strings.HasSuffix(e, sep+"***"):
				exclRecursive = append(exclRecursive, strings.TrimSuffix(e, sep+"***"))
			case strings.HasSuffix(e, sep+"*"):
				exclFilesLevel = append(exclFilesLevel, strings.TrimSuffix(e, sep+"*"))
			default:
				exclDirOnly = append(exclDirOnly, e)
			}
		} else {
			includes = append(includes, p)
		}
	}

	// helper to test exclusion
	isExcluded := func(path string) bool {
		// dir-only: skip if path == dir (but files under are OK)
		for _, d := range exclDirOnly {
			if path == d {
				return true
			}
		}
		// files-only: skip if parent dir == target
		for _, d := range exclFilesLevel {
			if filepath.Dir(path) == d {
				return true
			}
		}
		// recursive: skip anything under target
		for _, d := range exclRecursive {
			if strings.HasPrefix(path, d+string(os.PathSeparator)) {
				return true
			}
		}
		return false
	}

	for _, pat := range includes {
		if strings.Contains(pat, "**") {
			// recursive walk
			parts := strings.SplitN(pat, "**", 2)
			root := filepath.Clean(parts[0])
			suffix := strings.TrimLeft(parts[1], string(filepath.Separator))

			err := filepath.Walk(root, func(fp string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fi.IsDir() {
					// only skip subtree when this dir is in the recursive-exclude list
					for _, d := range exclRecursive {
						if fp == d {
							return filepath.SkipDir
						}
					}
					return nil
				}
				if match, _ := filepath.Match(suffix, filepath.Base(fp)); match {
					if !isExcluded(fp) {
						files = append(files, fp)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			// one-shot glob
			matches, err := filepath.Glob(pat)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				fi, err := os.Stat(m)
				if err != nil || fi.IsDir() {
					continue
				}
				if !isExcluded(m) {
					files = append(files, m)
				}
			}
		}
	}

	return files, nil
}

func GradientText(s string, hexColors ...string) string {
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

func LogOutput(writer io.Writer) func() {
	logFolder := filepath.Join(ConfigDirectory, "logs")
	logfile := filepath.Join(logFolder, fmt.Sprintf("log-%s.txt", time.Now().Format("2006-01-02_15-04-05")))
	err := os.MkdirAll(logFolder, 0755)
	if err != nil {
		return func() {}
	}
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return func() {}
	}

	mw := io.MultiWriter(writer, f)
	r, w := io.Pipe()

	log.SetOutput(mw)

	exit := make(chan struct{})
	go func() {
		_, _ = io.Copy(mw, r)
		close(exit)
	}()

	return func() {
		_ = w.Close()
		<-exit
		fi, err := f.Stat()
		_ = f.Close()
		if err == nil && fi.Size() == 0 {
			_ = os.Remove(logfile)
		}
	}
}
