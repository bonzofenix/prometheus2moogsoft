package client

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	yaml "gopkg.in/yaml.v2"
)

type FakeMoogsoftServer struct {
	engine         *gin.Engine
	server         *httptest.Server
	token          string
	ReceivedEvents []PrometheusEvent
}

func (fms *FakeMoogsoftServer) Start() {
	fms.engine = gin.New()
	fms.server = httptest.NewServer(fms.engine)

	rand.Seed(time.Now().UTC().UnixNano())

	fms.token = fmt.Sprintf("%d", rand.Intn(9999))
	fms.ReceivedEvents = []PrometheusEvent{}

	fms.engine.POST(fms.GetEventsEndpoint(), func(c *gin.Context) {
		if c.GetHeader("Authorization") == fmt.Sprintf("Basic %s", fms.token) {
			rawBody, _ := c.GetRawData()

			var prometheusEvent PrometheusEvent
			yaml.Unmarshal(rawBody, &prometheusEvent)

			fms.ReceivedEvents = append(fms.ReceivedEvents, prometheusEvent)

			c.String(http.StatusOK, "")
		} else {
			c.String(http.StatusForbidden, "Your credentials are invalid")
		}
	})
}

func (fms *FakeMoogsoftServer) URL() string {
	return fms.server.URL
}

func (fms *FakeMoogsoftServer) GetToken() string {
	return fms.token
}

func (fms *FakeMoogsoftServer) GetEventsEndpoint() string {
	return "/custom_moogsoft_events"
}
