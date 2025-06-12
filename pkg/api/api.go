package api

import (
	"fmt"
	"io"
	"time"

	"vrc-moments/pkg/flight"
	"vrc-moments/pkg/vrc"
)

type Server interface {
	ValidUser(string) error // integrate with flight.Cache to prevent api spam
	Upload(UploadPayload) error
}

type UploadPayload struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	Files    []File `json:"files"`
}

type File struct {
	Date       time.Time      `json:"date"`
	Filename   string         `json:"filename"`
	Screenshot vrc.Screenshot `json:"screenshot"`
	Data       io.Reader      `json:"-"`
}

type LocalServer struct {
	usernameCache flight.Cache[string, bool]
}

func NewServer() *LocalServer {
	return &LocalServer{
		usernameCache: flight.NewCache(func(string) (bool, error) {
			return true, nil
		}),
	}
}

func (s *LocalServer) Upload(payload UploadPayload) error {
	return nil
}

func (s *LocalServer) ValidUser(username string) error {
	valid, err := s.usernameCache.Get(username)
	if err != nil {
		return fmt.Errorf("could not determine if user %q is valid: %w", username, err)
	}
	if !valid {
		return fmt.Errorf("user %q is not valid", username)
	}

	return nil
}

func (s *LocalServer) validUser(_ string) (bool, error) {
	return true, nil
}
