package worker

import (
	"context"
	"sync"
)

type Job interface{}

type ProcessFunc func(ctx context.Context, job Job) error

type WorkerPool struct {
	numWorkers int
	jobs       chan Job
	processor  ProcessFunc
	wg         sync.WaitGroup
}

func NewWorkerPool(numWorkers int, bufferSize int, processor ProcessFunc) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, bufferSize),
		processor:  processor,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 1; i <= wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}
			wp.processor(ctx, job)
		}
	}
}

func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}
