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

func cli(args ...string) *gexec.Session {
	command := exec.Command(args[0], args[1:]...) //nolint: gosec
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	Eventually(session.Out).Should(gbytes.Say(`started`))

	return session
}

var _ = Describe("Running the CLI", func() {
	var (
		bucketName string
		client     *sdk.Client
		port       int
		s3Server   *mocks.S3Server
		session    *gexec.Session
		workPath   string
	)

	BeforeEach(func() {
		path, err := gexec.Build(
			"github.com/jtarchie/sqlite-tsdb",
			"--tags", "fts5 json1",
		)
		Expect(err).NotTo(HaveOccurred())

		bucketName = fmt.Sprintf("bucket-name-%d", GinkgoParallelProcess())

		port, err = freeport.GetFreePort()
		Expect(err).NotTo(HaveOccurred())

		client, err = sdk.New(fmt.Sprintf("http://localhost:%d", port))
		Expect(err).NotTo(HaveOccurred())

		s3Server, err = mocks.NewS3Server(bucketName)
		Expect(err).NotTo(HaveOccurred())

		workPath, err = os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())

		session = cli(path,
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
	})

	AfterEach(func() {
		session.Kill()

		s3Server.Close()

		err := os.RemoveAll(workPath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("runs successfully", func() {
		By("sending a single event", func() {
			err := client.SendEvent(sdk.Event{
				Time: sdk.Time(time.Now().UnixNano()),
				Labels: sdk.Labels{
					"hello": "world",
				},
				Value: "This is a test value",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		By("can /ping", func() {
			ok, err := client.Ping()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		By("increases the insert operations", func() {
			stats, err := client.Stats()
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Count.Insert).To(BeEquivalentTo(1))
		})

		By("has database files", func() {
			matches, err := filepath.Glob(filepath.Join(workPath, "*.db"))
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(HaveLen(2))
		})

		By("exports on to s3", func() {
			Eventually(func() int {
				count, err := s3Server.HasObject(`\d+.db`)
				Expect(err).NotTo(HaveOccurred())

				return count
			}).Should(BeEquivalentTo(1))
		})
	})
})
