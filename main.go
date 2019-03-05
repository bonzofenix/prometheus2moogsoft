package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bonzofenix/prometheus2moogsoft/client"
	"github.com/gin-gonic/gin"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Port string `short:"p" long:"prefix" description:"Port where app will be running." optional:"true"`
}

var opts Options

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	log.SetOutput(os.Stdout)
	gin.SetMode(gin.ReleaseMode)

	p2mServer := gin.Default()

	if opts.Port == "" {
		opts.Port = os.Getenv("PORT")
	}

	client := client.Client{
		Env:               os.Getenv("MOOGSOFT_ENV"),
		URL:               os.Getenv("MOOGSOFT_URL"),
		EventsEndpoint:    os.Getenv("MOOGSOFT_ENDPOINT"),
		XMattersGroupName: os.Getenv("XMATTERS_GROUP_NAME"),
	}

	token := os.Getenv("MOOGSOFT_TOKEN")
	redactedToken := ""
	if token != "" {
		redactedToken = "[REDACTED]"
	}

	p2mServer.GET("/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"moogsoft_url":             client.URL,
			"moogsoft_events_endpoint": client.EventsEndpoint,
			"moogsoft_token":           redactedToken,
		})
	})

	p2mServer.POST("/prometheus_webhook_event", func(c *gin.Context) {
		body, _ := c.GetRawData()

		responseCode, err := client.SendEvents(string(body), token)

		if err != nil {
			c.String(responseCode, err.Error())

			fmt.Println(err.Error())
		} else {
			c.String(responseCode, "events sent")
		}
	})

	p2mServer.Run(fmt.Sprintf(":%s", opts.Port))
}
