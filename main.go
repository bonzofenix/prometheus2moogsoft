package main

import (
	"io/ioutil"

	. "github.com/bonzofenix/prometheus2moogsoft/client"
	"github.com/gin-gonic/gin"
)

type Config struct {
	Moogsoft struct {
		Endpoint string `yaml:"endpoint"`
		Token    string `yaml:"token"`
	} `yaml:"moogsoft"`
}

func main() {
	moaccServer := gin.Defaults()

	var config Config

	_, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		config = parseYAML("../config.yaml")
	} else {
		config = parseYAML("config.yaml")
	}

	client := Client{
		Endpoint: config.Moogsoft.Endpoint,
		Token:    config.Moogsoft.Token,
	}

	moaccServer.POST("/prometheus_webhook_event", func(c *gin.Context) {
		Moogsoft.post("")
		c.String(200, "events sent")
	})

	moaccServer.Run(":3000")
}
