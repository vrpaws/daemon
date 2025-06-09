package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// ExpandPatterns walks or globs each pattern and returns all files.
func ExpandPatterns(patterns ...string) ([]string, error) {
	var files []string
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = os.Getenv("USERPROFILE")
	}
	if homedir == "" {
		return nil, fmt.Errorf("cannot get home dir %w", err)
	}
	for _, pat := range patterns {
		if strings.HasPrefix(pat, "~/") {
			pat = filepath.Join(homedir, pat[2:])
		}
		// if it contains "**", do a recursive Walk
		if strings.Contains(pat, "**") {
			parts := strings.SplitN(pat, "**", 2)
			root, suffix := parts[0], strings.TrimLeft(parts[1], `\/`)
			err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return err
				}
				// try to match the "**/...suffix" portion against the tail of a path
				rel := filepath.Base(path)
				matched, matchErr := filepath.Match(suffix, rel)
				if matchErr != nil {
					return matchErr
				}
				if matched {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return files, err
			}
		} else {
			// simple one-shot Glob
			matches, err := filepath.Glob(pat)
			if err != nil {
				return files, err
			}
			for _, match := range matches {
				fi, err := os.Stat(match)
				if err != nil {
					continue
				}
				if !fi.IsDir() {
					files = append(files, match)
				}
			}
		}
	}

	return files, nil
}
