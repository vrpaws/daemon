package vrpaws

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"vrc-moments/pkg/flight"
)

type VRPaws struct {
	client      *http.Client
	context     context.Context
	accessToken string
	tokenCache  flight.Cache[string, bool]
	remote      *url.URL
}

func NewVRPaws(remote *url.URL, ctx context.Context, token string) *VRPaws {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "token", token)

	v := &VRPaws{
		client:      &http.Client{Timeout: 5 * time.Minute},
		context:     ctx,
		accessToken: token,
		remote:      remote,
	}
	v.tokenCache = flight.NewCache(v.validToken)

	return v
}

func (s *VRPaws) ValidToken(token string) error {
	valid, err := s.tokenCache.Get(token)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("token is not valid")
	}

	return errors.New("not yet implemented")
}

func (s *VRPaws) validToken(token string) (bool, error) {
	return false, errors.New("not yet implemented")
}

// Deprecated: username is not required, use ValidToken
func (s *VRPaws) ValidUser(string) error {
	return nil
}

func (s *VRPaws) SetRemote(remote string) error {
	parsed, err := url.Parse(remote)
	if err != nil {
		return err
	}
	s.remote = parsed

	return nil
}
