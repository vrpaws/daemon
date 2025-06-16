package api

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"vrc-moments/pkg/flight"
)

type LocalServer struct {
	usernameCache flight.Cache[string, bool]
	tokenCache    flight.Cache[string, bool]
	remote        *url.URL
}

func NewLocal(remote *url.URL) *LocalServer {
	return &LocalServer{
		usernameCache: flight.NewCache(valid[string]),
		tokenCache:    flight.NewCache(valid[string]),
		remote:        remote,
	}
}

func valid[T any](T) (bool, error) {
	return true, nil
}

func (s *LocalServer) Upload(ctx context.Context, payload UploadPayload) (bool, error) {
	return false, errors.New("not yet implemented")
}

func (s *LocalServer) ValidUser(username string) (bool, error) {
	valid, err := s.usernameCache.Get(username)
	if err != nil {
		return false, fmt.Errorf("could not determine if user %q is valid: %w", username, err)
	}
	if !valid {
		return false, fmt.Errorf("user %q is not valid", username)
	}

	return false, errors.New("not yet implemented")
}

func (s *LocalServer) ValidToken(token string) (bool, error) {
	valid, err := s.tokenCache.Get(token)
	if err != nil {
		return false, fmt.Errorf("could not determine if token %q is valid: %w", token, err)
	}
	if !valid {
		return false, fmt.Errorf("token %q is not valid", token)
	}

	return false, errors.New("not yet implemented")
}

func (s *LocalServer) SetRemote(remote string) error {
	parsed, err := url.Parse(remote)
	if err != nil {
		return err
	}
	s.remote = parsed
	return nil
}
