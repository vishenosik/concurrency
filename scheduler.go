package concurrency

import (
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Queuer interface {
	Next() (time.Time, bool)
	Execute()
	AddChan() <-chan struct{}
}

type SchedulerOption func(*Scheduler)

type Scheduler struct {
	queue       Queuer
	idleTimeout time.Duration
	stopCH      chan struct{}
	wg          sync.WaitGroup
}

func defaultScheduler() *Scheduler {
	return &Scheduler{
		idleTimeout: time.Hour,
		stopCH:      make(chan struct{}),
	}
}

func NewScheduler(queuer Queuer, opts ...SchedulerOption) (*Scheduler, error) {
	scheduler := defaultScheduler()
	if queuer == nil {
		return nil, errors.New("Queue must not be nil")
	}
	scheduler.queue = queuer
	for _, opt := range opts {
		opt(scheduler)
	}
	return scheduler, nil
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	close(s.stopCH)
	s.wg.Wait()
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {

		next, ok := s.queue.Next()
		if ok {
			timer.Reset(time.Until(next))
		} else {
			timer.Reset(s.idleTimeout)
		}

		select {
		case <-s.stopCH:
			return
		case <-s.queue.AddChan():
		case <-timer.C:
			s.runJob()
		}
	}
}

func (s *Scheduler) runJob() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job panicked: %v\n", r)
		}
	}()
	s.queue.Execute()
}

type heapScheduler struct {
	scheduler *Scheduler
	queue     *HeapQueue
}

func NewHeapScheduler() (*heapScheduler, error) {
	queue, err := NewHeapQueue(NewJobList())
	if err != nil {
		return nil, err
	}

	scheduler, err := NewScheduler(queue)
	if err != nil {
		return nil, err
	}

	return &heapScheduler{
		scheduler: scheduler,
		queue:     queue,
	}, nil

}

func (hs *heapScheduler) AddJob(job Job) {
	hs.queue.AddJob(job)
}

func (hs *heapScheduler) Start() {
	hs.scheduler.Start()
}

func (hs *heapScheduler) Stop() {
	hs.scheduler.Stop()
}
