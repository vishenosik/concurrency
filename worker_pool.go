package concurrency

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

var (
	// worker pool is closed
	ErrPoolClosed = errors.New("worker pool is closed")
)

type internalPoolMetrics struct {
	// mu             sync.Mutex
	WorkersCurrent int32
	WorkersMin     int32
	WorkersMax     int32
	QueueDepth     int
	HighQueueDepth int
	TasksProcessed atomic.Int64
	TasksFailed    atomic.Int64
}

type PoolMetrics struct {
	WorkersCurrent int32
	WorkersMin     int32
	WorkersMax     int32
	QueueDepth     int
	HighQueueDepth int
	TasksProcessed int64
	TasksFailed    int64
}

type Priority int

const (
	PriorityLow Priority = iota
	PriorityHigh
)

func (pr Priority) String() string {
	switch pr {
	case PriorityLow:
		return "low"
	case PriorityHigh:
		return "high"
	default:
		return "undefined"
	}
}

type WorkerStatus int

const (
	StatusIdle WorkerStatus = iota
	StatusWorking
	StatusTerminated
)

func (ws WorkerStatus) String() string {
	switch ws {
	case StatusIdle:
		return "idle"
	case StatusWorking:
		return "working"
	case StatusTerminated:
		return "terminated"
	default:
		return "undefined"
	}
}

type WorkerStats struct {
	ID              int32
	StartTime       time.Time
	LastActivity    time.Time
	TasksHandled    int64
	Status          WorkerStatus
	CurrentTask     string // task description or ID
	PanicsRecovered int64
}

type Func = func()

type Task struct {
	ID       string
	Func     Func
	Priority Priority
}

type Pool struct {
	// Workers control
	numWorkers  atomic.Int32
	mu          sync.RWMutex // protects min/max workers
	minWorkers  int32
	maxWorkers  int32
	initWorkers int32

	// Scaling timeouts
	upTimeout   time.Duration
	downTimeout time.Duration
	taskTimeout time.Duration

	// Channels
	bufferCH_0 chan Task
	bufferCH_1 chan Task
	addCH      chan struct{}
	closed     atomic.Bool

	// Context
	ctx       context.Context
	cancelCtx context.CancelFunc

	// Metrics
	metrics internalPoolMetrics

	workerStats sync.Map // map[int32]*WorkerStats

	// Synchronization
	wg sync.WaitGroup
}

type PoolOption func(*Pool)

func NewWorkerPool(opts ...PoolOption) *Pool {
	return NewWorkerPoolContext(context.Background(), opts...)
}

func NewWorkerPoolContext(ctx context.Context, opts ...PoolOption) *Pool {

	p := &Pool{
		minWorkers:  3,
		maxWorkers:  256,
		initWorkers: 10,
		upTimeout:   time.Millisecond * 20,
		downTimeout: time.Millisecond * 100,
		taskTimeout: time.Second * 15,
		addCH:       make(chan struct{}, 1024),
		bufferCH_0:  make(chan Task, 1024),
		bufferCH_1:  make(chan Task, 1024),
	}

	p.ctx, p.cancelCtx = context.WithCancel(ctx)

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func WithWorkersControl[Int constraints.Signed](min, max, init Int) PoolOption {
	if min > max {
		min = max
	}
	if init < min {
		init = min
	} else if init > max {
		init = max
	}
	return func(p *Pool) {
		p.minWorkers = int32(min)
		p.maxWorkers = int32(max)
		p.initWorkers = int32(init)
	}
}

func WithTimeouts(up, down time.Duration) PoolOption {
	return func(p *Pool) {
		if up != 0 {
			p.upTimeout = up
		}
		if down != 0 {
			p.downTimeout = down
		}
	}
}

func (p *Pool) Start(_ context.Context) error {
	for range p.initWorkers {
		p.addWorker()
	}

	go func() {
		defer func() {
			close(p.addCH)
		}()
		for {
			if p.closed.Load() {
				return
			}
			select {
			case <-p.ctx.Done():
				return
			case <-p.addCH:
			case <-time.After(p.upTimeout):
				p.addWorker()
			}
		}
	}()

	go func() {
		p.wg.Wait()
		p.close()
		close(p.bufferCH_0)
	}()
	return nil
}

func (p *Pool) close() {
	if p.closed.Swap(true) {
		return
	}

	p.cancelCtx()
}

func (p *Pool) Stop(_ context.Context) error {
	p.close()
	return nil
}

func (p *Pool) AddTask(task Task) (string, error) {
	if p.closed.Load() {
		return "", ErrPoolClosed
	}

	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d_%s", time.Now().UnixNano(), task.Priority)
	}

	_select := func(ch chan Task) (string, error) {
		select {
		case ch <- task:
			return task.ID, nil
		case <-p.ctx.Done():
			return "", p.ctx.Err()
		}
	}

	switch task.Priority {
	case PriorityLow:
		return _select(p.bufferCH_0)
	case PriorityHigh:
		return _select(p.bufferCH_1)
	default:
		return _select(p.bufferCH_0)
	}
}

