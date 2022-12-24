package sdk

import (
	"fmt"
	"net/http"
)

type Labels map[string]string
type Time uint64
type Value string

type Event struct {
	Labels Labels
	Time   Time
	Value  Value
}

func (c *Client) SendEvent(event Event) error {
	client := c.client

	response, err := client.R().
		SetBodyJsonMarshal(event).
		Put(fmt.Sprintf("%s/api/events", c.endpoint))
	if err != nil {
		return fmt.Errorf("could not PUT /api/events: %w", err)
	}

	if response.StatusCode == http.StatusCreated {
		return nil
	}

	return fmt.Errorf("the PUT to /api/events failed")
}