package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"vrc-moments/pkg/exif"
	"vrc-moments/pkg/vrcx"
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
			files = append(files, matches...)
		}
	}

	return files, nil
}

func GetVRCXDataFromFile(path string) (vrcx.Screenshot, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	return getVRCXData[vrcx.Screenshot](f)
}

func GetVRCXData(r io.ReadSeeker) (vrcx.Screenshot, error) {
	return getVRCXData[vrcx.Screenshot](r)
}

func getVRCXData[T vrcx.Screenshot](r io.ReadSeeker) (T, error) {
	entries, err := exif.Parse(r)
	if err != nil {
		return T{}, fmt.Errorf("parsing exif: %w", err)
	}
	if len(entries) < 1 {
		return T{}, errors.New("no exif")
	}

	for _, entry := range entries {
		if entry.ChunkType != exif.ChunkiTXT || entry.Keyword != exif.KeywordDescription {
			continue
		}
		var t T
		if err = json.NewDecoder(bytes.NewReader(entry.Text)).Decode(&t); err != nil {
			continue
		}
		return t, nil
	}

	return T{}, fmt.Errorf("could not parse exif: %w", err)
}
