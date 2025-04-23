package concurrency

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

type Job interface {
	NextRun() time.Time
	Run()
	Reset()
}

type JobList []Job

func NewJobList() *JobList {
	list := make(JobList, 0)
	return &list
}

func (jb JobList) Len() int           { return len(jb) }
func (jb JobList) Less(i, j int) bool { return jb[i].NextRun().Before(jb[j].NextRun()) }
func (jb JobList) Swap(i, j int)      { jb[i], jb[j] = jb[j], jb[i] }

func (jb *JobList) Push(x any) {
	if v, ok := x.(Job); ok {
		*jb = append(*jb, v)
	}
}

func (jb *JobList) Pop() any {
	old := *jb
	n := len(old) - 1
	item := old[n]
	*jb = old[0:n]
	return item
}

func (jb *JobList) NextRun() (time.Time, bool) {
	if len(*jb) <= 0 {
		return time.Time{}, false
	}
	first := *jb
	return first[0].NextRun(), true
}

type HeapQueuer interface {
	heap.Interface
	NextRun() (time.Time, bool)
}

type HeapQueue struct {
	mu    sync.Mutex
	queue HeapQueuer
	addCH chan struct{}
}

func NewHeapQueue(queue HeapQueuer) (*HeapQueue, error) {
	if queue == nil {
		return nil, errors.New("queue is nil")
	}
	return &HeapQueue{
		addCH: make(chan struct{}, 1),
		queue: queue,
	}, nil
}

func (hq *HeapQueue) AddJob(job Job) {
	hq.mu.Lock()
	heap.Push(hq.queue, job)
	hq.mu.Unlock()
	hq.addCH <- struct{}{}
}

func (hq *HeapQueue) NextRun() (time.Time, bool) {
	return hq.queue.NextRun()
}

func (s *HeapQueue) Execute() {

	s.mu.Lock()
	if s.queue.Len() == 0 {
		s.mu.Unlock()
		return
	}
	currentTask := heap.Pop(s.queue).(Job)
	s.mu.Unlock()

	go currentTask.Run()

	currentTask.Reset()

	s.mu.Lock()
	heap.Push(s.queue, currentTask)
	s.mu.Unlock()
}

func (hq *HeapQueue) AddChan() <-chan struct{} {
	return hq.addCH
}
