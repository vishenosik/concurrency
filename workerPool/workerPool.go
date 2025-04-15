package concurrent

import (
	"context"
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

type Task func()

type Pool struct {
	// workers control
	numWorkers  atomic.Int32
	minWorkers  int32
	maxWorkers  int32
	initWorkers int32

	// scaling timeouts
	upTimeout   time.Duration
	downTimeout time.Duration
	taskTimeout time.Duration

	// channels
	bufferCH chan Task
	taskCH   chan Task
	closed   atomic.Bool

	// context
	ctx       context.Context
	cancelCtx context.CancelFunc

	// synchronization
	wg sync.WaitGroup
}

type PoolOption func(*Pool)

func defaultWP() *Pool {
	return &Pool{
		minWorkers:  3,
		maxWorkers:  256,
		initWorkers: 10,
		upTimeout:   time.Millisecond * 20,
		downTimeout: time.Millisecond * 100,
		taskTimeout: time.Second * 15,
		taskCH:      make(chan Task),
		bufferCH:    make(chan Task, 1024),
	}
}

func NewWorkerPool(opts ...PoolOption) *Pool {
	return NewWorkerPoolContext(context.Background(), opts...)
}

func NewWorkerPoolContext(ctx context.Context, opts ...PoolOption) *Pool {

	p := defaultWP()
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

func (p *Pool) Start() {
	for range p.initWorkers {
		p.addWorker()
	}

	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case p.taskCH <- <-p.bufferCH:
			case <-time.After(p.upTimeout):
				p.addWorker()
			}
		}
	}()

	go func() {
		p.wg.Wait()
		close(p.taskCH)
		p.close()
	}()
}

func (p *Pool) close() {
	if p.closed.Swap(true) {
		return // already closed
	}

	p.cancelCtx()
}

func (p *Pool) Stop() {
	p.close()
}

func (p *Pool) AddTask(task Task) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.bufferCH <- task:
		return nil
	}

}

func (p *Pool) addWorker() {
	if !p.canScaleUp() {
		return
	}

	p.wg.Add(1)
	p.numWorkers.Add(1)

	go func() {
		defer p.wg.Done()
		defer p.numWorkers.Add(-1)

		p.worker(p.numWorkers.Load())
	}()
}

func (p *Pool) worker(id int32) {
	log.Printf("worker %d started", id)
	defer log.Printf("worker %d stopped", id)

	timer := time.NewTimer(p.downTimeout)
	defer timer.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-timer.C:
			if p.canScaleDown() {
				return
			}
			timer.Reset(p.downTimeout)
		case task, ok := <-p.taskCH:
			if !ok {
				return
			}

			// Execute task with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("worker %d: task panicked: %v", id, r)
					}
				}()
				task()
			}()

			// Reset timer after successful task execution
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(p.downTimeout)
		}
	}
}

func (p *Pool) canScaleUp() bool {
	return p.numWorkers.Load() < p.maxWorkers
}

func (p *Pool) canScaleDown() bool {
	return p.numWorkers.Load() > p.minWorkers
}
