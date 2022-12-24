package sdk

import (
	"fmt"
	"net/url"

	"github.com/imroc/req/v3"
)

type Client struct {
	client   *req.Client
	endpoint string
}

func New(host string) (*Client, error) {
	uri, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("invalid host provided: %w", err)
	}

	return &Client{
		client:   req.C(),
		endpoint: uri.String(),
	}, nil
}
