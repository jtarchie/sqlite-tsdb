package sdk

import (
	"fmt"
	"net/http"
)

type StatsPayload struct {
	Count struct {
		Insert uint64 `json:"insert"`
	} `json:"count"`
}

func (c *Client) Stats() (*StatsPayload, error) {
	payload := &StatsPayload{}

	client := c.client

	response, err := client.R().
		SetSuccessResult(payload).
		Get(fmt.Sprintf("%s/api/stats", c.endpoint))
	if err != nil {
		return nil, fmt.Errorf("could not GET /api/stats: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		return payload, nil
	}

	return nil, fmt.Errorf("could not load /api/status")
}
