package concurrency

import (
	"log"
	"sync"
	"time"
)

type QueueManager interface {
	Next() (time.Time, bool)
	Run()
	Add(job Job)
}

type Scheduler struct {
	queue       QueueManager
	idleTimeout time.Duration
	addCH       chan struct{}
	stopCH      chan struct{}
	wg          sync.WaitGroup
}

type SchedulerOption func(*Scheduler)

func defaultScheduler() *Scheduler {
	return &Scheduler{
		queue:       MustInitHeapQueue(),
		idleTimeout: time.Hour,
		stopCH:      make(chan struct{}),
		addCH:       make(chan struct{}),
	}
}

func MustInitScheduler(opts ...SchedulerOption) *Scheduler {
	scheduler, err := NewScheduler(opts...)
	if err != nil {
		panic(err)
	}
	return scheduler
}

func NewScheduler(opts ...SchedulerOption) (*Scheduler, error) {
	scheduler := defaultScheduler()
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
	close(s.addCH)
	s.wg.Wait()
}

func (s *Scheduler) Add(job Job) {
	s.queue.Add(job)
	s.addCH <- struct{}{}
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
		case <-s.addCH:
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
	s.queue.Run()
}

func WithQueueManager(QueueManager QueueManager) SchedulerOption {
	return func(s *Scheduler) {
		if QueueManager != nil {
			s.queue = QueueManager
		}
	}
}
