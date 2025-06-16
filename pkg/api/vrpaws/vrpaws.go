package vrpaws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	lib "vrc-moments/pkg"
	"vrc-moments/pkg/flight"
)

type VRPaws struct {
	client      *http.Client
	context     context.Context
	accessToken string
	tokenCache  flight.Cache[string, *Me]
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

func (s *VRPaws) ValidToken(token string) (*Me, error) {
	me, err := s.tokenCache.Get(token)
	if err != nil {
		return nil, err
	}

	if me == nil || me.User.AccessToken != token {
		return nil, errors.New("invalid token")
	}

	return me, nil
}

func (s *VRPaws) validToken(token string) (*Me, error) {
	if token == "" {
		return nil, errors.New("invalid token")
	}

	u := *s.remote
	u.Path = path.Join(u.Path, "users", "@me")

	q := u.Query()
	q.Add("accessToken", token)
	u.RawQuery = q.Encode()

	resp, err := s.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error getting token response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get token request failed with status %s", resp.Status)
	}

	me, err := lib.Decode[*Me](resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding response from: %w", err)
	}

	return me, nil
}

func (s *VRPaws) SetRemote(remote string) error {
	parsed, err := url.Parse(remote)
	if err != nil {
		return err
	}
	s.remote = parsed

	return nil
}
