package worker_test

import (
	"sync/atomic"
	"testing"

	"github.com/jtarchie/sqlite-tsdb/worker"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWorker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Worker Suite")
}

var _ = Describe("Worker", func() {
	It("can process work", func() {
		count := int32(0)
		w := worker.New[int](1, 1, func(index, value int) {
			defer GinkgoRecover()

			Expect(index).To(Equal(1))
			Expect(value).To(Equal(100))

			atomic.AddInt32(&count, 1)
		})
		defer w.Close()

		w.Enqueue(100)

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(1))
	})

	It("can process work across workers", func() {
		count := int32(0)
		w := worker.New[int](1, 100, func(index, value int) {
			defer GinkgoRecover()

			Expect(value).To(Equal(100))

			atomic.AddInt32(&count, 1)
		})
		defer w.Close()

		w.Enqueue(100)

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(1))
	})

	DescribeTable("handling a lot of work", func(queueSize, workerSize, elements int) {
		count := int32(0)
		w := worker.New[int](queueSize, workerSize, func(index, value int) {
			atomic.AddInt32(&count, 1)
		})
		defer w.Close()

		go func() {
			for i := 0; i < elements; i++ {
				w.Enqueue(i)
			}
		}()

		Eventually(func() int32 {
			return atomic.LoadInt32(&count)
		}).Should(BeEquivalentTo(elements))
	},
		Entry("1,1,100", 1, 1, 100),
		Entry("1,1,1000", 1, 1, 100),
		Entry("10,1,1000", 10, 1, 1000),
		Entry("1,10,1000", 10, 1, 1000),
		Entry("10,10,1000", 10, 10, 1000),
		Entry("10,10,1000", 10, 10, 100_000),
	)
})
