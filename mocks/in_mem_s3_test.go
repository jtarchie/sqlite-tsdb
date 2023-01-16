package mocks_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jtarchie/sqlite-tsdb/mocks"
)

var _ = Describe("InMemS3", func() {
	It("can save and files", func() {
		server, err := mocks.NewS3Server("testing-1")
		Expect(err).NotTo(HaveOccurred())
		defer server.Close()

		count, err := server.HasObject(`.*.txt`)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeEquivalentTo(0))

		err = server.PutObject("test1.txt", strings.NewReader("testing"))
		Expect(err).NotTo(HaveOccurred())

		err = server.PutObject("test100.txt", strings.NewReader("testing"))
		Expect(err).NotTo(HaveOccurred())

		err = server.PutObject("test2.txt", strings.NewReader("testing"))
		Expect(err).NotTo(HaveOccurred())

		count, err = server.HasObject(`test\d.txt`)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeEquivalentTo(2))

		count, err = server.HasObject(`test\d{3}.txt`)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(BeEquivalentTo(1))
	})

	When("the server stops", func() {
		It("returns errors", func() {
			server, err := mocks.NewS3Server("testing-1")
			Expect(err).NotTo(HaveOccurred())

			server.Close()

			count, err := server.HasObject(`.*.txt`)
			Expect(err).To(HaveOccurred())
			Expect(count).To(BeEquivalentTo(0))

			err = server.PutObject("test1.txt", strings.NewReader("testing"))
			Expect(err).To(HaveOccurred())
		})
	})

	When("getting a count of files", func() {
		It("requires a valid regex", func() {
			server, err := mocks.NewS3Server("testing-1")
			Expect(err).NotTo(HaveOccurred())
			defer server.Close()

			count, err := server.HasObject(`*.txt`)
			Expect(err).To(HaveOccurred())
			Expect(count).To(BeEquivalentTo(0))
		})
	})
})
