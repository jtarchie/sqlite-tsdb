package main_test

import (
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
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

var path string

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

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
