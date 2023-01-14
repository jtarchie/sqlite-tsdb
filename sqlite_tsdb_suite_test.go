package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jtarchie/sqlite-tsdb/mocks"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/phayes/freeport"
)

func TestSqliteTSDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SqliteTSDB Suite")
}

var (
	bucketName string
	client     *sdk.Client
	path       string
	workPath   string
	port       int
	s3Server   *mocks.S3Server
)

var _ = SynchronizedBeforeSuite(func() []byte {
	path, err := gexec.Build(
		"github.com/jtarchie/sqlite-tsdb",
		"--tags", "fts5 json1",
	)
	Expect(err).NotTo(HaveOccurred())

	return []byte(path)
}, func(data []byte) {
	path = string(data)
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

	bucketName = fmt.Sprintf("bucket-name-%d", GinkgoParallelProcess())

	port, err = freeport.GetFreePort()
	Expect(err).NotTo(HaveOccurred())

	client, err = sdk.New(fmt.Sprintf("http://localhost:%d", port))
	Expect(err).NotTo(HaveOccurred())

	s3Server, err = mocks.NewS3Server(bucketName)
	Expect(err).NotTo(HaveOccurred())

	workPath, err = os.MkdirTemp("", "")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterEach(func() {
	s3Server.Close()

	err := os.RemoveAll(workPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Starting the database", func() {
	It("can /ping", func() {
		session := cli(
			"--port", strconv.Itoa(port),
			"--work-path", workPath,
		)
		defer session.Kill()

		ok, err := client.Ping()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

	When("inserting an event", func() {
		var session *gexec.Session

		BeforeEach(func() {
			session = cli(
				"--port", strconv.Itoa(port),
				"--work-path", workPath,
				"--flush-size=1",
				"--s3-access-key-id", "access-key",
				"--s3-secret-access-key", "secret-key",
				"--s3-bucket", bucketName,
				"--s3-endpoint", s3Server.URL,
				"--s3-region", "fake-region",
				"--s3-skip-verify",
				"--s3-force-path-style",
			)

			err := client.SendEvent(sdk.Event{
				Time: sdk.Time(time.Now().UnixNano()),
				Labels: sdk.Labels{
					"hello": "world",
				},
				Value: "This is a test value",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			session.Kill()
		})

		It("increases the insert operations", func() {
			stats, err := client.Stats()
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Count.Insert).To(BeEquivalentTo(1))
		})

		It("has database files", func() {
			matches, err := filepath.Glob(filepath.Join(workPath, "*.db"))
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(HaveLen(1))
		})

		PIt("exports on to s3", func() {
			count, err := s3Server.HasObject(iso8601Regex)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeEquivalentTo(1))
		})
	})
})

const iso8601Regex = `^(?:[1-9]\d{3}-(?:(?:0[1-9]|1[0-2])-(?:0[1-9]|1\d|2[0-8])|(?:0[13-9]|1[0-2])-(?:29|30)|(?:0[13578]|1[02])-31)|(?:[1-9]\d(?:0[48]|[2468][048]|[13579][26])|(?:[2468][048]|[13579][26])00)-02-29)T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:Z|[+-][01]\d:[0-5]\d)$`
