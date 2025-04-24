package concurrency

import (
	"container/heap"
	"sync"
	"time"
)

type Job interface {
	Next() time.Time
	Run()
	Reset()
}

type JobList []Job

func MustInitJobList() *JobList {
	list, err := NewJobList()
	if err != nil {
		panic(err)
	}
	return list
}

func NewJobList() (*JobList, error) {
	list := make(JobList, 0)
	return &list, nil
}

func (jb JobList) Len() int           { return len(jb) }
func (jb JobList) Less(i, j int) bool { return jb[i].Next().Before(jb[j].Next()) }
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

func (jb *JobList) Next() (time.Time, bool) {
	if len(*jb) <= 0 {
		return time.Time{}, false
	}
	first := *jb
	return first[0].Next(), true
}

type HeapQueueManager interface {
	heap.Interface
	Next() (time.Time, bool)
}

type HeapQueue struct {
	mu    sync.Mutex
	queue HeapQueueManager
}

func defaultHeapQueue() *HeapQueue {
	return &HeapQueue{
		queue: MustInitJobList(),
	}
}

func MustInitHeapQueue() *HeapQueue {
	hq, err := NewHeapQueue()
	if err != nil {
		panic(err)
	}
	return hq
}

func NewHeapQueue() (*HeapQueue, error) {
	queue := defaultHeapQueue()
	return queue, nil
}

func (hq *HeapQueue) Add(job Job) {
	hq.mu.Lock()
	heap.Push(hq.queue, job)
	hq.mu.Unlock()
}

func (hq *HeapQueue) Next() (time.Time, bool) {
	return hq.queue.Next()
}

func (s *HeapQueue) Run() {

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
