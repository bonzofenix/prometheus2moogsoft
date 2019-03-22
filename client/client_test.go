package client_test

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/bonzofenix/prometheus2moogsoft/client"
)

func assertEventCommonFields(e MoogsoftEvent) {
	ExpectWithOffset(1, e).ShouldNot(BeNil())
	ExpectWithOffset(1, e.AgentLocation).Should(Equal(""))          // Geographic location of prometheus
	ExpectWithOffset(1, e.AonMetricName).Should(Equal(""))          // disk percent
	ExpectWithOffset(1, e.AonMetricValue).Should(Equal(""))         // value of disk percent
	ExpectWithOffset(1, e.AonMonitoredEntityName).Should(Equal("")) // D:/ | linux mount path
	ExpectWithOffset(1, e.Agent).Should(Equal("dev"))
	ExpectWithOffset(1, e.AonXMattersGroupName).Should(Equal("xmatter-group-id"))
	ExpectWithOffset(1, e.AonSNOWGroupName).Should(Equal(""))
	ExpectWithOffset(1, e.Manager).Should(Equal("Prometheus"))
	ExpectWithOffset(1, e.Class).Should(Equal("PCF"))
	ExpectWithOffset(1, e.AonIPSubnet).Should(Equal("")) // can be empty
	ExpectWithOffset(1, e.SourceId).Should(Equal(""))    // timestamp + signature
	ExpectWithOffset(1, e.AonToolUrl).Should(Equal("https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"))
	ExpectWithOffset(1, e.AonJSONVersion).Should(Equal("2"))
	ExpectWithOffset(1, e.AgentTime).Should(Equal("1540313079")) //"startsAt":"2018-10-23T16:44:39.901211833Z",
}

