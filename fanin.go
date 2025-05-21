package concurrency

import (
	"context"
	"iter"
	"log"
	"sync"
)

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func MergeChannels[Type any, Uint Unsigned](
	ctx context.Context,
	bufsize Uint,
	channels ...chan Type,
) <-chan Type {

	if bufsize <= 0 {
		bufsize = 1
	}
	res := make(chan Type, bufsize)

	wg := sync.WaitGroup{}
	wg.Add(len(channels))

	for _, ch := range channels {
		go func() {
			defer wg.Done()
			merger(ctx, ch, res)
		}()
	}

	go func() {
		wg.Wait()
		close(res)
	}()

	return res
}

func MergeChannelsIter[Type any, Uint Unsigned](
	ctx context.Context,
	bufsize Uint,
	channels ...chan Type,
) iter.Seq[Type] {

	return func(yield func(Type) bool) {
		for r := range MergeChannels(ctx, bufsize, channels...) {
			if !yield(r) {
				return
			}
		}
	}
}

func merger[Type any](ctx context.Context, in, out chan Type) {
	for {
		select {
		case <-ctx.Done():
			log.Println(ctx.Err())
			return
		case val, ok := <-in:
			if !ok {
				return
			}
			out <- val
		}
	}
}
