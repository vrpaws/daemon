package vrpaws

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"vrc-moments/pkg/flight"
)

type VRPaws struct {
	client        *http.Client
	context       context.Context
	accessToken   string
	usernameCache flight.Cache[string, bool]
	remote        *url.URL
}

func NewVRPaws(remote *url.URL, ctx context.Context, accessToken string) *VRPaws {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "token", accessToken)

	return &VRPaws{
		client:      &http.Client{Timeout: 5 * time.Minute},
		context:     ctx,
		accessToken: accessToken,
		usernameCache: flight.NewCache(func(string) (bool, error) {
			return true, nil
		}),
		remote: remote,
	}
}
