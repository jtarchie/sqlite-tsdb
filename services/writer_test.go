package services_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jtarchie/sqlite-tsdb/services"
)

var _ = Describe("Writer", func() {
	It("creates a db file", func() {
		dbFile, err := os.CreateTemp("", "")
		Expect(err).NotTo(HaveOccurred())
		
		err = dbFile.Close()
		Expect(err).NotTo(HaveOccurred())
		
		writer, err := services.NewWriter(dbFile.Name())
		Expect(err).NotTo(HaveOccurred())

		err = writer.Close()
		Expect(err).NotTo(HaveOccurred())

		info, err := os.Stat(dbFile.Name())
		Expect(err).NotTo(HaveOccurred())

		Expect(info.Size()).To(BeNumerically(">", 0))
	})
})
