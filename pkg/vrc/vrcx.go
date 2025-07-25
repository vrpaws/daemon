package vrc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"vrc-moments/pkg/exif"
)

type Metadata struct {
	Application string `json:"application"`
	Version     int64  `json:"version"`
	Author      User   `json:"author"`
	World       World  `json:"world"`
	Players     []User `json:"players"`
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type World struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	InstanceID string `json:"instanceId"`
}

func GetVRCXDataFromFile(path string) (Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	return getVRCXData[Metadata](f)
}

func GetVRCXData(r io.ReadSeeker) (Metadata, error) {
	return getVRCXData[Metadata](r)
}

var EOF = errors.New("EOF")

// Write parses bytes passed onto it to exif.Parse
// It only processes the first p bytes passed onto it and always returns an error.
// A successful Metadata read returns EOF so that users of io.Copy will return early.
func (m *Metadata) Write(p []byte) (n int, err error) {
	if m == nil {
		return 0, errors.New("nil metadata")
	}
	if len(p) == 0 {
		return 0, errors.New("empty metadata")
	}

	data, err := getVRCXData[Metadata](bytes.NewReader(p))
	if err != nil {
		return 0, fmt.Errorf("error parsing VRCX data: %w", err)
	}
	*m = data

	return len(p), EOF
}

func getVRCXData[T Metadata](r io.ReadSeeker) (T, error) {
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
