package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/sqweek/dialog"
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

func Random[T any](v ...T) T {
	return v[rand.IntN(len(v))]
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

func LogOutput(writer io.Writer) (io.Writer, func()) {
	logFolder := filepath.Join(ConfigDirectory, "logs")
	logfile := filepath.Join(logFolder, fmt.Sprintf("log-%s.txt", time.Now().Format("2006-01-02_15-04-05")))
	err := os.MkdirAll(logFolder, 0755)
	if err != nil {
		return io.Discard, func() {}
	}
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return io.Discard, func() {}
	}

	mw := io.MultiWriter(writer, f)
	r, w := io.Pipe()

	log.SetOutput(mw)

	exit := make(chan struct{})
	go func() {
		_, _ = io.Copy(mw, r)
		close(exit)
	}()

	return f, func() {
		_ = w.Close()
		<-exit
		fi, err := f.Stat()
		_ = f.Close()
		if err == nil && fi.Size() == 0 {
			_ = os.Remove(logfile)
		}
	}
}

func SelectVRChatDirectory(path string) (string, []string, error) {
	startDir := "."
	homedir, err := os.UserHomeDir()
	if err == nil {
		if path != "" && FileExists(path) {
			startDir = path
		} else if vrcDefault := filepath.Join(homedir, "Pictures", "VRChat"); FileExists(vrcDefault) {
			startDir = vrcDefault
		}
	}

	directory, err := dialog.Directory().SetStartDir(startDir).Title("Choose your VRChat Photos folder").Browse()
	if err != nil {
		return "", nil, fmt.Errorf("error getting directory: %w", err)
	}

	if strings.HasPrefix(directory, homedir) {
		directory = filepath.Join("~", strings.TrimPrefix(directory, homedir))
	}

	if strings.HasSuffix(directory, "***") {
		directory = strings.TrimRight(directory, `*\/`+string(filepath.Separator))
	}

	prints := filepath.Join("!"+directory, "Prints", "***")
	stickers := filepath.Join("!"+directory, "Stickers", "***")
	directory = filepath.Join(directory, "***")
	directory = strings.ReplaceAll(directory, `\`, "/")

	patterns, err := ExpandPatterns(true, false, directory, prints, stickers)
	if err != nil {
		return "", nil, err
	}

	return directory, patterns, nil
}
