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
