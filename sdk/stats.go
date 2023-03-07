package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type StatsPayload struct {
	Count struct {
		Insert uint64 `json:"insert"`
	} `json:"count"`
}

func (c *Client) Stats() (*StatsPayload, error) {
	client := c.client

	response, err := client.R().Get(fmt.Sprintf("%s/api/stats", c.endpoint))
	if err != nil {
		return nil, fmt.Errorf("could not GET /api/stats: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		payload := &StatsPayload{}
		err = json.NewDecoder(response.Body).Decode(payload)

		defer response.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("could not parse /api/status: %w", err)
		}

		return payload, nil
	}

	return nil, fmt.Errorf("could not load /api/status")
}
