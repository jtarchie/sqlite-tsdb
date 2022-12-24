package main_test

import (
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/jtarchie/sqlite-tsdb/sdk"
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
	client *sdk.Client
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

var _ = BeforeEach(func() {
	var err error

	port, err = freeport.GetFreePort()
	Expect(err).NotTo(HaveOccurred())

	client, err = sdk.New(fmt.Sprintf("http://localhost:%d", port))
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Starting the database", func() {
	It("can /ping", func() {
		session := cli("--port", strconv.Itoa(port))
		defer session.Kill()

		ok, err := client.Ping()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	When("inserting an event", func() {
		It("increases the insert operations", func() {
			session := cli("--port", strconv.Itoa(port))
			defer session.Kill()

			err := client.SendEvent(sdk.Event{
				Time: sdk.Time(time.Now().UnixNano()),
				Labels: sdk.Labels{
					"hello": "world",
				},
				Value: "This is a test value",
			})
			Expect(err).NotTo(HaveOccurred())

			stats, err := client.Stats()
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Count.Insert).To(BeEquivalentTo(1))
		})
	})
})
