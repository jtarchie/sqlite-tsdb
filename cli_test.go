package main_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/jtarchie/sqlite-tsdb/mocks"
	"github.com/jtarchie/sqlite-tsdb/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gmeasure"
	"github.com/phayes/freeport"
)

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

		session = cli(path,
			"--port", strconv.Itoa(port),
			"--work-path", workPath,
			"--flush-size=100",
			"--buffer-size=100000",
			"--s3-access-key-id", "minio",
			"--s3-secret-access-key", "password",
			"--s3-bucket", bucketName,
			"--s3-endpoint", s3Server.URL(),
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

	It("runs successfully", Serial, Label("measurement"), func() {
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

		experiment := gmeasure.NewExperiment("Message Inserts")
		AddReportEntry(experiment.Name, experiment)

		// we sample a function repeatedly to get a statistically significant set of measurements
		experiment.Sample(func(idx int) {
			message := faker.Sentence()

			// measure how long it takes to RecomputePages() and store the duration in a "repagination" measurement
			experiment.MeasureDuration("send event", func() {
				_ = client.SendEvent(sdk.Event{
					Time: sdk.Time(time.Now().UnixNano()),
					Labels: sdk.Labels{
						"user_id":    "1234",
						"channel_id": "4567",
					},
					Value: sdk.Value(message),
				})
			})
		}, gmeasure.SamplingConfig{N: 1000, Duration: 10 * time.Second}) // we'll sample the function up to 20 times or up to a minute, whichever comes first.

		By("can /ping", func() {
			ok, err := client.Ping()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		By("increases the insert operations", func() {
			stats, err := client.Stats()
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Count.Insert).To(BeEquivalentTo(1001))
		})

		By("exports on to s3", func() {
			Eventually(func() int {
				count, err := s3Server.HasObject(`\d+.db`)
				Expect(err).NotTo(HaveOccurred())

				return count
			}).Should(BeEquivalentTo(10))
		})

		By("has database files", func() {
			matches, err := filepath.Glob(filepath.Join(workPath, "*.db"))
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(HaveLen(11))
		})
	})
})
