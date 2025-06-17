package vrc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var DefaultLogPath = filepath.Join(os.Getenv("AppData"), "..", "LocalLow", "VRChat", "VRChat")

func GetUsername(logPath string) ([]string, error) {
	if logPath == "" {
		logPath = DefaultLogPath
	}

	username, err := ExtractUsernameFromLogs(logPath)
	if err == nil {
		return []string{username}, nil
	}

	return nil, fmt.Errorf("failed to load username")
}

var usernameRegex = regexp.MustCompile(`User Authenticated: (.+) \(usr_[a-z0-9-]+\)$`)
var roomNameRegex = regexp.MustCompile(`\[Behaviour] Joining or Creating Room: ([^\r\n]+)$`)
var imageRegex = regexp.MustCompile(`\[Image Download] Attempting to load image from URL '([^']+)'$`)
var stringRegex = regexp.MustCompile(`\[String Download] Attempting to load String from URL '([^']+)'$`)

func ExtractUsernameFromLogs(logDir string) (string, error) {
	if logDir == "" {
		logDir = DefaultLogPath
	}
	return ExtractReader(logDir, Scanner, usernameRegex)
}

func ExtractCurrentRoomName(logDir string) (string, error) {
	if logDir == "" {
		logDir = DefaultLogPath
	}
	return ExtractReader(logDir, ReverseLines, roomNameRegex)
}

type logFile struct {
	path string
	time time.Time
}

func ExtractReader(logDir string, scanner func(io.ReadSeeker) iter.Seq2[string, error], match *regexp.Regexp) (string, error) {
	pattern := filepath.Join(logDir, "output_log_*.txt")

	logFiles, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to list log files: %w", err)
	}

	if len(logFiles) == 0 {
		return "", errors.New("no log files found")
	}

	var logs []logFile
	for _, file := range logFiles {
		baseName := filepath.Base(file)
		timestamp := strings.TrimPrefix(baseName, "output_log_")
		timestamp = strings.TrimSuffix(timestamp, ".txt")

		fileTime, err := time.Parse("2006-01-02_15-04-05", timestamp)
		if err != nil {
			continue
		}

		logs = append(logs, logFile{path: file, time: fileTime})
	}

	if len(logs) == 0 {
		return "", errors.New("no log files found")
	}

	slices.SortFunc(logs, func(a, b logFile) int {
		return b.time.Compare(a.time) // Sort by time descending
	})

	for _, l := range logs {
		file, err := os.Open(l.path)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w\n", l.path, err)
		}

		for line := range scanner(file) {
			matches := match.FindStringSubmatch(line)
			if len(matches) > 1 {
				file.Close()
				return matches[1], nil
			}
		}

		file.Close()
	}

	return "", errors.New("no matching line found in any log files")
}

func Scanner(r io.ReadSeeker) iter.Seq2[string, error] {
	scanner := bufio.NewScanner(r)
	return func(yield func(string, error) bool) {
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				yield("", err)
				return
			}
			line := scanner.Text()
			if line == "" {
				continue
			}
			if !yield(line, nil) {
				return
			}
		}
	}
}

// ReverseLines returns an iterator that yields each line of rs
// in reverse order (last line first). Lines are returned without
// trailing newline or carriage-return bytes. If a seek or read
// error occurs, iteration stops silently.
func ReverseLines(rs io.ReadSeeker) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		defer rs.Seek(0, io.SeekStart)

		size, err := rs.Seek(0, io.SeekEnd)
		if err != nil {
			yield("", err)
			return
		}

		const chunkSize = 4096
		buf := make([]byte, chunkSize)
		pos, carry := size, make([]byte, 0, chunkSize)

		for {
			// if we have a newline, pull out the last line
			if idx := bytes.LastIndexByte(carry, '\n'); idx >= 0 {
				line := carry[idx+1:]
				carry = carry[:idx]

				// strip trailing CR
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				// skip empty
				if len(line) > 0 {
					if !yield(string(line), nil) {
						return
					}
				}
				continue
			}

			// reached start of file
			if pos == 0 {
				if len(carry) > 0 {
					line := carry
					if line[len(line)-1] == '\r' {
						line = line[:len(line)-1]
					}
					if len(line) > 0 {
						yield(string(line), nil)
					}
				}
				return
			}

			// back up by chunkSize (or to zero)
			readSize := chunkSize
			if pos < int64(chunkSize) {
				readSize = int(pos)
			}
			pos -= int64(readSize)

			if _, err = rs.Seek(pos, io.SeekStart); err != nil {
				yield("", err)
				return
			}
			n, err := rs.Read(buf[:readSize])
			if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
				yield("", err)
				return
			}
			if n == 0 {
				return // nothing more to read
			}

			carry = append(buf[:n], carry...)
		}
	}
}
