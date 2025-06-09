package exif

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

type ChunkType = string

const (
	ChunkiTXT ChunkType = "iTXt"
	ChunkEND  ChunkType = "IEND"
)

type KeywordType = string

const (
	KeywordDescription KeywordType = "Description"
)

// Entry represents a single iTXt chunk parsed from a PNG.
type Entry struct {
	// ChunkType is the PNG chunk type
	ChunkType ChunkType
	// Keyword is the keyword (e.g. "Description")
	Keyword KeywordType
	// Compressed reports whether the text was zlib-compressed
	Compressed bool
	// LanguageTag is the optional language tag (may be empty)
	LanguageTag string
	// TranslatedKeyword is the optional translated keyword (may be empty)
	TranslatedKeyword string
	// Text is the final UTF-8 text payload
	Text []byte
}

// Parse scans the provided PNG data (via io.ReadSeeker).
func Parse(r io.ReadSeeker) ([]Entry, error) {
	// Skip PNG signature (8 bytes)
	if _, err := r.Seek(8, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking past PNG signature: %w", err)
	}

	var entries []Entry

read:
	for {
		// Read chunk length (4 bytes, big-endian)
		var length uint32
		if err := binary.Read(r, binary.BigEndian, &length); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading chunk length: %w", err)
		}

		// Read chunk type (4-byte ASCII)
		chunkTypeBytes := make([]byte, 4)
		if _, err := io.ReadFull(r, chunkTypeBytes); err != nil {
			return nil, fmt.Errorf("reading chunk type: %w", err)
		}
		cType := string(chunkTypeBytes)

		// Read chunk data
		data := make([]byte, length)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("reading chunk data for %s: %w", cType, err)
		}

		// Skip CRC (4 bytes)
		if _, err := r.Seek(4, io.SeekCurrent); err != nil {
			return nil, fmt.Errorf("skipping CRC: %w", err)
		}

		switch cType {
		case ChunkiTXT:
			entry, err := parseITXtData(cType, data)
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		case ChunkEND:
			break read
		default:
			continue
		}
	}

	return entries, nil
}

// parseITXtData extracts fields from raw iTXt chunk data.
func parseITXtData(cType string, data []byte) (Entry, error) {
	// Split keyword and rest at first NUL
	parts := bytes.SplitN(data, []byte{0}, 2)
	if len(parts) < 2 {
		return Entry{}, fmt.Errorf("invalid iTXt chunk: missing keyword field")
	}
	keyword := string(parts[0])
	rest := parts[1]
	if len(rest) < 2 {
		return Entry{}, fmt.Errorf("invalid iTXt chunk: missing compression flags")
	}

	// compression flag: 0 = uncompressed, 1 = compressed
	compressed := rest[0] == 1
	// skip compression method byte (rest[1])
	body := rest[2:]

	// Split language tag, translated keyword, and actual text
	fields := bytes.SplitN(body, []byte{0}, 3)
	if len(fields) < 3 {
		return Entry{}, fmt.Errorf("invalid iTXt chunk: missing fields after compression flags")
	}
	languageTag := string(fields[0])
	translatedKey := string(fields[1])
	textData := fields[2]

	// Decompress if needed
	if compressed {
		zr, err := zlib.NewReader(bytes.NewReader(textData))
		if err != nil {
			return Entry{}, fmt.Errorf("zlib decompression failed: %w", err)
		}
		var out bytes.Buffer
		if _, err := io.Copy(&out, zr); err != nil {
			zr.Close()
			return Entry{}, fmt.Errorf("reading decompressed data: %w", err)
		}
		zr.Close()
		textData = out.Bytes()
	}

	return Entry{
		ChunkType:         cType,
		Keyword:           keyword,
		Compressed:        compressed,
		LanguageTag:       languageTag,
		TranslatedKeyword: translatedKey,
		Text:              textData,
	}, nil
}
