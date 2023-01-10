package server_test

import (
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jtarchie/sqlite-tsdb/server"
	"github.com/labstack/echo/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("Server", func() {
	When("using the logging middleware", func() {
		var (
			buffer     *gbytes.Buffer
			serverHTTP *ghttp.Server
		)

		BeforeEach(func() {
			buffer = gbytes.NewBuffer()
			logger := zap.New(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionConfig().EncoderConfig),
					zapcore.AddSync(buffer),
					zapcore.InfoLevel,
				),
			)

			router := echo.New()
			router.Use(server.ZapLogger(logger))
			router.GET("/", func(c echo.Context) error {
				return c.String(200, "abcd")
			})

			serverHTTP = ghttp.NewServer()
			serverHTTP.AppendHandlers(router.ServeHTTP)
		})

		AfterEach(func() {
			serverHTTP.Close()
		})

		When("no request ID is provided", func() {
			It("generates a request ID", func() {
				_, err := req.C().R().
					Get(serverHTTP.URL())
				Expect(err).NotTo(HaveOccurred())

				Eventually(buffer).Should(gbytes.Say(`"level":"info"`))
				Eventually(buffer).Should(gbytes.Say(`"ts":\d+\.\d+`))
				Eventually(buffer).Should(gbytes.Say(`"msg":"http_request"`))
				Eventually(buffer).Should(gbytes.Say(`"bytes_in":"0"`))
				Eventually(buffer).Should(gbytes.Say(`"bytes_out":"4"`))
				Eventually(buffer).Should(gbytes.Say(`"status":200`))
				Eventually(buffer).Should(gbytes.Say(`"id":"[0-9a-fA-F]{8}-[[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"`))
				Eventually(buffer).Should(gbytes.Say(`"latency_human":"\d+`))
				Eventually(buffer).Should(gbytes.Say(`"method":"GET"`))
				Eventually(buffer).Should(gbytes.Say(`"remote_ip":"\d+\.\d+\.\d+\.\d+"`))
			})
		})

		When("a request ID is provided", func() {
			It("uses that one", func() {
				_, err := req.C().R().
					SetHeader("X-Request-Id", "unique-id").
					Get(serverHTTP.URL())
				Expect(err).NotTo(HaveOccurred())

				Eventually(buffer).Should(gbytes.Say(`"level":"info"`))
				Eventually(buffer).Should(gbytes.Say(`"msg":"http_request"`))
				Eventually(buffer).Should(gbytes.Say(`"status":200`))
				Eventually(buffer).Should(gbytes.Say(`"id":"unique-id"`))
			})
		})
	})
})
