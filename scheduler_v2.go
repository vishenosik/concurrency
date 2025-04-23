package concurrency

import (
	"container/heap"
	"log"
	"sync"
	"time"
)

type Queuer interface {
	NextRun() (time.Time, bool)
	Execute()
	AddChan() <-chan struct{}
}

type Scheduler_v2 struct {
	queue       Queuer
	idleTimeout time.Duration
	stopCH      chan struct{}
	wg          sync.WaitGroup
}

func defaultScheduler() *Scheduler_v2 {
	return &Scheduler_v2{
		idleTimeout: time.Hour,
		stopCH:      make(chan struct{}),
	}
}

type SchedulerOption func(*Scheduler_v2)

func NewScheduler_v2(queuer Queuer, opts ...SchedulerOption) *Scheduler_v2 {
	scheduler := defaultScheduler()
	scheduler.queue = queuer
	for _, opt := range opts {
		opt(scheduler)
	}
	return scheduler
}

func (s *Scheduler_v2) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler_v2) Stop() {
	close(s.stopCH)
	s.wg.Wait()
}

func (s *Scheduler_v2) run() {
	defer s.wg.Done()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {

		next, ok := s.queue.NextRun()
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

func (s *Scheduler_v2) runJob() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job panicked: %v\n", r)
		}
	}()
	s.queue.Execute()
}

type Jobs []*Job

func (jb Jobs) Len() int           { return len(jb) }
func (jb Jobs) Less(i, j int) bool { return jb[i].NextRun.Before(jb[j].NextRun) }
func (jb Jobs) Swap(i, j int)      { jb[i], jb[j] = jb[j], jb[i] }

func (jb *Jobs) Push(x any) {
	if v, ok := x.(*Job); ok {
		*jb = append(*jb, v)
	}
}

func (jb *Jobs) Pop() any {
	old := *jb
	n := len(old) - 1
	item := old[n]
	*jb = old[0:n]
	return item
}

type HeapQueue struct {
	mu    sync.Mutex
	queue Jobs
	addCH chan struct{}
}

func NewHeapQueue() *HeapQueue {
	return &HeapQueue{
		addCH: make(chan struct{}, 1),
		queue: make(Jobs, 0),
	}
}

func (hq *HeapQueue) AddJob(job *Job) {
	hq.mu.Lock()
	heap.Push(&hq.queue, job)
	hq.mu.Unlock()
	hq.addCH <- struct{}{}
}

func (hq *HeapQueue) NextRun() (time.Time, bool) {
	if len(hq.queue) <= 0 {
		return time.Time{}, false
	}
	return hq.queue[0].NextRun, true
}

func (s *HeapQueue) Execute() {

	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		return
	}
	currentTask := heap.Pop(&s.queue).(*Job)
	s.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Job %d panicked: %v\n", currentTask.ID, r)
			}
		}()
		currentTask.Job()
	}()

	log.Printf("Job %d rescheduled\n", currentTask.ID)

	now := time.Now()
	// Reschedule the task
	currentTask.LastRun = now
	currentTask.NextRun = now.Add(currentTask.Interval)
	s.mu.Lock()
	heap.Push(&s.queue, currentTask)
	s.mu.Unlock()
}

func (hq *HeapQueue) AddChan() <-chan struct{} {
	return hq.addCH
}