func (p *Pool) addWorker() {
	if !p.canScaleUp() {
		return
	}

	p.wg.Add(1)
	workerID := p.numWorkers.Add(1)

	// Initialize worker stats
	stats := &WorkerStats{
		ID:        workerID,
		StartTime: time.Now(),
	}
	p.workerStats.Store(workerID, stats)

	go func() {
		defer p.wg.Done()
		defer p.numWorkers.Add(-1)
		defer func() {
			if stats, ok := p.workerStats.Load(workerID); ok {
				ws := stats.(*WorkerStats)
				ws.Status = StatusTerminated
				ws.LastActivity = time.Now()
			}
		}()

		p.worker(workerID)
	}()
}

func (p *Pool) worker(id int32) {

	stats, _ := p.workerStats.Load(id)
	ws := stats.(*WorkerStats)

	timer := time.NewTimer(p.downTimeout)
	defer timer.Stop()

	for {
		ws.Status = StatusIdle
		ws.LastActivity = time.Now()
		ws.CurrentTask = ""

		select {
		// Check high priority first
		case task, ok := <-p.bufferCH_1:
			if !ok {
				return
			}
			p.executeTask(ws, task, timer)

		// Then normal priority
		case task, ok := <-p.bufferCH_0:
			if !ok {
				return
			}
			p.executeTask(ws, task, timer)

		case <-p.ctx.Done():
			return

		case <-timer.C:
			if p.canScaleDown() {
				return
			}
			timer.Reset(p.downTimeout)
		}
	}
}

func (p *Pool) executeTask(ws *WorkerStats, task Task, timer *time.Timer) {

	ws.Status = StatusWorking
	ws.CurrentTask = task.ID
	ws.LastActivity = time.Now()

	// Execute task with panic recovery

	defer func() {
		if r := recover(); r != nil {
			log.Printf("worker %d: task panicked: %v", ws.ID, r)
			p.metrics.TasksFailed.Add(1)
		}
	}()
	task.Func()
	p.metrics.TasksProcessed.Add(1)

	// Reset timer after successful task execution
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(p.downTimeout)
}

// Update canScaleUp/Down to use RLock
func (p *Pool) canScaleUp() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.numWorkers.Load() < p.maxWorkers
}

func (p *Pool) canScaleDown() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.numWorkers.Load() > p.minWorkers
}

// New methods for dynamic adjustment
func (p *Pool) SetMinWorkers(n int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 1 || n > p.maxWorkers {
		return
	}
	p.minWorkers = n
}

func (p *Pool) SetMaxWorkers(n int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < p.minWorkers {
		return
	}
	p.maxWorkers = n
}

func (p *Pool) GetMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolMetrics{
		WorkersCurrent: p.numWorkers.Load(),
		WorkersMin:     p.minWorkers,
		WorkersMax:     p.maxWorkers,
		QueueDepth:     len(p.bufferCH_0),
		HighQueueDepth: len(p.bufferCH_1),
		TasksProcessed: p.metrics.TasksProcessed.Load(),
		TasksFailed:    p.metrics.TasksFailed.Load(),
	}
}
