package client_test

import (
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/bonzofenix/prometheus2moogsoft/client"
)

var _ = Describe("Client", func() {
	var client Client
	var moogsoftServer FakeMoogsoftServer

	gin.SetMode(gin.ReleaseMode)

	BeforeEach(func() {
		moogsoftServer.Start()

		client = Client{
			URL:            moogsoftServer.URL(),
			EventsEndpoint: moogsoftServer.GetEventsEndpoint(),
		}
	})

	Context("#SendEvents", func() {
		Context("when using the wrong credentials", func() {
			It("Should not return an error with 401 anauthorized", func() {
				code, err := client.SendEvents("{}", "Wrong token")

				Expect(code).Should(Equal(401))
				Expect(err).Should(BeNil())
			})
		})

		It("Should post events to moogsoft in the right format", func() {
		})
	})
})
