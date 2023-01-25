package services_test

import (
	"os"

	"github.com/jtarchie/sqlite-tsdb/sdk"
	"github.com/jtarchie/sqlite-tsdb/services"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Writer", func() {
	It("creates a db file", func() {
		dbFile, err := os.CreateTemp("", "")
		Expect(err).NotTo(HaveOccurred())

		err = dbFile.Close()
		Expect(err).NotTo(HaveOccurred())

		writer, err := services.NewWriter(dbFile.Name())
		Expect(err).NotTo(HaveOccurred())

		By("writing a payload", func() {
			err = writer.Insert(&sdk.Event{})
			Expect(err).NotTo(HaveOccurred())
		})

		err = writer.Close()
		Expect(err).NotTo(HaveOccurred())

		Expect(writer.Filename()).To(Equal(dbFile.Name()))

		info, err := os.Stat(writer.Filename())
		Expect(err).NotTo(HaveOccurred())

		Expect(info.Size()).To(BeNumerically(">", 0))
	})
})
