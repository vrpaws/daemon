package api

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"

	"vrc-moments/pkg/vrc"
)

type Server[user any] interface {
	ValidToken(string) (user, error) // integrate with flight.Cache to prevent api spam
	Upload(context.Context, UploadPayload) error
	SetRemote(string) error
}

type UploadPayload struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	File     *File  `json:"file"`
}

type File struct {
	Date     time.Time     `json:"date"`
	Filename string        `json:"filename"`
	MD5Hash  string        `json:"md5"`
	SHA256   string        `json:"sha256"`
	XXHash   string        `json:"xx"`
	Metadata *vrc.Metadata `json:"metadata"`
	Data     io.ReadCloser `json:"-"`
}

func (f *File) Close() error {
	if f.Data == nil {
		return errors.New("file is nil")
	}

	return f.Data.Close()
}

func OpenFile(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	f, err := Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %v", err)
	}

	f.Filename = filepath.Base(path)
	f.Data = file

	stat, err := file.Stat()
	if err != nil {
		f.Date = time.Now()
	} else {
		f.Date = stat.ModTime()
	}

	return f, nil
}

func Parse(file io.ReadSeeker) (*File, error) {
	sha256, err := digest(sha256.New(), file)
	if err != nil {
		return nil, err
	}

	hash, err := digest(md5.New(), file)
	if err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}

	xxhash, err := digest(xxhash.New(), file)
	if err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}

	metadata, err := digest(new(vrc.Metadata), file)
	if err != nil {
		metadata = nil
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error seeking to beginning of file: %w", err)
	}

	return &File{
		SHA256:   sum(sha256),
		MD5Hash:  sum(hash),
		XXHash:   sum(xxhash),
		Metadata: metadata,
	}, nil
}

func digest[T io.Writer](t T, file io.ReadSeeker) (T, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return t, fmt.Errorf("error seeking to beginning of file: %w", err)
	}

	if _, err := io.Copy(t, file); err != nil && !errors.Is(err, vrc.EOF) {
		return t, fmt.Errorf("error hashing file: %w", err)
	}

	return t, nil
}

func sum(hash hash.Hash) string {
	return hex.EncodeToString(hash.Sum(nil))
}
