package client

import (
	"fmt"
	"net/http"
	"strings"
)

// Moogsoft client
type Client struct {
	URL            string
	EventsEndpoint string
}

type PrometheusEvent struct{}

func (c *Client) SendEvents(payload string, token string) (int, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.URL, c.EventsEndpoint), strings.NewReader(payload))
	if err != nil {
		return 500, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", token))
	res, err := http.DefaultClient.Do(req)

	return res.StatusCode, err
}
