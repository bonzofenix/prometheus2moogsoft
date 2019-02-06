package client

import (
	"fmt"
	"net/http"
)

type Client struct {
	URL            string
	EventsEndpoint string
}

func (c *Client) SendEvents(payload string, token string) (int, error) {
	res, err := http.Post(fmt.Sprintf(c.EventsEndpoint), "application/json", []byte(payload))

	return 200, err
}
