package sdk

import (
	"fmt"
	"net/http"
)

func (c *Client) Ping() (bool, error) {
	client := c.client

	response, err := client.R().
		Get(fmt.Sprintf("%s/ping", c.endpoint))
	if err != nil {
		return false, fmt.Errorf("could not GET /ping: %w", err)
	}

	if response.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, nil
}