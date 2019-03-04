package client_test

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/bonzofenix/prometheus2moogsoft/client"
)

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
		var annotations string
		var statusCode int
		var err error

		BeforeEach(func() {
			labels = `{ 
            "alertname":"SomeAlert"
      }`

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
                "status":"firing",
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
					Expect(event).ShouldNot(BeNil())
					Expect(event.Signature).Should(Equal("BoshJobUnhealthy::test::test-director::az1::cf::cc::0"))
					Expect(event.SourceId).Should(Equal("")) // timestamp + signature
					Expect(event.ExternalId).Should(Equal("test/test-director/cf"))
					Expect(event.Manager).Should(Equal("Prometheus"))
					Expect(event.Class).Should(Equal("PCF"))
					Expect(event.Type).Should(Equal(service))
					Expect(event.Severity).Should(Equal(4)) // 5 "critical", 4 "major", 3 minor 2 warning 1 indeterminate -0 "clear"
					Expect(event.Description).Should(Equal("BOSH Job test/test-director/cf/cc/0 is being reported unhealthy"))
					Expect(event.AgentTime).Should(Equal("1540313079")) //"startsAt":"2018-10-23T16:44:39.901211833Z",
					Expect(event.Agent).Should(Equal("dev"))
					Expect(event.AgentLocation).Should(Equal(""))          // Geographic location of prometheus
					Expect(event.AonMetricName).Should(Equal(""))          // disk percent
					Expect(event.AonMetricValue).Should(Equal(""))         // value of disk percent
					Expect(event.AonMonitoredEntityName).Should(Equal("")) // D:/ | linux mount path
					Expect(event.AonXMattersGroupName).Should(Equal("xmatter-group-id"))
					Expect(event.AonSNOWGroupName).Should(Equal(""))
					Expect(event.AonToolUrl).Should(Equal("https://prometheus.your-domain.com/graph?g0.expr=up+%3D%3D+0\u0026g0.tab=1"))
					Expect(event.AonIPAddress).Should(Equal("1.2.3.4")) // ip address to the machine where the disk event
					Expect(event.AonIPSubnet).Should(Equal(""))         // can be empty
					Expect(event.AonJSONVersion).Should(Equal("2"))
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
				Expect(event).ShouldNot(BeNil())

				Expect(event.Signature).Should(Equal("PrometheusScrapeError::concourse/web"))

				/*
				   {
				     "events": [{
				       "signature": "MonitorClassName::MonitoredEntityName::MonitoredMetricName::DeviceHostname",
				       "source_id": "UniqueEventID",
				       "external_id": "DeviceIdentifier",
				       "manager": "EventSourceName",
				       "source": "DeviceHostname",
				       "class": ""ExternalHosting-"HostingVendor",
				       "agent_location": "",
				       "type": "MonitorClassName",
				       "severity": SeverityID,
				       "description": "FullEventMessage",
				       "agent_time": "TimeOfEvent",
				       "agent": "",

				       "aonMetricName": "MonitoredMetricName",
				       "aonMetricValue": "MonitoredMetricValue",
				       "aonMonitoredEntityName": "MonitoredEntityName",
				       "aonXMattersGroupName": "",
				       "aonSNOWGroupName": "",
				       "aonToolURL": "",
				       "aonIPAddress": "",
				       "aonIPSubnet": "",
				       "aonJSONversion": "2"
				     }]
				   }
				*/
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
