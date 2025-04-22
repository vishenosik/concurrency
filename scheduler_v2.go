package concurrency

// import (
// 	"log"
// 	"sync"
// 	"time"
// )

// type Queue interface {
// 	Pop() (Executor, bool)
// 	Push(task Executor)
// 	Reschedule(task Executor)
// 	Len() int
// 	Next() Executor
// }

// type Executor interface {
// 	Execute()
// 	NextRun() time.Time
// }

// type Scheduler struct {
// 	mu    sync.Mutex
// 	queue Queue

// 	addCH  chan Executor
// 	stopCH chan struct{}

// 	wg sync.WaitGroup
// }

// func NewScheduler() *Scheduler {
// 	return &Scheduler{
// 		addCH:  make(chan Executor, 1),
// 		stopCH: make(chan struct{}),
// 	}
// }

// func (s *Scheduler) Start() {
// 	s.wg.Add(1)
// 	go s.run()
// }

// func (s *Scheduler) Stop() {
// 	close(s.stopCH)
// 	s.wg.Wait()
// }

// func (s *Scheduler) AddJob(task Executor) {
// 	s.addCH <- task
// }

// func (s *Scheduler) run() {
// 	defer s.wg.Done()

// 	timer := time.NewTimer(0)
// 	if !timer.Stop() {
// 		<-timer.C
// 	}
// 	defer timer.Stop()

// 	for {
// 		s.mu.Lock()

// 		log.Println("task queue state", s.queue)

// 		if s.queue.Len() > 0 {
// 			timer.Reset(time.Until(s.queue.Next().NextRun()))
// 		} else {
// 			timer.Reset(time.Hour) // Long duration when empty
// 		}
// 		s.mu.Unlock()

// 		select {
// 		case <-s.stopCH:
// 			return

// 		case task := <-s.addCH:
// 			s.queue.Push(task)

// 		case <-timer.C:
// 			s.runJob()
// 		}
// 	}
// }

// func (s *Scheduler) runJob() {

// 	currentTask, ok := s.queue.Pop()
// 	if !ok {
// 		return
// 	}

// 	defer func() {
// 		if r := recover(); r != nil {
// 			log.Printf("Job panicked: %v\n", r)
// 		}
// 	}()
// 	currentTask.Execute()

// 	s.queue.Reschedule(currentTask)
// }

// type Job struct {
// 	ID       int
// 	interval time.Duration
// 	lastRun  time.Time
// 	nextRun  time.Time
// 	job      func()
// }

// type HeapQueue struct {
// 	mu    sync.Mutex
// 	queue []*Job
// }

// func (hq HeapQueue) Len() int           { return len(hq.queue) }
// func (hq HeapQueue) Less(i, j int) bool { return hq.queue[i].nextRun.Before(hq.queue[j].nextRun) }
// func (hq HeapQueue) Swap(i, j int)      { hq.queue[i], hq.queue[j] = hq.queue[j], hq.queue[i] }

// func (hq *HeapQueue) Push(x any) {
// 	if v, ok := x.(*Job); ok {
// 		hq.queue = append(hq.queue, v)
// 	}
// }

// func (hq *HeapQueue) Pop() any {
// 	old := hq
// 	n := len(old) - 1
// 	item := old[n]
// 	*hq = old[0:n]
// 	return item
// }
