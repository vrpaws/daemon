package vrc

import (
	"bufio"
	"errors"
	"fmt"
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
	return ExtractReader(logDir, usernameRegex)
}

func ExtractCurrentRoomName(logDir string) (string, error) {
	if logDir == "" {
		logDir = DefaultLogPath
	}
	return ExtractReader(logDir, roomNameRegex)
}

type logFile struct {
	path string
	time time.Time
}

func ExtractReader(logDir string, match *regexp.Regexp) (string, error) {
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

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			matches := match.FindStringSubmatch(scanner.Text())
			if len(matches) > 1 {
				file.Close()
				return matches[1], nil
			}
		}
		file.Close()
	}

	return "", errors.New("no matching line found in any log files")
}
