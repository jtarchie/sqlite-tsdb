package mocks_test

import (
	"strings"
	"testing"

	"github.com/jtarchie/sqlite-tsdb/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMocks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mocks Suite")
}

var _ = Describe("Mocks", func() {
	When("creating an s3 server", func() {
		It("can save files", func() {
			server, err := mocks.NewS3Server("testing-1")
			Expect(err).NotTo(HaveOccurred())

			err = server.PutObject("test1.txt", strings.NewReader("testing"))
			Expect(err).NotTo(HaveOccurred())

			err = server.PutObject("test100.txt", strings.NewReader("testing"))
			Expect(err).NotTo(HaveOccurred())

			err = server.PutObject("test2.txt", strings.NewReader("testing"))
			Expect(err).NotTo(HaveOccurred())

			count, err := server.HasObject(`test\d.txt`)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeEquivalentTo(2))
		})
	})
})