var _ = Describe("Client", func() {

	var prometheusEvent string
	var client Client
	var moogsoftServer FakeMoogsoftServer

	gin.SetMode(gin.ReleaseMode)

	BeforeEach(func() {
		moogsoftServer.Start()

		client = Client{
			URL:               moogsoftServer.URL(),
			EventsEndpoint:    moogsoftServer.GetEventsEndpoint(),
			Env:               "dev",
			XMattersGroupName: "xmatter-group-id",
		}
	})

	Context("#SendEvents", func() {
		var token string
		var labels string
		var status string
		var annotations string
		var statusCode int
		var err error

		BeforeEach(func() {
			labels = `{ 
            "alertname": "SomeAlert",
            "service": "prometheus",
            "severity":"warning"
      }`

			status = "firing"

			annotations = `{
        "description":"some alert description",
        "summary":" some alert summary"
      }`

			token = moogsoftServer.GetToken()
		})

		JustBeforeEach(func() {
			prometheusEvent = `{
            "receiver":"default",
            "status":"firing",
            "groupLabels":{},
            "commonLabels": { "severity":"warning" },
            "commonAnnotations":{},
            "externalURL":"https://alertmanager.your-domain.com",
            "version":"4",
            "groupKey":"{}:{}",
            "alerts": [
              {
                "status": "` + status + `",
                "labels": ` + labels + `,
                "annotations": ` + annotations + `,
                "startsAt":"2018-10-23T16:44:39.901211833Z", 
                "endsAt":"2018-11-07T11:45:39.901211833Z",
                "generatorURL":"https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"
              }
            ]
          }`
		})

		Context("when using the wrong credentials", func() {
			BeforeEach(func() { token = "wrong-token" })

			It("Should not return an error with 401 anauthorized", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusForbidden))
			})
		})

		Context("when using the right credentials", func() {
			BeforeEach(func() { token = moogsoftServer.GetToken() })

			It("Should return connect to moogsoft server", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusOK))
			})
		})

		Context("when alert gets resolved", func() {
			BeforeEach(func() { status = "resolved" })

			It("Should send severity 0", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusOK))

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(1))
				event := moogsoftServer.ReceivedEvents[0]
				Expect(event).ShouldNot(BeNil())
				Expect(event.Severity).Should(Equal(CLEAR)) // 5 "critical", 4 "major", 3 minor 2 warning 1 indeterminate -0 "clear"
			})

		})
		Context("when receiving cf alert from the cf_exporter", func() {
			BeforeEach(func() {
				labels = `{
            "alertname":"CFRoutesNotBeingRegistered",
            "bosh_deployment":"cf-123",
            "environment":"dev",
            "service": "cf",
            "severity":"warning"
          }`

				annotations = ` {
					  "summary": "number of routes too low in dev/cf-123",
					  "description": "There has been only 0 routes in the routing table at CF dev/cf-123 during the last 5m"
					}`
			})

			It("Should parse warnings and send event", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusOK))

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(1))
				event := moogsoftServer.ReceivedEvents[0]
				assertEventCommonFields(event)

				Expect(event.Signature).Should(Equal("CFRoutesNotBeingRegistered::dev::cf-123"))
				Expect(event.Type).Should(Equal("cf"))

				Expect(event.ExternalId).Should(Equal(event.Signature))
				Expect(event.Severity).Should(Equal(MAJOR)) // 5 "critical", 4 "major", 3 minor 2 warning 1 indeterminate -0 "clear"
				Expect(event.Description).Should(Equal("There has been only 0 routes in the routing table at CF dev/cf-123 during the last 5m"))
				Expect(event.AonIPAddress).Should(Equal("")) // ip address to the machine where the disk event
			})
		})

		for _, service := range [...]string{"bosh-deployment", "bosh-job", "bosh-job-process"} {
			Context(fmt.Sprintf("when receiving %s alert from the bosh_exporter", service), func() {
				BeforeEach(func() {
					labels = `{
            "alertname":"BoshJobUnhealthy",
            "environment": "test",
            "bosh_name": "test-director",
            "bosh_uuid": "some-uuid",
            "bosh_deployment": "cf",
            "bosh_job_name": "cc",
            "bosh_job_id": "some-job-id-uuid",
            "bosh_job_index": "0",
            "bosh_job_az": "az1",
            "bosh_job_ip": "1.2.3.4",
            "severity":"warning",
            "service": "` + service + `"
          }`

					annotations = ` {
					  "summary": "BOSH Job test/test-director/cf/cc/0 is unhealthy",
					  "description": "BOSH Job test/test-director/cf/cc/0 is being reported unhealthy"
					}`
				})

				It("Should parse warnings and send event", func() {
					statusCode, err = client.SendEvents(prometheusEvent, token)
					Expect(err).Should(BeNil())
					Expect(statusCode).Should(Equal(http.StatusOK))

					Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(1))
					event := moogsoftServer.ReceivedEvents[0]
					assertEventCommonFields(event)

					Expect(event.Signature).Should(Equal("BoshJobUnhealthy::test::test-director::az1::cf::cc::0"))
					Expect(event.ExternalId).Should(Equal(event.Signature))

					Expect(event.Type).Should(Equal(service))
					Expect(event.Severity).Should(Equal(MAJOR)) // 5 "critical", 4 "major", 3 minor 2 warning 1 indeterminate -0 "clear"
					Expect(event.Description).Should(Equal("BOSH Job test/test-director/cf/cc/0 is being reported unhealthy"))
					Expect(event.AgentTime).Should(Equal("1540313079")) //"startsAt":"2018-10-23T16:44:39.901211833Z",
					Expect(event.AonIPAddress).Should(Equal("1.2.3.4")) // ip address to the machine where the disk event
				})
			})
		}

		Context("when receiving prometheus alerts", func() {
			BeforeEach(func() {
				labels = `{
            "alertname":"PrometheusScrapeError",
            "bosh_deployment":"concourse",
            "instance":"1.2.3.4:9391",
            "job":"web",
            "service":"prometheus",
            "severity":"warning"
          }`
			})

			It("Should parse and send event", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusOK))

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(1))
				event := moogsoftServer.ReceivedEvents[0]

				assertEventCommonFields(event)

				Expect(event.Signature).Should(Equal("PrometheusScrapeError::concourse::web"))
				Expect(event.Type).Should(Equal("prometheus"))
			})
		})

		Context("when reciving multiple alerts in one call", func() {
			JustBeforeEach(func() {
				prometheusEvent = `{
            "receiver":"default",
            "status":"firing",
            "groupLabels":{},
            "commonLabels": { "severity":"warning" },
            "commonAnnotations":{},
            "externalURL":"https://alertmanager.your-domain.com",
            "version":"4",
            "groupKey":"{}:{}",
            "alerts": [
              {
                "status":"firing",
                "labels": ` + labels + `,
                "annotations": ` + annotations + `,
                "startsAt":"2018-10-23T16:44:39.901211833Z", 
                "endsAt":"2018-11-07T11:45:39.901211833Z",
                "generatorURL":"https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"
              },
              {
                "status":"firing",
                "labels": ` + labels + `,
                "annotations": ` + annotations + `,
                "startsAt":"2018-10-24T16:44:39.901211833Z", 
                "endsAt":"2018-11-07T11:45:39.901211833Z",
                "generatorURL":"https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"
              }
            ]
          }`
			})

			It("Should send multiple events in the same call to moogsoft", func() {
				statusCode, err = client.SendEvents(prometheusEvent, token)
				Expect(err).Should(BeNil())
				Expect(statusCode).Should(Equal(http.StatusOK))

				Expect(moogsoftServer.ReceivedEvents).Should(HaveLen(2))

				firstEvent := moogsoftServer.ReceivedEvents[0]
				secondEvent := moogsoftServer.ReceivedEvents[1]

				// Check that both event time is different
				Expect(firstEvent.AgentTime).ShouldNot(Equal(secondEvent.AgentTime))
			})
		})
	})
})
