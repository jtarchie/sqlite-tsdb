package main_test

import (
	"fmt"
	"os/exec"
	"strconv"
	"testing"

	"github.com/imroc/req/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/phayes/freeport"
)

func TestSqliteTsdb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SqliteTSDB Suite")
}

var (
	client *req.Client
	path   string
	port   int
)

var _ = BeforeSuite(func() {
	var err error
	path, err = gexec.Build("github.com/jtarchie/sqlite-tsdb")
	Expect(err).NotTo(HaveOccurred())
})

func cli(args ...string) *gexec.Session {
	command := exec.Command(path, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

func host(path string) string {
	return fmt.Sprintf("http://localhost:%d%s", port, path)
}

var _ = BeforeEach(func() {
	var err error

	port, err = freeport.GetFreePort()
	Expect(err).NotTo(HaveOccurred())

	client = req.C()
})

var _ = Describe("Starting the database", func() {
	It("can /ping", func() {
		session := cli("--port", strconv.Itoa(port))
		defer session.Kill()

		Eventually(session.Out).Should(gbytes.Say(`started`))
		response, err := client.R().Get(host("/ping"))
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(200))
	})
})
