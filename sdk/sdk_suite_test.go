package sdk_test

import (
	"testing"

	"github.com/jtarchie/sqlite-tsdb/sdk"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestSDK(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SDK Suite")
}

var _ = Describe("Client", func() {
	var (
		server *ghttp.Server
		client *sdk.Client
	)

	BeforeEach(func() {
		var err error

		server = ghttp.NewServer()
		client, err = sdk.New(server.URL())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	When("pinging", func() {
		It("returns false on non-200", func() {
			for _, statusCode := range []int{400, 500} {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/ping"),
						ghttp.RespondWith(statusCode, ``),
					),
				)

				ok, err := client.Ping()
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeFalse())
			}
		})

		It("errors on network issues", func() {
			server.Close()

			ok, err := client.Ping()
			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("returns true on 200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/ping"),
					ghttp.RespondWith(200, ``),
				),
			)

			ok, err := client.Ping()
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})

	When("retrieving stats", func() {
		It("returns false on non-200", func() {
			for _, statusCode := range []int{400, 500} {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/stats"),
						ghttp.RespondWith(statusCode, ``),
					),
				)

				stats, err := client.Stats()
				Expect(err).To(HaveOccurred())
				Expect(stats).To(BeNil())
			}
		})

		It("errors on network issues", func() {
			server.Close()

			stats, err := client.Stats()
			Expect(err).To(HaveOccurred())
			Expect(stats).To(BeNil())
		})

		It("returns stats on 200", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/stats"),
					ghttp.RespondWith(200, `{
						"count": {
							"insert": 1
						}	
					}`),
				),
			)

			stats, err := client.Stats()
			Expect(err).NotTo(HaveOccurred())
			Expect(stats.Count.Insert).To(BeEquivalentTo(1))
		})
	})
})
