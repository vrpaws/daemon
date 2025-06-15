package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ExpandPatterns walks or globs each pattern and returns all matching files or
// directories (depending on the flags), applying exclusions inline.
// Supported exclusions:
//   - "!path"       – exclude exactly that directory (but not its contents)
//   - "!path/*"     – exclude files directly under that directory
//   - "!path/***"   – exclude that directory and everything under it
func ExpandPatterns(matchDirs, matchFiles bool, patterns ...string) ([]string, error) {
	// default to files-only
	if !matchDirs && !matchFiles {
		matchFiles = true
	}

	pm, err := NewPatternMatcher(matchDirs, matchFiles, patterns...)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, inc := range pm.includes {
		if strings.Contains(inc, "**") {
			parts := strings.SplitN(inc, "**", 2)
			root := filepath.Clean(parts[0])
			err := filepath.Walk(root, func(fp string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// skip whole subtree if this dir is recursive-excluded
				if fi.IsDir() {
					if slices.Contains(pm.exclRecursive, fp) {
						return filepath.SkipDir
					}
				}
				ok, err := pm.Matches(fp)
				if err != nil {
					if os.IsNotExist(err) {
						return nil
					}
					return err
				}
				if ok {
					results = append(results, fp)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			matches, err := filepath.Glob(inc)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				ok, err := pm.Matches(m)
				if err != nil {
					continue
				}
				if ok {
					results = append(results, m)
				}
			}
		}
	}

	return results, nil
}

// PatternMatcher holds include/exclude rules plus file/dir flags.
type PatternMatcher struct {
	includes       []string
	exclDirOnly    []string
	exclFilesLevel []string
	exclRecursive  []string
	matchDirs      bool
	matchFiles     bool
}

// NewPatternMatcher parses the patterns (with ~ expansion) into a matcher.
func NewPatternMatcher(matchDirs, matchFiles bool, patterns ...string) (*PatternMatcher, error) {
	pm := &PatternMatcher{matchDirs: matchDirs, matchFiles: matchFiles}
	if !matchDirs && !matchFiles {
		pm.matchFiles = true
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = os.Getenv("USERPROFILE")
	}
	if homedir == "" {
		return nil, fmt.Errorf("cannot get home dir: %w", err)
	}

	for _, pat := range patterns {
		isExclude := strings.HasPrefix(pat, "!")
		p := pat
		if isExclude {
			p = strings.TrimPrefix(pat, "!")
		}

		// expand ~/ to homedir
		if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(filepath.Separator)) {
			p = filepath.Join(homedir, p[2:])
		}
		p = filepath.Clean(p)

		if isExclude {
			e := p
			sep := string(os.PathSeparator)
			switch {
			case strings.HasSuffix(e, sep+"***"):
				pm.exclRecursive = append(pm.exclRecursive, strings.TrimSuffix(e, sep+"***"))
			case strings.HasSuffix(e, sep+"*"):
				pm.exclFilesLevel = append(pm.exclFilesLevel, strings.TrimSuffix(e, sep+"*"))
			default:
				pm.exclDirOnly = append(pm.exclDirOnly, e)
			}
		} else {
			pm.includes = append(pm.includes, p)
		}
	}

	return pm, nil
}

// Matches reports whether a path (file or dir) passes includes/excludes & flags.
func (pm *PatternMatcher) Matches(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if fi.IsDir() {
		if !pm.matchDirs {
			return false, nil
		}
		if slices.Contains(pm.exclDirOnly, path) {
			return false, nil
		}
	} else {
		if !pm.matchFiles {
			return false, nil
		}
		if slices.Contains(pm.exclFilesLevel, filepath.Dir(path)) {
			return false, nil
		}
	}

	for _, d := range pm.exclRecursive {
		if strings.HasPrefix(path, d+string(os.PathSeparator)) {
			return false, nil
		}
	}

	for _, inc := range pm.includes {
		if strings.Contains(inc, "**") {
			parts := strings.SplitN(inc, "**", 2)
			root := filepath.Clean(parts[0])
			suffix := strings.TrimLeft(parts[1], string(os.PathSeparator))
			if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
				if match, _ := filepath.Match(suffix, filepath.Base(path)); match {
					return true, nil
				}
			}
		} else {
			if match, _ := filepath.Match(inc, path); match {
				return true, nil
			}
		}
	}

	return false, nil
}
