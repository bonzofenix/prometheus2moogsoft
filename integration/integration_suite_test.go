package integration_test

import (
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/gomega/gexec"
)

var prometheusToMoogsoftPath string

var _ = BeforeSuite(func() {
	var err error

	prometheusToMoogsoftPath, err = gexec.Build("github.com/bonzofenix/prometheus2moogsoft")
	Expect(err).ShouldNot(HaveOccurred())

	gin.SetMode(gin.ReleaseMode)
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
