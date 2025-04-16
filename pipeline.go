package concurrency

import (
	"context"
	"iter"
	"log"
	"sync"
)

type Pipeline[Type any] interface {
	Fetch(context.Context) iter.Seq[Type]
	// Commit(Type)
}

type pipeline[Type any] struct {
	// Initial
	pipline Pipeline[Type]
	// Steps
	commit func(Type)
	steps  []handlerFunc[Type]
	// Context
	ctx    context.Context
	cancel context.CancelFunc
	// Pipeline concurrency control
	goroutinesLimit int
	done            chan struct{}
	running         bool
	// Panic recovery
	recoverPanics bool
	recoverer     Recoverer
}

type handlerFunc[Type any] func(in Type) (out Type, ok bool)

func NewPipeline[Type any](pipline Pipeline[Type]) *pipeline[Type] {
	return NewPipelineContext(pipline, context.Background())
}

func NewPipelineContext[Type any](pipline Pipeline[Type], ctx context.Context) *pipeline[Type] {
	pln := defaultPipline[Type]()
	pln.pipline = pipline
	pln.ctx, pln.cancel = context.WithCancel(ctx)
	return pln
}

func defaultPipline[Type any]() *pipeline[Type] {
	return &pipeline[Type]{
		commit:          func(_ Type) {},
		goroutinesLimit: 1,
		recoverPanics:   true,
		recoverer:       &noopRecoveryHandler{},
		done:            make(chan struct{}),
	}
}

func (pln *pipeline[Type]) Run() {
	pln.running = true
	go pln.run()
	pln.stop(false)
}

func (pln *pipeline[Type]) run() {

	pln.steps = append(pln.steps, func(in Type) (out Type, ok bool) {
		pln.commit(in)
		return in, true
	})
	data := pln.fetchData()
	for _, stepHandler := range pln.steps {
		data = pln.step(data, stepHandler)
	}

	for {
		select {
		case _, ok := <-data:
			if !ok {
				pln.done <- struct{}{}
				close(pln.done)
				return
			}
		}
	}
}

func (pln *pipeline[Type]) GracefulStop() {
	pln.stop(true)
}

func (pln *pipeline[Type]) stop(graceful bool) bool {
	if !pln.running {
		return true
	}
	if graceful {
		pln.cancel()
	}
	<-pln.done
	pln.running = false
	pln.cancel()
	return true
}

func (pln *pipeline[Type]) fetchData() <-chan Type {
	out := make(chan Type, pln.goroutinesLimit)
	go func() {

		defer close(out)
		defer pln._recover()

		for fetched := range pln.pipline.Fetch(pln.ctx) {
			select {
			case <-pln.ctx.Done():
				log.Println(pln.ctx.Err())
				return
			default:
				out <- fetched
			}
		}
	}()
	return out
}

func (pln *pipeline[Type]) step(in <-chan Type, handler handlerFunc[Type]) <-chan Type {
	out := make(chan Type, pln.goroutinesLimit)
	limiter := make(chan struct{}, pln.goroutinesLimit)
	var wg sync.WaitGroup

	handle := func(in Type) {
		defer pln._recover()
		if result, ok := handler(in); ok {
			out <- result
		}
		<-limiter
		wg.Done()
	}

	go func() {
		for i := range in {
			wg.Add(1)
			limiter <- struct{}{}
			go handle(i)
		}
		wg.Wait()
		close(limiter)
		close(out)
	}()
	return out
}
