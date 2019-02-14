package client_test

import (
	"net/http"

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
				code, err := client.SendEvents("{}", "wrong-token")

				Expect(err).Should(BeNil())
				Expect(code).Should(Equal(http.StatusForbidden))
			})
		})

		Context("when using the right credentials", func() {
			It("Should return connect to moogsoft server", func() {
				code, err := client.SendEvents("{}", moogsoftServer.GetToken())

				Expect(err).Should(BeNil())
				Expect(code).Should(Equal(http.StatusOK))

			})

			It("Should send event to moogsoft server", func() {
				prometheusEvent := `{
            "receiver":"default",
            "status":"firing",
            "groupLabels":{},
            "commonLabels": { "severity":"warning" },
            "commonAnnotations":{},
            "externalURL":"https://alertmanager.your-domain.com",
            "version":"4",
            "groupKey":"{}:{}"
            "alerts": [
              {
                "status":"firing",
                "labels": {
                  "alertname":"PrometheusScrapeError",
                  "bosh_deployment":"concourse",
                  "instance":"1.2.3.4:9391",
                  "job":"concourse",
                  "service":"prometheus",
                  "severity":"warning"
                },
                "annotations": {
                  "description":"some alert description",
                  "summary":" some alert summary"
                },
                "startsAt":"2018-10-23T16:44:39.901211833Z",
                "endsAt":"2018-11-07T11:45:39.901211833Z",
                "generatorURL":"https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"
              }
            ]
          }`

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(0))

				code, err := client.SendEvents(prometheusEvent, moogsoftServer.GetToken())

				Expect(err).Should(BeNil())
				Expect(code).Should(Equal(http.StatusOK))

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(1))
				event := moogsoftServer.ReceivedEvents[0]
				Expect(event).ShouldNot(BeNil())

				/*
				   {"events":
				     [
				       {
				         "signature": "my_test_box:application:Network",
				         "source_id": "1.2.3.4",
				         "external_id": "id-1234",
				         "manager": "my_manager",
				         "source": "my_test_box",
				         "class": "application",
				         "agent_location": "my_agent_location",
				         "type": "Network",
				         "severity": 3,
				         "description": "high network utilization in application A",
				         "agent_time": "1411134582"
				       }
				     ]
				   }
				*/
			})
		})
	})
})
