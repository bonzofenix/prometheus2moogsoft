package integration_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/bonzofenix/prometheus2moogsoft/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var prometheusToMoogsoftCmd *exec.Cmd

var _ = Describe("Prometheus2Moogsoft", func() {
	var moogsoftServer client.FakeMoogsoftServer
	var session *gexec.Session
	var err error
	var prometheusPayload []byte

	BeforeEach(func() {
		moogsoftServer.Start()

		os.Setenv("MOOGSOFT_URL", moogsoftServer.URL())
		os.Setenv("MOOGSOFT_ENDPOINT", moogsoftServer.GetEventsEndpoint())
		os.Setenv("MOOGSOFT_TOKEN", moogsoftServer.GetToken())

		prometheusToMoogsoftCmd = exec.Command(prometheusToMoogsoftPath, "-p 3000")
	})

	AfterEach(func() {
		moogsoftServer.Stop()
		session.Kill()
	})

	JustBeforeEach(func() {
		session, err = gexec.Start(prometheusToMoogsoftCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("When receiving supported alert", func() {
		JustBeforeEach(func() {
			prometheusPayload, err = ioutil.ReadFile(AssetPathFor("supported_alerts.json"))
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(serverIsRunning, "2s").Should(BeTrue())
		})

		It("Should send alert to moogsoft", func() {
			POST("http://localhost:3000/prometheus_webhook_event", prometheusPayload)
			Eventually(moogsoftServer.ReceivedEvents, "2s").Should(HaveLen(2))
		})
	})

	Context("When receiving unsupported alert", func() {
		JustBeforeEach(func() {
			prometheusPayload, err = ioutil.ReadFile(AssetPathFor("unsupported_alerts.json"))
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(serverIsRunning, "2s").Should(BeTrue())
		})

		It("Should still send the alert the alert to moogsoft", func() {
			POST("http://localhost:3000/prometheus_webhook_event", prometheusPayload)
			Eventually(moogsoftServer.ReceivedEvents, "2s").Should(HaveLen(1))
		})

		It("Should log the alert as unsopported", func() {
			session.Kill()
			Expect(string(session.Wait().Out.Contents()[:])).Should(ContainSubstring("Unsopported parsed service: some-alert-service"))
		})
	})

	//Context("When moogsoft returns an error", func() {
	//	BeforeEach(func() {
	//		prometheusToMoogsoftCmd = exec.Command(prometheusToMoogsoftPath, "-p 3000")
	//	})

	//	JustBeforeEach(func() {
	//		prometheusPayload, err = ioutil.ReadFile(AssetPathFor("supported_alerts.yml"))
	//		Expect(err).ShouldNot(HaveOccurred())
	//	})

	//	XIt("Should error", func() {
	//		Eventually(session, 60*time.Second).Should(gexec.Exit(1))
	//		Expect(string(session.Wait().Out.Contents()[:])).Should(ContainSubstring("Unable to write secrets to vault"))
	//	})
	//})
})

func POST(uri string, rawData []byte) string {
	req, err := http.NewRequest("POST", uri, bytes.NewReader(rawData))
	Expect(err).ShouldNot(HaveOccurred())

	req.Close = true

	res, err := http.DefaultClient.Do(req)
	Expect(err).ShouldNot(HaveOccurred())
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	Expect(err).ShouldNot(HaveOccurred())

	return string(body)
}

func serverIsRunning() bool {
	_, err := net.Dial("tcp", "localhost:3000")
	return err == nil
}

func AssetPathFor(filename string) string {
	return fmt.Sprintf("%s/src/github.com/bonzofenix/prometheus2moogsoft/integration/assets/%s", os.Getenv("GOPATH"), filename)
}
