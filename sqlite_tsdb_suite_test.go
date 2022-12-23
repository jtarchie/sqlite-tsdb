package main_test

import (
	"encoding/json"
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

	Eventually(session.Out).Should(gbytes.Say(`started`))

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

		response, err := client.R().Get(host("/ping"))
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(200))
	})

	When("inserting a value", func() {
		type statsPayload struct {
				Count struct {
					Insert int64
				}
		}

		It("increases the insert operations", func() {
			session := cli("--port", strconv.Itoa(port))
			defer session.Kill()

			response, err := client.R().Put(host("/api/events"))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(201))

			response, err = client.R().Get(host("/api/stats"))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(200))

			payload := statsPayload{}
			err = json.NewDecoder(response.Body).Decode(&payload)
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()

			Expect(payload.Count.Insert).To(BeEquivalentTo(1))
		})
	})
})
