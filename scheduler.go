package concurrency

import (
	"container/heap"
	"log"
	"sync"
	"time"
)

type Job struct {
	ID       int
	Interval time.Duration
	LastRun  time.Time
	NextRun  time.Time
	Job      func()
}

type Queue []*Job

func (tq Queue) Len() int           { return len(tq) }
func (tq Queue) Less(i, j int) bool { return tq[i].NextRun.Before(tq[j].NextRun) }
func (tq Queue) Swap(i, j int)      { tq[i], tq[j] = tq[j], tq[i] }

func (tq *Queue) Push(x any) {
	if v, ok := x.(*Job); ok {
		*tq = append(*tq, v)
	}
}

func (tq *Queue) Pop() any {
	old := *tq
	n := len(old) - 1
	item := old[n]
	*tq = old[0:n]
	return item
}

type Scheduler struct {
	queue    Queue
	mu       sync.Mutex
	addChan  chan *Job
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		addChan:  make(chan *Job, 1),
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) AddTask(task *Job) {
	s.addChan <- task
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		s.mu.Lock()

		log.Println("task queue state", s.queue)

		if len(s.queue) > 0 {
			timer.Reset(time.Until(s.queue[0].NextRun))
		} else {
			timer.Reset(time.Hour) // Long duration when empty
		}
		s.mu.Unlock()

		select {
		case <-s.stopChan:
			return

		case task := <-s.addChan:
			s.mu.Lock()
			heap.Push(&s.queue, task)
			s.mu.Unlock()

		case <-timer.C:
			s.runJob()
		}
	}
}

func (s *Scheduler) runJob() {
	s.mu.Lock()
	if len(s.queue) == 0 {
		s.mu.Unlock()
		return
	}

	currentTask := heap.Pop(&s.queue).(*Job)
	s.mu.Unlock()

	now := time.Now()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job %d panicked: %v\n", currentTask.ID, r)
		}
	}()
	currentTask.Job()

	log.Printf("Job %d rescheduled\n", currentTask.ID)

	// Reschedule the task
	currentTask.LastRun = now
	currentTask.NextRun = now.Add(currentTask.Interval)
	s.mu.Lock()
	heap.Push(&s.queue, currentTask)
	s.mu.Unlock()
}
